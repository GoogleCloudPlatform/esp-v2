// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <chrono>

#include "absl/strings/match.h"
#include "common/http/utility.h"
#include "src/envoy/http/service_control/handler_impl.h"
#include "src/envoy/http/service_control/handler_utils.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

using ::Envoy::StreamInfo::FilterState;
using ::espv2::api_proxy::service_control::CheckResponseInfo;
using ::espv2::api_proxy::service_control::OperationInfo;
using ::espv2::api_proxy::service_control::ScResponseErrorType;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

// The HTTP header suffix to send consumer info to backend.
constexpr char kConsumerTypeHeaderSuffix[] = "api-consumer-type";
constexpr char kConsumerNumberHeaderSuffix[] = "api-consumer-number";

// CheckRequest headers
const Envoy::Http::LowerCaseString kIosBundleIdHeader{
    "x-ios-bundle-identifier"};
const Envoy::Http::LowerCaseString kAndroidPackageHeader{"x-android-package"};
const Envoy::Http::LowerCaseString kAndroidCertHeader{"x-android-cert"};
const Envoy::Http::LowerCaseString kRefererHeader{"referer"};

constexpr char JwtPayloadIssuerPath[] = "iss";
constexpr char JwtPayloadAudiencePath[] = "aud";

ServiceControlHandlerImpl::ServiceControlHandlerImpl(
    const Envoy::Http::RequestHeaderMap& headers,
    const Envoy::StreamInfo::StreamInfo& stream_info, const std::string& uuid,
    const FilterConfigParser& cfg_parser, Envoy::TimeSource& time_source,
    ServiceControlFilterStats& filter_stats)
    : cfg_parser_(cfg_parser),
      stream_info_(stream_info),
      time_source_(time_source),
      uuid_(uuid),
      consumer_type_header_(cfg_parser_.config().generated_header_prefix() +
                            kConsumerTypeHeaderSuffix),
      consumer_number_header_(cfg_parser_.config().generated_header_prefix() +
                              kConsumerNumberHeaderSuffix),
      is_grpc_(false),
      filter_stats_(filter_stats) {
  is_grpc_ = Envoy::Grpc::Common::hasGrpcContentType(headers);

  http_method_ = std::string(utils::readHeaderEntry(headers.Method()));
  path_ = std::string(utils::readHeaderEntry(headers.Path()));

  const absl::string_view operation = utils::getStringFilterState(
      stream_info_.filterState(), utils::kFilterStateOperation);

  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    ENVOY_LOG(debug, "No operation found");
    // Extract api-key to be used for Report for non-matched requests.
    extractAPIKey(headers, cfg_parser_.default_api_keys().locations(),
                  api_key_);
    return;
  }

  require_ctx_ = cfg_parser_.find_requirement(operation);
  if (!require_ctx_) {
    ENVOY_LOG(debug, "No requirement matched!");
    // Extract api-key to be used for Report for an operation without
    // requirement.
    extractAPIKey(headers, cfg_parser_.default_api_keys().locations(),
                  api_key_);
    return;
  }

  if (require_ctx_->config().api_key().locations_size() > 0) {
    extractAPIKey(headers, require_ctx_->config().api_key().locations(),
                  api_key_);
  } else {
    extractAPIKey(headers, cfg_parser_.default_api_keys().locations(),
                  api_key_);
  }
}

ServiceControlHandlerImpl::~ServiceControlHandlerImpl() {}

void ServiceControlHandlerImpl::fillFilterState(FilterState& filter_state) {
  utils::setStringFilterState(filter_state, utils::kFilterStateApiKey,
                              api_key_);

  utils::setStringFilterState(filter_state, utils::kFilterStateApiMethod,
                              require_ctx_->config().operation_name());
}

void ServiceControlHandlerImpl::onDestroy() {
  if (cancel_fn_) {
    cancel_fn_();
    cancel_fn_ = nullptr;
  }
}

void ServiceControlHandlerImpl::fillOperationInfo(
    ::espv2::api_proxy::service_control::OperationInfo& info) {
  info.operation_id = uuid_;
  info.operation_name = require_ctx_->config().operation_name();
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();
  info.current_time = time_source_.systemTime();

  if (stream_info_.downstreamRemoteAddress()->type() ==
      Envoy::Network::Address::Type::Ip) {
    info.client_ip =
        stream_info_.downstreamRemoteAddress()->ip()->addressAsString();
  }

  info.api_key = api_key_;
}

void ServiceControlHandlerImpl::prepareReportRequest(
    ::espv2::api_proxy::service_control::ReportRequestInfo& info) {
  fillOperationInfo(info);

  info.url = path_;
  info.method = http_method_;
  info.api_method = require_ctx_->config().operation_name();
  info.api_name = require_ctx_->config().api_name();
  info.api_version = require_ctx_->config().api_version();
  info.log_message = info.api_method + " is called";

  info.check_response_info = check_response_info_;
  info.status = check_status_;

  fillGCPInfo(cfg_parser_.config(), info);
}

