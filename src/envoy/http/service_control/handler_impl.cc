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

#include "src/envoy/http/service_control/handler_impl.h"

#include <chrono>

#include "absl/strings/match.h"
#include "source/common/common/empty_string.h"
#include "source/common/http/headers.h"
#include "source/common/http/utility.h"
#include "src/envoy/http/service_control/handler_utils.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

using Envoy::Http::CustomHeaders;
using Envoy::Http::CustomInlineHeaderRegistry;
using Envoy::Http::RegisterCustomInlineHeader;
using ::Envoy::StreamInfo::FilterState;
using ::espv2::api_proxy::service_control::CheckResponseInfo;
using ::espv2::api_proxy::service_control::OperationInfo;
using ::espv2::api_proxy::service_control::QuotaResponseInfo;
using ::espv2::api_proxy::service_control::ScResponseError;
using ::espv2::api_proxy::service_control::ScResponseErrorType;
using ::google::protobuf::util::OkStatus;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::StatusCode;

namespace {
RegisterCustomInlineHeader<CustomInlineHeaderRegistry::Type::RequestHeaders>
    referer_handle(CustomHeaders::get().Referer);

// The HTTP header suffix to send consumer info to backend.
constexpr char kConsumerTypeHeaderSuffix[] = "api-consumer-type";
constexpr char kConsumerNumberHeaderSuffix[] = "api-consumer-number";

// CheckRequest headers
const Envoy::Http::LowerCaseString kIosBundleIdHeader{
    "x-ios-bundle-identifier"};
const Envoy::Http::LowerCaseString kAndroidPackageHeader{"x-android-package"};
const Envoy::Http::LowerCaseString kAndroidCertHeader{"x-android-cert"};

constexpr char JwtPayloadIssuerPath[] = "iss";
constexpr char JwtPayloadAudiencePath[] = "aud";
}  // namespace

ServiceControlHandlerImpl::ServiceControlHandlerImpl(
    const Envoy::Http::RequestHeaderMap& headers,
    Envoy::Http::StreamDecoderFilterCallbacks* decoder_callbacks,
    const std::string& uuid, const FilterConfigParser& cfg_parser,
    Envoy::TimeSource& time_source, ServiceControlFilterStats& filter_stats)
    : cfg_parser_(cfg_parser),
      stream_info_(decoder_callbacks->streamInfo()),
      decoder_callbacks_(decoder_callbacks),
      time_source_(time_source),
      uuid_(uuid),
      request_header_size_(headers.byteSize()),
      consumer_type_header_(cfg_parser_.config().generated_header_prefix() +
                            kConsumerTypeHeaderSuffix),
      consumer_number_header_(cfg_parser_.config().generated_header_prefix() +
                              kConsumerNumberHeaderSuffix),
      is_grpc_(false),
      filter_stats_(filter_stats) {
  is_grpc_ = Envoy::Grpc::Common::hasGrpcContentType(headers);

  http_method_ = std::string(utils::readHeaderEntry(headers.Method()));
  path_ = std::string(utils::readHeaderEntry(headers.Path()));

  const auto operation = getOperationFromPerRoute();
  if (!operation.empty()) {
    require_ctx_ = cfg_parser_.find_requirement(operation);
    if (!require_ctx_) {
      ENVOY_LOG(debug, "No requirement matched!");
    }
  } else {
    ENVOY_LOG(debug, "No operation found");
  }
  if (require_ctx_ == nullptr) {
    ENVOY_LOG(debug, "Use non matched requirement.");
    require_ctx_ = cfg_parser_.non_match_rqm_ctx();
  }

  if (require_ctx_->config().api_key().locations_size() > 0) {
    extractAPIKey(headers, require_ctx_->config().api_key().locations(),
                  api_key_);
  } else {
    extractAPIKey(headers, cfg_parser_.default_api_keys().locations(),
                  api_key_);
  }

  if (require_ctx_->service_ctx().config().client_ip_from_forward_header()) {
    client_ip_from_forward_header_ = extractIPFromForwardHeader(headers);
  }
}

ServiceControlHandlerImpl::~ServiceControlHandlerImpl() {}

