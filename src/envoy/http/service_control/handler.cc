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
#include "common/http/utility.h"

using ::google::api::envoy::http::service_control::APIKeyLocation;
using ::google::api_proxy::service_control::CheckResponseInfo;
using ::google::api_proxy::service_control::OperationInfo;
using ::google::protobuf::util::Status;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {
const Http::LowerCaseString kConsumerProjectId("x-endpoint-api-project-id");
}

Handler::Handler(const Http::HeaderMap &headers, FilterConfigSharedPtr config)
    : config_(config) {
  http_method_ = headers.Method()->value().c_str();
  path_ = headers.Path()->value().c_str();
  require_ctx_ = config_->cfg_parser().FindRequirement(http_method_, path_);
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

bool Handler::extractAPIKeyFromQuery(const Http::HeaderMap &headers,
                                     const std::string &query) {
  if (!params_parsed_) {
    parsed_params_ =
        Http::Utility::parseQueryString(headers.Path()->value().c_str());
    params_parsed_ = true;
  }

  const auto &it = parsed_params_.find(query);
  if (it != parsed_params_.end()) {
    api_key_ = it->second;
    ENVOY_LOG(debug, "api-key: {} from query: {}", api_key_, query);
    return true;
  }
  return false;
}

bool Handler::extractAPIKeyFromHeader(const Http::HeaderMap &headers,
                                      const std::string &header) {
  // TODO(qiwzhang): optimize this by using LowerCaseString at init.
  auto *entry = headers.get(Http::LowerCaseString(header));
  if (entry) {
    api_key_ = std::string(entry->value().c_str(), entry->value().size());
    ENVOY_LOG(debug, "api-key: {} from header: {}", api_key_, header);
    return true;
  }
  return false;
}

bool Handler::extractAPIKeyFromCookie(const Http::HeaderMap &headers,
                                      const std::string &cookie) {
  std::string api_key = Http::Utility::parseCookieValue(headers, cookie);
  if (!api_key.empty()) {
    api_key_ = api_key;
    ENVOY_LOG(debug, "api-key: {} from cookie: {}", api_key_, cookie);
    return true;
  }
  return false;
}

bool Handler::extractAPIKey(
    const Http::HeaderMap &headers,
    const ::google::protobuf::RepeatedPtrField<
        ::google::api::envoy::http::service_control::APIKeyLocation>
        &locations) {
  for (const auto &location : locations) {
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
    ::google::api_proxy::service_control::OperationInfo &info) {
  info.operation_id = uuid_;
  info.operation_name = require_ctx_->config().operation_name();
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();
  info.request_start_time = std::chrono::system_clock::now();
}

void Handler::fillGCPInfo(
    ::google::api_proxy::service_control::ReportRequestInfo &info) {
  const auto &filter_config = config_->config();
  if (!filter_config.has_gcp_attributes()) {
    info.compute_platform =
        ::google::api_proxy::service_control::compute_platform::UNKNOWN;
    return;
  }

  const auto &gcp_attributes = filter_config.gcp_attributes();
  if (!gcp_attributes.zone().empty()) {
    info.location = gcp_attributes.zone();
  }

  const std::string &platform = gcp_attributes.platform();
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

void Handler::callCheck(Http::HeaderMap &headers, CheckDoneCallback &callback) {
  check_callback_ = &callback;

  // Make a check call
  ::google::api_proxy::service_control::CheckRequestInfo info;
  fillOperationInfo(info);

  // Check and Report has different rule to send api-key
  info.api_key = api_key_;

  // TODO(qiwzhang): b/124521039 to fill these api-key restriction used fields

  ::google::api::servicecontrol::v1::CheckRequest check_request;
  require_ctx_->service_ctx().builder().FillCheckRequest(info, &check_request);
  ENVOY_LOG(debug, "Sending check : {}", check_request.DebugString());

  aborted_.reset(new bool(false));
  require_ctx_->service_ctx().getTLCache().client_cache().callCheck(
      check_request,
      [this, aborted = aborted_, &headers](
          const Status &status, const CheckResponseInfo &response_info) {
        if (*aborted) return;
        onCheckResponse(headers, status, response_info);
      });
}

void Handler::onCheckResponse(Http::HeaderMap &headers, const Status &status,
                              const CheckResponseInfo &response_info) {
  check_response_info_ = response_info;
  check_status_ = status;

  // Set consumer project_id to backend.
  if (!response_info.consumer_project_id.empty()) {
    headers.setReferenceKey(kConsumerProjectId,
                            response_info.consumer_project_id);
  }

  check_callback_->onCheckDone(status);
}

void Handler::callReport(const Http::HeaderMap * /*response_headers*/,
                         const Http::HeaderMap * /*response_trailers*/,
                         const StreamInfo::StreamInfo &stream_info) {
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

  // TODO(qiwzhang): figure out frontend_protocol and backend_protocol:
  // b/123948413

  fillGCPInfo(info);

  // TODO(qiwzhang): figure out backend latency: b/123950502

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