void ServiceControlHandlerImpl::callCheck(
    Envoy::Http::RequestHeaderMap& headers, Envoy::Tracing::Span& parent_span,
    CheckDoneCallback& callback) {
  if (!isConfigured()) {
    filter_stats_.filter_.denied_producer_error_.inc();
    callback.onCheckDone(Status(Code::NOT_FOUND, "Method does not exist."));
    return;
  }
  check_callback_ = &callback;

  if (!isCheckRequired()) {
    callQuota();
    return;
  }

  if (!hasApiKey()) {
    filter_stats_.filter_.denied_consumer_error_.inc();
    check_status_ =
        Status(Code::UNAUTHENTICATED,
               "Method doesn't allow unregistered callers (callers without "
               "established identity). Please use API Key or other form of "
               "API consumer identity to call this API.");
    callback.onCheckDone(check_status_);
    return;
  }

  // Make a check call
  ::espv2::api_proxy::service_control::CheckRequestInfo info;
  fillOperationInfo(info);

  info.ios_bundle_id =
      std::string(utils::extractHeader(headers, kIosBundleIdHeader));
  info.referer = std::string(utils::extractHeader(headers, kRefererHeader));
  info.android_package_name =
      std::string(utils::extractHeader(headers, kAndroidPackageHeader));
  info.android_cert_fingerprint =
      std::string(utils::extractHeader(headers, kAndroidCertHeader));

  on_check_done_called_ = false;
  cancel_fn_ = require_ctx_->service_ctx().call().callCheck(
      info, parent_span,
      [this, &headers](const Status& status,
                       const CheckResponseInfo& response_info) {
        cancel_fn_ = nullptr;
        on_check_done_called_ = true;
        onCheckResponse(headers, status, response_info);
      });
  if (on_check_done_called_) {
    cancel_fn_ = nullptr;
  }
}

// TODO(taoxuy): add unit test
void ServiceControlHandlerImpl::callQuota() {
  if (!isQuotaRequired()) {
    check_callback_->onCheckDone(check_status_);
    return;
  }

  ::espv2::api_proxy::service_control::QuotaRequestInfo info{
      require_ctx_->metric_costs()};
  info.method_name = require_ctx_->config().operation_name();
  fillOperationInfo(info);

  // TODO: if quota cache is disabled, need to use in-flight
  // transport, need to save its cancel function.
  // For now, quota cache is always enabled, in-flight transport
  // is not called.
  require_ctx_->service_ctx().call().callQuota(
      info, [this](const Status& status) {
        check_status_ = status;
        check_callback_->onCheckDone(status);
      });
}

void ServiceControlHandlerImpl::onCheckResponse(
    Envoy::Http::RequestHeaderMap& headers, const Status& status,
    const CheckResponseInfo& response_info) {
  check_response_info_ = response_info;

  check_status_ = status;

  // Set consumer info to backend. Since consumer_project_id is deprecated and
  // replaced by consumer_number so don't set it here.
  if (!response_info.consumer_type.empty()) {
    headers.setReferenceKey(consumer_type_header_, response_info.consumer_type);
  }

  if (!response_info.consumer_number.empty()) {
    headers.setReferenceKey(consumer_number_header_,
                            response_info.consumer_number);
  }

  if (!check_status_.ok()) {
    check_callback_->onCheckDone(check_status_);
    return;
  }

  callQuota();
}

void ServiceControlHandlerImpl::callReport(
    const Envoy::Http::RequestHeaderMap* request_headers,
    const Envoy::Http::ResponseHeaderMap* response_headers,
    const Envoy::Http::ResponseTrailerMap* response_trailers) {
  if (require_ctx_ == nullptr) {
    require_ctx_ = cfg_parser_.non_match_rqm_ctx();
  }

  if (!isReportRequired()) {
    return;
  }

  ::espv2::api_proxy::service_control::ReportRequestInfo info;
  prepareReportRequest(info);
  fillLoggedHeader(request_headers,
                   require_ctx_->service_ctx().config().log_request_headers(),
                   info.request_headers);
  fillLoggedHeader(response_headers,
                   require_ctx_->service_ctx().config().log_response_headers(),
                   info.response_headers);
  fillJwtPayloads(
      stream_info_.dynamicMetadata(),
      require_ctx_->service_ctx().config().jwt_payload_metadata_name(),
      require_ctx_->service_ctx().config().log_jwt_payloads(),
      info.jwt_payloads);

  fillJwtPayload(
      stream_info_.dynamicMetadata(),
      require_ctx_->service_ctx().config().jwt_payload_metadata_name(),
      JwtPayloadIssuerPath, info.auth_issuer);

  fillJwtPayload(
      stream_info_.dynamicMetadata(),
      require_ctx_->service_ctx().config().jwt_payload_metadata_name(),
      JwtPayloadAudiencePath, info.auth_audience);

  info.frontend_protocol = getFrontendProtocol(response_headers, stream_info_);
  info.backend_protocol =
      getBackendProtocol(require_ctx_->service_ctx().config());

  uint64_t request_header_size = 0;
  if (request_headers) {
    info.referer =
        std::string(utils::extractHeader(*request_headers, kRefererHeader));
    request_header_size = request_headers->byteSize();
  }

  fillLatency(stream_info_, info.latency, filter_stats_);

  info.response_code = stream_info_.responseCode().value_or(500);

  info.request_size = stream_info_.bytesReceived() + request_header_size;

  uint64_t response_header_size = 0;
  if (response_headers) {
    response_header_size += response_headers->byteSize();
  }
  if (response_trailers) {
    response_header_size += response_trailers->byteSize();
  }
  info.response_size = stream_info_.bytesSent() + response_header_size;

  require_ctx_->service_ctx().call().callReport(info);
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
