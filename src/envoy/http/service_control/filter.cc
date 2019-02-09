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

#include "src/envoy/http/service_control/filter.h"
#include "common/http/utility.h"
#include "envoy/http/header_map.h"
#include "src/api_proxy/service_control/request_builder.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

using ::google::api::envoy::http::service_control::APIKeyLocation;
using ::google::api::envoy::http::service_control::APIKeyRequirement;
using ::google::api_proxy::service_control::CheckResponseInfo;
using ::google::protobuf::util::Status;
using Http::HeaderMap;
using Http::LowerCaseString;
using std::string;

bool Filter::ExtractAPIKeyFromQuery(const HeaderMap &headers,
                                    const string &query) {
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

bool Filter::ExtractAPIKeyFromHeader(const HeaderMap &headers,
                                     const string &header) {
  // TODO(qiwzhang): optimize this by using LowerCaseString at init.
  auto *entry = headers.get(LowerCaseString(header));
  if (entry) {
    api_key_ = std::string(entry->value().c_str(), entry->value().size());
    ENVOY_LOG(debug, "api-key: {} from header: {}", api_key_, header);
    return true;
  }
  return false;
}

bool Filter::ExtractAPIKeyFromCookie(const HeaderMap &headers,
                                     const string &cookie) {
  std::string api_key = Http::Utility::parseCookieValue(headers, cookie);
  if (!api_key.empty()) {
    api_key_ = api_key;
    ENVOY_LOG(debug, "api-key: {} from cookie: {}", api_key_, cookie);
    return true;
  }
  return false;
}

bool Filter::ExtractAPIKey(
    const HeaderMap &headers,
    const ::google::protobuf::RepeatedPtrField<
        ::google::api::envoy::http::service_control::APIKeyLocation>
        &locations) {
  for (const auto &location : locations) {
    switch (location.key_case()) {
      case APIKeyLocation::kQuery:
        if (ExtractAPIKeyFromQuery(headers, location.query())) return true;
        break;
      case APIKeyLocation::kHeader:
        if (ExtractAPIKeyFromHeader(headers, location.header())) return true;
        break;
      case APIKeyLocation::kCookie:
        if (ExtractAPIKeyFromCookie(headers, location.cookie())) return true;
        break;
      case APIKeyLocation::KEY_NOT_SET:
        break;
    }
  }
  return false;
}

Http::FilterHeadersStatus Filter::decodeHeaders(HeaderMap &headers, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  uuid_ = config_->random().uuid();
  require_ctx_ = config_->cfg_parser().FindRequirement(
      headers.Method()->value().c_str(), headers.Path()->value().c_str());
  if (!require_ctx_) {
    ENVOY_LOG(debug, "No requirement matched!");
    rejectRequest(Http::Code(404),
                  "Path does not match any requirement uri_template.");
    return Http::FilterHeadersStatus::StopIteration;
  }

  operation_name_ = require_ctx_->config().operation_name();
  api_name_ = require_ctx_->config().api_name();
  api_version_ = require_ctx_->config().api_version();

  // TODO add integration tests
  if (require_ctx_->config().api_key().allow_without_api_key()) {
    ENVOY_LOG(debug, "Service control check is not needed");
    return Http::FilterHeadersStatus::Continue;
  }

  if (require_ctx_->config().api_key().locations_size() > 0) {
    ExtractAPIKey(headers, require_ctx_->config().api_key().locations());
  } else {
    ExtractAPIKey(headers, config_->default_api_keys().locations());
  }

  state_ = Calling;
  stopped_ = false;

  // Make a check call
  ::google::api_proxy::service_control::CheckRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();
  info.api_key = api_key_;
  info.request_start_time = std::chrono::system_clock::now();

  ::google::api::servicecontrol::v1::CheckRequest check_request;
  require_ctx_->service_ctx().builder().FillCheckRequest(info, &check_request);
  ENVOY_LOG(debug, "Sending check : {}", check_request.DebugString());

  aborted_.reset(new bool);
  *aborted_ = false;

  require_ctx_->service_ctx().getTLCache().client_cache().callCheck(
      check_request,
      [this, aborted = aborted_](const Status &status,
                                 const CheckResponseInfo &response_info) {
        if (*aborted) return;
        onCheckResponse(status, response_info);
      });

  if (state_ == Complete) {
    return Http::FilterHeadersStatus::Continue;
  }
  ENVOY_LOG(debug, "Called ServiceControl filter : Stop");
  stopped_ = true;
  return Http::FilterHeadersStatus::StopIteration;
}

void Filter::onDestroy() {
  if (aborted_) {
    *aborted_ = true;
  }
}

void Filter::rejectRequest(Http::Code code, absl::string_view error_msg) {
  config_->stats().denied_.inc();
  state_ = Responded;

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

void Filter::onCheckResponse(const Status &status,
                             const CheckResponseInfo &response_info) {
  check_response_info_ = response_info;
  check_status_ = status;

  if (!status.ok()) {
    rejectRequest(Http::Code(401), "Check failed");
    return;
  }

  config_->stats().allowed_.inc();
  state_ = Complete;
  if (stopped_) {
    decoder_callbacks_->continueDecoding();
  }
}

Http::FilterDataStatus Filter::decodeData(Buffer::Instance &, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus Filter::decodeTrailers(HeaderMap &) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

void Filter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks &callbacks) {
  decoder_callbacks_ = &callbacks;
}

void Filter::log(const HeaderMap *request_headers,
                 const HeaderMap * /*response_headers*/,
                 const HeaderMap * /*response_trailers*/,
                 const StreamInfo::StreamInfo &stream_info) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!require_ctx_) {
    return;
  }

  ::google::api_proxy::service_control::ReportRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id =
      require_ctx_->service_ctx().config().producer_project_id();

  if (check_response_info_.is_api_key_valid &&
      check_response_info_.service_is_activated) {
    info.api_key = api_key_;
  }

  info.request_start_time = std::chrono::system_clock::now();
  info.api_method = operation_name_;
  info.api_name = api_name_;
  info.api_version = api_version_;
  info.log_message = operation_name_ + " is called";

  if (request_headers) {
    if (request_headers->Path()) {
      info.url = request_headers->Path()->value().c_str();
    }
    if (request_headers->Method()) {
      info.method = request_headers->Method()->value().c_str();
    }
  }

  info.check_response_info = check_response_info_;
  info.response_code = stream_info.responseCode().value_or(500);
  info.status = check_status_;

  // TODO(qiwzhang): figure out frontend_protocol and backend_protocol:
  // b/123948413

  // TODO(qiwzhang): figure out platform: b/123950206

  // TODO(qiwzhang): figure out backend latency: b/123950502

  // TODO(qiwzhang): sending streaming multiple reports: b/123950356

  info.response_code = stream_info.responseCode().value_or(500);
  info.request_size = stream_info.bytesReceived();
  // TODO(qiwzhang): b/123950356, multiple reports will be send for long duration requests.
  // request_bytes is number of bytes when an intermediate Report is sending.
  // For now, we only send the final report, request_bytes is the same as request_size.
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
