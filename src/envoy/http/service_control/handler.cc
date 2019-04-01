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

#include "src/envoy/http/service_control/handler.h"
#include "absl/strings/match.h"
#include "common/http/utility.h"

using ::google::api::envoy::http::service_control::APIKeyLocation;
using ::google::api_proxy::service_control::CheckResponseInfo;
using ::google::api_proxy::service_control::LatencyInfo;
using ::google::api_proxy::service_control::OperationInfo;
using ::google::api_proxy::service_control::protocol::Protocol;
using ::google::protobuf::util::Status;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

const std::string kContentTypeApplicationGrpcPrefix = "application/grpc";

// The HTTP header to send consumer project to backend.
const Http::LowerCaseString kConsumerProjectId("x-endpoint-api-project-id");

// CheckRequest headers
const Http::LowerCaseString kIosBundleIdHeader{"x-ios-bundle-identifier"};
const Http::LowerCaseString kAndroidPackageHeader{"x-android-package"};
const Http::LowerCaseString kAndroidCertHeader{"x-android-cert"};
const Http::LowerCaseString kRefererHeader{"referer"};
const Http::LowerCaseString kContentTypeHeader{"content-type"};

inline int64_t convertNsToMs(std::chrono::nanoseconds ns) {
  return std::chrono::duration_cast<std::chrono::milliseconds>(ns).count();
}

void fillLatency(const StreamInfo::StreamInfo& stream_info,
                 LatencyInfo& latency) {
  if (stream_info.requestComplete()) {
    latency.request_time_ms =
        convertNsToMs(stream_info.requestComplete().value());
  }

  auto start = stream_info.firstUpstreamTxByteSent();
  auto end = stream_info.lastUpstreamRxByteReceived();
  if (start && end && end.value() >= start.value()) {
    latency.backend_time_ms = convertNsToMs(end.value() - start.value());
  }

  if (latency.backend_time_ms >= 0 &&
      latency.request_time_ms >= latency.backend_time_ms) {
    latency.overhead_time_ms =
        latency.request_time_ms - latency.backend_time_ms;
  }
}

std::string extractHeader(const Envoy::Http::HeaderMap& headers,
                          const Envoy::Http::LowerCaseString& header) {
  auto* entry = headers.get(header);
  if (entry) {
    return entry->value().c_str();
  }
  return "";
}

bool isGrpcRequest(const std::string& content_type) {
  // Formally defined as:
  // `application/grpc(-web(-text))[+proto/+json/+thrift/{custom}]`
  //
  // The worst case is `application/grpc{custom}`. Just check the beginning.
  return absl::StartsWith(content_type, kContentTypeApplicationGrpcPrefix);
}

Protocol getFrontendProtocol(const std::string& content_type, bool http) {
  if (isGrpcRequest(content_type)) {
    return Protocol::GRPC;
  }

  if (!http) {
    return Protocol::UNKNOWN;
  }

  // TODO(toddbeckman) figure out HTTPS
  return Protocol::HTTP;
}

Protocol getBackendProtocol(const std::string& protocol) {
  if (protocol == "http1" || protocol == "http2") {
    return Protocol::HTTP;
  }

  if (protocol == "grpc") {
    return Protocol::GRPC;
  }

  return Protocol::UNKNOWN;
}

}  // namespace

Handler::Handler(const Http::HeaderMap& headers, const std::string& operation,
                 FilterConfigSharedPtr config)
    : config_(config) {
  http_method_ = headers.Method()->value().c_str();
  path_ = headers.Path()->value().c_str();
  require_ctx_ = config_->cfg_parser().FindRequirement(operation);
  if (!require_ctx_) {
    ENVOY_LOG(debug, "No requirement matched!");
    return;
  }

  // This uuid is shared for Check and report
  uuid_ = config_->random().uuid();

  if (!isCheckRequired()) {
    ENVOY_LOG(debug, "Service control check is not needed");
    return;
  }

  if (require_ctx_->config().api_key().locations_size() > 0) {
    extractAPIKey(headers, require_ctx_->config().api_key().locations());
  } else {
    extractAPIKey(headers, config_->default_api_keys().locations());
  }
}

Handler::~Handler() {
  if (aborted_) {
    *aborted_ = true;
  }
}

bool Handler::extractAPIKeyFromQuery(const Http::HeaderMap& headers,
                                     const std::string& query) {
  if (!params_parsed_) {
    parsed_params_ =
        Http::Utility::parseQueryString(headers.Path()->value().c_str());
    params_parsed_ = true;
  }

  const auto& it = parsed_params_.find(query);
  if (it != parsed_params_.end()) {
    api_key_ = it->second;
    ENVOY_LOG(debug, "api-key: {} from query: {}", api_key_, query);
    return true;
  }
  return false;
}

bool Handler::extractAPIKeyFromHeader(const Http::HeaderMap& headers,
                                      const std::string& header) {
  // TODO(qiwzhang): optimize this by using LowerCaseString at init.
  auto* entry = headers.get(Http::LowerCaseString(header));
  if (entry) {
    api_key_ = std::string(entry->value().c_str(), entry->value().size());
    ENVOY_LOG(debug, "api-key: {} from header: {}", api_key_, header);
    return true;
  }
  return false;
}