absl::string_view ServiceControlHandlerImpl::getOperationFromPerRoute() {
  const auto* per_route =
      ::Envoy::Http::Utility::resolveMostSpecificPerFilterConfig<
          PerRouteFilterConfig>(decoder_callbacks_);
  if (per_route == nullptr) {
    ENVOY_LOG(debug, "no per-route config");
    return Envoy::EMPTY_STRING;
  }
  ENVOY_LOG(debug, "get operation_name: {}", per_route->operation_name());
  return per_route->operation_name();
}

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

  if (!client_ip_from_forward_header_.empty()) {
    info.client_ip = client_ip_from_forward_header_;
  } else {
    if (stream_info_.downstreamAddressProvider().remoteAddress()->type() ==
        Envoy::Network::Address::Type::Ip) {
      info.client_ip = stream_info_.downstreamAddressProvider()
                           .remoteAddress()
                           ->ip()
                           ->addressAsString();
    }
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
  // Don't have per-route config so pass through the request, regarded as the
  // unknown method.
  if (!isConfigured()) {
    callback.onCheckDone(OkStatus(), "");
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
        Status(StatusCode::kUnauthenticated,
               "Method doesn't allow unregistered callers (callers without "
               "established identity). Please use API Key or other form of "
               "API consumer identity to call this API.");
    callback.onCheckDone(
        check_status_,
        utils::generateRcDetails(utils::kRcDetailFilterServiceControl,
                                 utils::kRcDetailErrorTypeBadRequest,
                                 utils::kRcDetailErrorMissingApiKey));
    return;
  }

  // Make a check call
  ::espv2::api_proxy::service_control::CheckRequestInfo info;
  fillOperationInfo(info);

  info.referer = std::string(
      utils::readHeaderEntry(headers.getInline(referer_handle.handle())));
  info.ios_bundle_id =
      std::string(utils::extractHeader(headers, kIosBundleIdHeader));
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
    check_callback_->onCheckDone(check_status_, rc_detail_);
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
      info,
      [this](const Status& status, const QuotaResponseInfo& response_info) {
        if (!response_info.error.name.empty()) {
          rc_detail_ = utils::generateRcDetails(
              utils::kRcDetailFilterServiceControl,
              response_info.error.is_network_error
                  ? utils::kRcDetailErrorTypeScQuotaNetwork
                  : utils::kRcDetailErrorTypeScQuota,
              response_info.error.name);
        }
        check_status_ = status;
        check_callback_->onCheckDone(status, rc_detail_);
      });
}

void ServiceControlHandlerImpl::onCheckResponse(
    Envoy::Http::RequestHeaderMap& headers, const Status& status,
    const CheckResponseInfo& response_info) {
  check_response_info_ = response_info;

  if (!response_info.error.name.empty()) {
    rc_detail_ =
        utils::generateRcDetails(utils::kRcDetailFilterServiceControl,
                                 response_info.error.is_network_error
                                     ? utils::kRcDetailErrorTypeScCheckNetwork
                                     : utils::kRcDetailErrorTypeScCheck,
                                 response_info.error.name);
  }
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
    check_callback_->onCheckDone(check_status_, rc_detail_);
    return;
  }

  callQuota();
}

void ServiceControlHandlerImpl::callReport(
    const Envoy::Http::RequestHeaderMap* request_headers,
    const Envoy::Http::ResponseHeaderMap* response_headers,
    const Envoy::Http::ResponseTrailerMap* response_trailers,
    const Envoy::Tracing::Span& parent_span) {
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

  if (request_headers) {
    info.referer = std::string(utils::readHeaderEntry(
        request_headers->getInline(referer_handle.handle())));
  }

  fillLatency(stream_info_, info.latency, filter_stats_);
  fillStatus(response_headers, response_trailers, stream_info_, info);

  info.request_size = stream_info_.bytesReceived() + request_header_size_;

  uint64_t response_header_size = 0;
  if (response_headers) {
    response_header_size += response_headers->byteSize();
  }
  if (response_trailers) {
    response_header_size += response_trailers->byteSize();
  }
  info.response_size = stream_info_.bytesSent() + response_header_size;

  info.response_code_detail = stream_info_.responseCodeDetails().value_or("");

  info.trace_id = parent_span.getTraceIdAsHex();

  require_ctx_->service_ctx().call().callReport(info);
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
