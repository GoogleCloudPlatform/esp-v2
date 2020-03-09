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

#include "common/grpc/status.h"
#include "envoy/http/header_map.h"
#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/handler.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

struct RcDetailsValues {
  // Rejected by service control check call.
  const std::string RejectedByServiceControlCheck =
      "rejected_by_service_control_check";
};
typedef ConstSingleton<RcDetailsValues> RcDetails;

}  // namespace

void ServiceControlFilter::onDestroy() {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (handler_) {
    handler_->onDestroy();
  }
}

Http::FilterHeadersStatus ServiceControlFilter::decodeHeaders(
    Http::RequestHeaderMap& headers, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  Envoy::Tracing::Span& parent_span = decoder_callbacks_->activeSpan();

  handler_ = factory_.createHandler(headers, decoder_callbacks_->streamInfo());

  state_ = Calling;
  stopped_ = false;

  handler_->callCheck(headers, parent_span, *this);

  // If success happens synchronously, continue now.
  if (state_ == Complete) {
    return Http::FilterHeadersStatus::Continue;
  }

  // Stop for now. If an async request is made, it will continue in onCheckDone.
  ENVOY_LOG(debug, "Called ServiceControl filter : Stop");
  stopped_ = true;
  return Http::FilterHeadersStatus::StopIteration;
}

void ServiceControlFilter::onCheckDone(
    const ::google::protobuf::util::Status& status) {
  if (!status.ok()) {
    // protobuf::util::Status.error_code is the same as Envoy GrpcStatus
    // This cast is safe.
    auto http_code = Grpc::Utility::grpcToHttpStatus(
        static_cast<Grpc::Status::GrpcStatus>(status.error_code()));
    rejectRequest(static_cast<Http::Code>(http_code), status.ToString());
    return;
  }

  stats_.allowed_.inc();
  state_ = Complete;
  if (stopped_) {
    decoder_callbacks_->continueDecoding();
  }
}

void ServiceControlFilter::rejectRequest(Http::Code code,
                                         absl::string_view error_msg) {
  stats_.denied_.inc();
  state_ = Responded;

  decoder_callbacks_->sendLocalReply(
      code, error_msg, nullptr, absl::nullopt,
      RcDetails::get().RejectedByServiceControlCheck);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

Http::FilterDataStatus ServiceControlFilter::decodeData(Buffer::Instance& data,
                                                        bool end_stream) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!end_stream && data.length() > 0) {
    handler_->tryIntermediateReport();
  }

  if (state_ == Calling) {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus ServiceControlFilter::decodeTrailers(
    Http::RequestTrailerMap&) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

Http::FilterHeadersStatus ServiceControlFilter::encodeHeaders(
    Http::ResponseHeaderMap& headers, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {} before", __func__);

  // For the cases the decodeHeaders not called, like the request get failed in
  // the Jwt-Authn filter, the handler_ is not initialized.
  if (handler_ != nullptr) {
    handler_->processResponseHeaders(headers);
  }
  return Http::FilterHeadersStatus::Continue;
}

Http::FilterDataStatus ServiceControlFilter::encodeData(Buffer::Instance& data,
                                                        bool end_stream) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!end_stream && data.length() > 0) {
    handler_->tryIntermediateReport();
  }
  return Http::FilterDataStatus::Continue;
}

void ServiceControlFilter::log(
    const Http::RequestHeaderMap* request_headers,
    const Http::ResponseHeaderMap* response_headers,
    const Http::ResponseTrailerMap* response_trailers,
    const StreamInfo::StreamInfo& stream_info) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!handler_) {
    if (!request_headers) return;
    handler_ = factory_.createHandler(*request_headers, stream_info);
  }

  handler_->callReport(request_headers, response_headers, response_trailers);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