bool Handler::extractAPIKeyFromCookie(const Http::HeaderMap& headers,
                                      const std::string& cookie) {
  std::string api_key = Http::Utility::parseCookieValue(headers, cookie);
  if (!api_key.empty()) {
    api_key_ = api_key;
    ENVOY_LOG(debug, "api-key: {} from cookie: {}", api_key_, cookie);
    return true;
  }
  return false;
}

bool Handler::extractAPIKey(
    const Http::HeaderMap& headers,
    const ::google::protobuf::RepeatedPtrField<
        ::google::api::envoy::http::service_control::APIKeyLocation>&
        locations) {
  for (const auto& location : locations) {
    switch (location.key_case()) {
      case APIKeyLocation::kQuery:
        if (extractAPIKeyFromQuery(headers, location.query())) return true;
        break;
      case APIKeyLocation::kHeader:
        if (extractAPIKeyFromHeader(headers, location.header())) return true;
        break;
      case APIKeyLocation::kCookie:
        if (extractAPIKeyFromCookie(headers, location.cookie())) return true;
        break;
      case APIKeyLocation::KEY_NOT_SET:
        break;
    }
  }
  return false;
}

void Handler::fillOperationInfo(
    ::google::api_proxy::service_control::OperationInfo& info) {
  info.operation_id = uuid_;
  info.operation_name = require_ctx_->config().operation_name();
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();
  info.request_start_time = std::chrono::system_clock::now();
}

void Handler::fillGCPInfo(
    ::google::api_proxy::service_control::ReportRequestInfo& info) {
  const auto& filter_config = config_->config();
  if (!filter_config.has_gcp_attributes()) {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::UNKNOWN;
    return;
  }

  const auto& gcp_attributes = filter_config.gcp_attributes();
  if (!gcp_attributes.zone().empty()) {
    info.location = gcp_attributes.zone();
  }

  const std::string& platform = gcp_attributes.platform();
  if (platform == "GAE_FLEX") {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::GAE_FLEX;
  } else if (platform == "GKE") {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::GKE;
  } else if (platform == "GCE") {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::GCE;
  } else {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::UNKNOWN;
  }
}

void Handler::callCheck(Http::HeaderMap& headers, CheckDoneCallback& callback,
                        const StreamInfo::StreamInfo& stream_info) {
  check_callback_ = &callback;

  // Make a check call
  ::google::api_proxy::service_control::CheckRequestInfo info;
  fillOperationInfo(info);

  // Check and Report has different rule to send api-key
  info.api_key = api_key_;

  info.ios_bundle_id = extractHeader(headers, kIosBundleIdHeader);
  info.referer = extractHeader(headers, kRefererHeader);
  info.android_package_name = extractHeader(headers, kAndroidPackageHeader);
  info.android_cert_fingerprint = extractHeader(headers, kAndroidCertHeader);

  info.client_ip =
      stream_info.downstreamRemoteAddress()->ip()->addressAsString();

  ::google::api::servicecontrol::v1::CheckRequest check_request;
  require_ctx_->service_ctx().builder().FillCheckRequest(info, &check_request);
  ENVOY_LOG(debug, "Sending check : {}", check_request.DebugString());

  aborted_.reset(new bool(false));
  require_ctx_->service_ctx().getTLCache().client_cache().callCheck(
      check_request,
      [this, aborted = aborted_, &headers](
          const Status& status, const CheckResponseInfo& response_info) {
        if (*aborted) return;
        onCheckResponse(headers, status, response_info);
      });
}

void Handler::onCheckResponse(Http::HeaderMap& headers, const Status& status,
                              const CheckResponseInfo& response_info) {
  check_response_info_ = response_info;
  check_status_ = status;

  // Set consumer project_id to backend.
  if (!response_info.consumer_project_id.empty()) {
    headers.setReferenceKey(kConsumerProjectId,
                            response_info.consumer_project_id);
  }

  check_callback_->onCheckDone(status);
}

void Handler::callReport(const Http::HeaderMap* response_headers,
                         const Http::HeaderMap* /*response_trailers*/,
                         const StreamInfo::StreamInfo& stream_info) {
  ::google::api_proxy::service_control::ReportRequestInfo info;
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
  info.response_code = stream_info.responseCode().value_or(500);
  info.status = check_status_;

  info.frontend_protocol =
      getFrontendProtocol(extractHeader(*response_headers, kContentTypeHeader),
                          stream_info.protocol().has_value());

  info.backend_protocol = getBackendProtocol(
      require_ctx_->service_ctx().config().backend_protocol());

  fillGCPInfo(info);
  fillLatency(stream_info, info.latency);

  // TODO(qiwzhang): sending streaming multiple reports: b/123950356

  info.response_code = stream_info.responseCode().value_or(500);
  info.request_size = stream_info.bytesReceived();
  // TODO(qiwzhang): b/123950356, multiple reports will be send for long
  // duration requests. request_bytes is number of bytes when an intermediate
  // Report is sending. For now, we only send the final report, request_bytes is
  // the same as request_size.
  info.request_bytes = stream_info.bytesReceived();
  info.response_size = stream_info.bytesSent();
  info.response_bytes = stream_info.bytesSent();

  ::google::api::servicecontrol::v1::ReportRequest report_request;
  require_ctx_->service_ctx().builder().FillReportRequest(info,
                                                          &report_request);
  ENVOY_LOG(debug, "Sending report : {}", report_request.DebugString());

  require_ctx_->service_ctx().getTLCache().client_cache().callReport(
      report_request);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
