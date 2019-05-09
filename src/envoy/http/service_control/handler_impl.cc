// Copyright 2018 Google Cloud Platform Proxy Authors
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

using ::google::api::envoy::http::service_control::APIKeyLocation;
using ::google::api_proxy::service_control::CheckResponseInfo;
using ::google::api_proxy::service_control::LatencyInfo;
using ::google::api_proxy::service_control::OperationInfo;
using ::google::api_proxy::service_control::protocol::Protocol;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The HTTP header to send consumer project to backend.
const Http::LowerCaseString kConsumerProjectId("x-endpoint-api-project-id");

// CheckRequest headers
const Http::LowerCaseString kIosBundleIdHeader{"x-ios-bundle-identifier"};
const Http::LowerCaseString kAndroidPackageHeader{"x-android-package"};
const Http::LowerCaseString kAndroidCertHeader{"x-android-cert"};
const Http::LowerCaseString kRefererHeader{"referer"};

constexpr char JwtPayloadIssuerPath[] = "iss";
constexpr char JwtPayloadAuidencePath[] = "aud";

ServiceControlHandlerImpl::ServiceControlHandlerImpl(
    const Http::HeaderMap& headers, const StreamInfo::StreamInfo& stream_info,
    const std::string& uuid, const FilterConfigParser& cfg_parser,
    std::chrono::system_clock::time_point now)
    : cfg_parser_(cfg_parser),
      stream_info_(stream_info),
      uuid_(uuid),
      last_reported_(now) {
  http_method_ = std::string(Utils::getRequestHTTPMethodWithOverride(
      headers.Method()->value().getStringView(), headers));
  path_ = std::string(headers.Path()->value().getStringView());
  request_header_size_ = headers.byteSize();

  is_grpc_ = Envoy::Grpc::Common::hasGrpcContentType(headers);

  const absl::string_view operation = Utils::getStringFilterState(
      stream_info_.filterState(), Utils::kOperation);

  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    ENVOY_LOG(debug, "No operation found");
    return;
  }

  require_ctx_ = cfg_parser_.FindRequirement(operation);
  if (!require_ctx_) {
    ENVOY_LOG(debug, "No requirement matched!");
    return;
  }

  if (!isCheckRequired()) {
    ENVOY_LOG(debug, "Service control check is not needed");
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

ServiceControlHandlerImpl::~ServiceControlHandlerImpl() {
  if (aborted_) {
    *aborted_ = true;
  }
}

void ServiceControlHandlerImpl::fillOperationInfo(
    ::google::api_proxy::service_control::OperationInfo& info,
    std::chrono::system_clock::time_point now) {
  info.operation_id = uuid_;
  info.operation_name = require_ctx_->config().operation_name();
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();
  info.request_start_time = now;
}

void ServiceControlHandlerImpl::prepareReportRequest(
    ::google::api_proxy::service_control::ReportRequestInfo& info) {
  fillOperationInfo(info);

  // Check and Report has different rule to send api-key
  if (check_response_info_.is_api_key_valid &&
      check_response_info_.service_is_activated) {
    info.api_key = api_key_;
  }

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

void ServiceControlHandlerImpl::callCheck(Http::HeaderMap& headers,
                                          CheckDoneCallback& callback) {
  if (!isConfigured()) {
    callback.onCheckDone(Status(Code::NOT_FOUND, "Method does not exist."));
    return;
  }

  if (!isCheckRequired()) {
    callback.onCheckDone(Status::OK);
    return;
  }

  if (!hasApiKey()) {
    check_status_ =
        Status(Code::UNAUTHENTICATED,
               "Method doesn't allow unregistered callers (callers without "
               "established identity). Please use API Key or other form of "
               "API consumer identity to call this API.");
    callback.onCheckDone(check_status_);
    return;
  }

  check_callback_ = &callback;

  // Make a check call
  ::google::api_proxy::service_control::CheckRequestInfo info;
  fillOperationInfo(info);

  // Check and Report has different rule to send api-key
  info.api_key = api_key_;

  info.ios_bundle_id =
      std::string(Utils::extractHeader(headers, kIosBundleIdHeader));
  info.referer = std::string(Utils::extractHeader(headers, kRefererHeader));
  info.android_package_name =
      std::string(Utils::extractHeader(headers, kAndroidPackageHeader));
  info.android_cert_fingerprint =
      std::string(Utils::extractHeader(headers, kAndroidCertHeader));

  info.client_ip =
      stream_info_.downstreamRemoteAddress()->ip()->addressAsString();

  aborted_.reset(new bool(false));
  require_ctx_->service_ctx().call().callCheck(
      info, [this, aborted = aborted_, &headers](
                const Status& status, const CheckResponseInfo& response_info) {
        if (*aborted) return;
        onCheckResponse(headers, status, response_info);
      });
}

void ServiceControlHandlerImpl::onCheckResponse(
    Http::HeaderMap& headers, const Status& status,
    const CheckResponseInfo& response_info) {
  check_response_info_ = response_info;

  check_status_ = status;

  // Set consumer project_id to backend.
  if (!response_info.consumer_project_id.empty()) {
    headers.setReferenceKey(kConsumerProjectId,
                            response_info.consumer_project_id);
  }

  check_callback_->onCheckDone(check_status_);
}

void ServiceControlHandlerImpl::callReport(
    const Http::HeaderMap* request_headers,
    const Http::HeaderMap* response_headers,
    const Http::HeaderMap* response_trailers) {
  if (!isConfigured()) {
    return;
  }

  ::google::api_proxy::service_control::ReportRequestInfo info;
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
      JwtPayloadAuidencePath, info.auth_audience);

  info.frontend_protocol = getFrontendProtocol(response_headers, stream_info_);

  info.backend_protocol =
      getBackendProtocol(require_ctx_->service_ctx().config());

  if (request_headers) {
    info.referer =
        std::string(Utils::extractHeader(*request_headers, kRefererHeader));
  }

  fillLatency(stream_info_, info.latency);

  info.response_code = stream_info_.responseCode().value_or(500);

  info.request_size = stream_info_.bytesReceived() + request_header_size_;
  info.request_bytes = stream_info_.bytesReceived() + request_header_size_;

  uint64_t response_header_size = 0;
  if (response_headers) {
    response_header_size += response_headers->byteSize();
  }
  if (response_trailers) {
    response_header_size += response_trailers->byteSize();
  }
  info.response_size = stream_info_.bytesSent() + response_header_size;
  info.response_bytes = stream_info_.bytesSent() + response_header_size;

  require_ctx_->service_ctx().call().callReport(info);
}

void ServiceControlHandlerImpl::collectDecodeData(
    Buffer::Instance& request_data, std::chrono::system_clock::time_point now) {
  if (!is_grpc_) {
    return;
  }

  Envoy::Utils::IncrementMessageCounter(request_data, &grpc_request_counter_);
  streaming_info_.request_message_count = grpc_request_counter_.count;
  streaming_info_.request_bytes += request_data.length();

  tryIntermediateReport(now);
}

void ServiceControlHandlerImpl::collectEncodeData(
    Buffer::Instance& response_data,
    std::chrono::system_clock::time_point now) {
  if (!is_grpc_) {
    return;
  }

  Envoy::Utils::IncrementMessageCounter(response_data, &grpc_response_counter_);
  streaming_info_.response_message_count = grpc_response_counter_.count;
  streaming_info_.response_bytes += response_data.length();

  tryIntermediateReport(now);
}

void ServiceControlHandlerImpl::tryIntermediateReport(
    std::chrono::system_clock::time_point now) {
  if (streaming_info_.is_first_report) {
    streaming_info_.start_time = now;
  }
  // Avoid reporting more frequently than the configured interval.
  if (std::chrono::duration_cast<std::chrono::milliseconds>(now -
                                                            last_reported_)
          .count() <
      require_ctx_->service_ctx().get_min_stream_report_interval_ms()) {
    return;
  }

  ::google::api_proxy::service_control::ReportRequestInfo info;
  prepareReportRequest(info);

  info.request_bytes = streaming_info_.request_bytes;
  info.response_bytes = streaming_info_.response_bytes;
  info.streaming_request_message_counts = streaming_info_.request_message_count;
  info.streaming_response_message_counts =
      streaming_info_.response_message_count;

  info.streaming_durations =
      std::chrono::duration_cast<std::chrono::microseconds>(
          now - streaming_info_.start_time)
          .count();
  info.is_first_report = streaming_info_.is_first_report;
  info.is_final_report = false;

  require_ctx_->service_ctx().call().callReport(info);
  last_reported_ = now;
  streaming_info_.is_first_report = false;
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
