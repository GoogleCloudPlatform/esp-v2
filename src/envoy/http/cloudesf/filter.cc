/* Copyright 2017 Istio Authors. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "common/http/utility.h"

#include "src/envoy/http/cloudesf/filter.h"
#include "src/envoy/http/cloudesf/http_call.h"
#include "src/envoy/http/cloudesf/service_control/proto.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace CloudESF {

void Filter::ExtractRequestInfo(const Http::HeaderMap& headers) {
  uuid_ = config_->random().uuid();

  // operation_name from path
  const auto& path = headers.Path()->value();
  const char* query_start = Http::Utility::findQueryStringStart(path);
  if (query_start != nullptr) {
    operation_name_ = std::string(path.c_str(), query_start - path.c_str());
  } else {
    operation_name_ = std::string(path.c_str(), path.size());
  }

  // api key
  auto params =
      Http::Utility::parseQueryString(headers.Path()->value().c_str());
  const auto& it = params.find("key");
  if (it != params.end()) {
    api_key_ = it->second;
  }

  http_method_ = headers.Method()->value().c_str();
}

Http::FilterHeadersStatus Filter::decodeHeaders(Http::HeaderMap& headers,
                                                bool) {
  ENVOY_LOG(debug, "Called CloudESF Filter : {}", __func__);

  ExtractRequestInfo(headers);

  state_ = Calling;
  stopped_ = false;
  token_fetcher_ = TokenFetcher::create(config_->cm());
  token_fetcher_->fetch(config_->config().token_uri(), *this);

  if (state_ == Complete) {
    return Http::FilterHeadersStatus::Continue;
  }
  ENVOY_LOG(debug, "Called CloudESF filter : Stop");
  stopped_ = true;
  return Http::FilterHeadersStatus::StopIteration;
}

void Filter::onDestroy() {
  if (token_fetcher_) {
    token_fetcher_->cancel();
    token_fetcher_ = nullptr;
  }
  if (check_call_) {
    check_call_->cancel();
    check_call_ = nullptr;
  }
}

void Filter::onTokenSuccess(const std::string& token, int expires_in) {
  ENVOY_LOG(debug, "Fetched access_token : {}, expires_in {}", token,
            expires_in);
  token_ = token;
  // This stream has been reset, abort the callback.
  if (state_ == Responded) {
    return;
  }

  // Make a check call
  ::google::service_control::CheckRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id = config_->config().producer_project_id();
  info.api_key = api_key_;
  info.request_start_time = std::chrono::system_clock::now();

  ::google::api::servicecontrol::v1::CheckRequest check_request;
  config_->proto_builder().FillCheckRequest(info, &check_request);
  ENVOY_LOG(debug, "Sending check : {}", check_request.DebugString());

  std::string suffix_uri = config_->config().service_name() + ":check";
  auto on_done = [this](const ::google::protobuf::util::Status& status,
                        const std::string& body) {
    onCheckResponse(status, body);
  };
  check_call_ =
      HttpCall::create(config_->cm(), config_->config().service_control_uri());
  check_call_->call(suffix_uri, token_, check_request, on_done);
}

void Filter::onCheckResponse(const ::google::protobuf::util::Status& status,
                             const std::string& response_json) {
  ENVOY_LOG(debug, "Check response with : {}, body {}", status.ToString(),
            response_json);
  // This stream has been reset, abort the callback.
  check_call_ = nullptr;
  if (state_ == Responded) {
    return;
  }

  if (!status.ok()) {
    state_ = Responded;

    Http::Code code = Http::Code(401);
    decoder_callbacks_->sendLocalReply(code, "Check failed", nullptr);
    decoder_callbacks_->streamInfo().setResponseFlag(
        StreamInfo::ResponseFlag::UnauthorizedExternalService);
    return;
  }

  ::google::api::servicecontrol::v1::CheckResponse response_pb;
  Protobuf::util::JsonParseOptions options;
  options.ignore_unknown_fields = true;
  const auto json_status =
      Protobuf::util::JsonStringToMessage(response_json, &response_pb, options);
  if (!json_status.ok()) {
    state_ = Responded;

    Http::Code code = Http::Code(401);
    decoder_callbacks_->sendLocalReply(code, "Check failed", nullptr);
    decoder_callbacks_->streamInfo().setResponseFlag(
        StreamInfo::ResponseFlag::UnauthorizedExternalService);
    return;
  }

  check_status_ = ::google::service_control::Proto::ConvertCheckResponse(
      response_pb, config_->config().service_name(), &check_response_info_);
  if (!check_status_.ok()) {
    state_ = Responded;

    Http::Code code = Http::Code(401);
    decoder_callbacks_->sendLocalReply(code, "Check failed", nullptr);
    decoder_callbacks_->streamInfo().setResponseFlag(
        StreamInfo::ResponseFlag::UnauthorizedExternalService);
    return;
  }

  state_ = Complete;
  if (stopped_) {
    decoder_callbacks_->continueDecoding();
  }
}

void Filter::onTokenError(TokenFetcher::TokenReceiver::Failure) {
  // This stream has been reset, abort the callback.
  if (state_ == Responded) {
    return;
  }
  state_ = Responded;

  Http::Code code = Http::Code(401);
  decoder_callbacks_->sendLocalReply(code, "Failed to fetch access_token",
                                     nullptr);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

Http::FilterDataStatus Filter::decodeData(Buffer::Instance&, bool) {
  ENVOY_LOG(debug, "Called CloudESF Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus Filter::decodeTrailers(Http::HeaderMap&) {
  ENVOY_LOG(debug, "Called CloudESF Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

void Filter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks& callbacks) {
  decoder_callbacks_ = &callbacks;
}

void Filter::log(const Http::HeaderMap* /*request_headers*/,
                 const Http::HeaderMap* /*response_headers*/,
                 const Http::HeaderMap* /*response_trailers*/,
                 const StreamInfo::StreamInfo& stream_info) {
  ENVOY_LOG(debug, "Called CloudESF Filter : {}", __func__);

  ::google::service_control::ReportRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id = config_->config().producer_project_id();

  if (check_response_info_.is_api_key_valid &&
      check_response_info_.service_is_activated) {
    info.api_key = api_key_;
  }

  info.request_start_time = std::chrono::system_clock::now();
  info.api_method = operation_name_;
  info.api_name = "Bookstore";
  info.api_version = "1.0";
  info.log_message = operation_name_ + " is called";

  info.url = operation_name_;
  info.method = http_method_;

  info.check_response_info = check_response_info_;
  info.response_code = stream_info.responseCode().value_or(500);
  info.status = check_status_;

  info.response_code = stream_info.responseCode().value_or(500);
  info.request_size = stream_info.bytesReceived();
  info.response_size = stream_info.bytesSent();

  ::google::api::servicecontrol::v1::ReportRequest report_request;
  config_->proto_builder().FillReportRequest(info, &report_request);
  ENVOY_LOG(debug, "Sending report : {}", report_request.DebugString());

  std::string suffix_uri = config_->config().service_name() + ":report";
  auto dummy_on_done = [](const ::google::protobuf::util::Status&,
                          const std::string&) {};
  HttpCall* http_call =
      HttpCall::create(config_->cm(), config_->config().service_control_uri());
  http_call->call(suffix_uri, token_, report_request, dummy_on_done);
}

}  // namespace CloudESF
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
