
#include "common/http/utility.h"

#include <regex>

#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/http_call.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

using ::google::api::envoy::http::service_control::Requirement;
using ::google::api_proxy::envoy::http::service_control::ServiceControlRule;
using ::google::protobuf::util::Status;
using Envoy::Extensions::HttpFilters::ServiceControl::ServiceControlFilterConfigParser;
using Http::HeaderMap;
using std::string;

void Filter::ExtractRequestInfo(const HeaderMap& headers, Requirement*requirement)
{
  uuid_ = config_->random().uuid();
  const auto &path = headers.Path()->value();
  config_parser_->FindRequirement(headers.Method()->value().c_str(),
                                  path.c_str(), requirement);
}

Http::FilterHeadersStatus Filter::decodeHeaders(HeaderMap &headers, bool)
{
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  Requirement requirement;
  ExtractRequestInfo(headers, &requirement);
  // TODO(tianyuc): refactor the error checking logic.
  if (requirement.service_name() == "") {
    ENVOY_LOG(debug, "No requirement matched!");
    rejectRequest(Http::Code(401), "Path does not match any requirement uri_template.");
    return Http::FilterHeadersStatus::StopIteration;
  }
  operation_name_ = requirement.operation_name();

  // TODO(tianyuc): API key should be parsed from the request.
  api_key_ = "AIzaSyB3xeV9fv4agFXUpGVyPMtZ2xIMScEazrk";
  api_name_ = requirement.api_name();
  api_version_ = requirement.api_version();

  state_ = Calling;
  stopped_ = false;
  token_fetcher_ = config_->getCache().getTokenCacheByServiceName(requirement.service_name())
                       ->getToken(
                           [this](const Status &status, const string &result) {
                             onTokenDone(status, result);
                           });

  if (state_ == Complete)
  {
    return Http::FilterHeadersStatus::Continue;
  }
  ENVOY_LOG(debug, "Called ServiceControl filter : Stop");
  stopped_ = true;
  return Http::FilterHeadersStatus::StopIteration;
}

void Filter::onDestroy()
{
  if (token_fetcher_)
  {
    token_fetcher_();
    token_fetcher_ = nullptr;
  }
  if (check_call_)
  {
    check_call_->cancel();
    check_call_ = nullptr;
  }
}

void Filter::onTokenDone(const Status &status, const string &token)
{
  // This stream has been reset, abort the callback.
  token_fetcher_ = nullptr;
  if (state_ == Responded)
  {
    return;
  }

  if (!status.ok())
  {
    rejectRequest(Http::Code(401), "Failed to fetch access_token");
    return;
  }

  token_ = token;
  // Make a check call
  ::google::api_proxy::service_control::CheckRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id = config_->config().producer_project_id();
  info.api_key = api_key_;
  info.request_start_time = std::chrono::system_clock::now();

  ::google::api::servicecontrol::v1::CheckRequest check_request;
  config_->builder().FillCheckRequest(info, &check_request);
  ENVOY_LOG(debug, "Sending check : {}", check_request.DebugString());

  string suffix_uri = config_->config().service_name() + ":check";
  auto on_done = [this](const Status &status, const string &body) {
    onCheckResponse(status, body);
  };
  check_call_ =
      HttpCall::create(config_->cm(), config_->config().service_control_uri());
  check_call_->call(suffix_uri, token_, check_request, on_done);
}

void Filter::rejectRequest(Http::Code code, const string &error_msg)
{
  config_->stats().denied_.inc();
  state_ = Responded;

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

void Filter::onCheckResponse(const Status &status,
                             const string &response_json)
{
  ENVOY_LOG(debug, "Check response with : {}, body {}", status.ToString(),
            response_json);
  // This stream has been reset, abort the callback.
  check_call_ = nullptr;
  if (state_ == Responded)
  {
    return;
  }

  if (!status.ok())
  {
    rejectRequest(Http::Code(401), "Check failed");
    return;
  }

  ::google::api::servicecontrol::v1::CheckResponse response_pb;
  Protobuf::util::JsonParseOptions options;
  options.ignore_unknown_fields = true;
  const auto json_status =
      Protobuf::util::JsonStringToMessage(response_json, &response_pb, options);
  if (!json_status.ok())
  {
    rejectRequest(Http::Code(401), "Check failed");
    return;
  }

  check_status_ = ::google::api_proxy::service_control::RequestBuilder::
      ConvertCheckResponse(response_pb, config_->config().service_name(),
                           &check_response_info_);
  if (!check_status_.ok())
  {
    rejectRequest(Http::Code(401), "Check failed");
    return;
  }

  config_->stats().allowed_.inc();
  state_ = Complete;
  if (stopped_)
  {
    decoder_callbacks_->continueDecoding();
  }
}

Http::FilterDataStatus Filter::decodeData(Buffer::Instance &, bool)
{
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling)
  {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus Filter::decodeTrailers(HeaderMap &)
{
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling)
  {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

void Filter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks &callbacks)
{
  decoder_callbacks_ = &callbacks;
}

void Filter::log(const HeaderMap * /*request_headers*/,
                 const HeaderMap * /*response_headers*/,
                 const HeaderMap * /*response_trailers*/,
                 const StreamInfo::StreamInfo &stream_info)
{
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  ::google::api_proxy::service_control::ReportRequestInfo info;
  info.operation_id = uuid_;
  info.operation_name = operation_name_;
  info.producer_project_id = config_->config().producer_project_id();

  if (check_response_info_.is_api_key_valid &&
      check_response_info_.service_is_activated)
  {
    info.api_key = api_key_;
  }

  info.request_start_time = std::chrono::system_clock::now();
  info.api_method = operation_name_;
  info.api_name = api_name_;
  info.api_version = api_version_;
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
  config_->builder().FillReportRequest(info, &report_request);
  ENVOY_LOG(debug, "Sending report : {}", report_request.DebugString());

  string suffix_uri = config_->config().service_name() + ":report";
  auto dummy_on_done = [](const Status &, const string &) {};
  HttpCall *http_call =
      HttpCall::create(config_->cm(), config_->config().service_control_uri());
  http_call->call(suffix_uri, token_, report_request, dummy_on_done);
}

} // namespace ServiceControl
} // namespace HttpFilters
} // namespace Extensions
} // namespace Envoy
