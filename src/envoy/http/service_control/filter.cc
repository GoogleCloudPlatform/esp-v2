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

#include "common/grpc/status.h"
#include "envoy/http/header_map.h"

#include <chrono>

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

Http::FilterHeadersStatus ServiceControlFilter::decodeHeaders(
    Http::HeaderMap& headers, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  handler_ = std::move(
      factory_.createHandler(headers, decoder_callbacks_->streamInfo()));

  state_ = Calling;
  stopped_ = false;

  handler_->callCheck(headers, *this);

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
    handler_->collectDecodeData(data, std::chrono::system_clock::now());
  }

  if (state_ == Calling) {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus ServiceControlFilter::decodeTrailers(
    Http::HeaderMap&) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

void ServiceControlFilter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks& callbacks) {
  decoder_callbacks_ = &callbacks;
}

Http::FilterHeadersStatus ServiceControlFilter::encode100ContinueHeaders(
    Http::HeaderMap&) {
  return Http::FilterHeadersStatus::Continue;
}

Http::FilterHeadersStatus ServiceControlFilter::encodeHeaders(Http::HeaderMap&,
                                                              bool) {
  return Http::FilterHeadersStatus::Continue;
}

Http::FilterDataStatus ServiceControlFilter::encodeData(Buffer::Instance& data,
                                                        bool end_stream) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  if (!end_stream && data.length() > 0) {
    handler_->collectEncodeData(data, std::chrono::system_clock::now());
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus ServiceControlFilter::encodeTrailers(
    Http::HeaderMap&) {
  return Http::FilterTrailersStatus::Continue;
}

Http::FilterMetadataStatus ServiceControlFilter::encodeMetadata(
    Http::MetadataMap&) {
  return Http::FilterMetadataStatus::Continue;
}

void ServiceControlFilter::setEncoderFilterCallbacks(
    Http::StreamEncoderFilterCallbacks& callbacks) {
  encoder_callbacks_ = &callbacks;
}

void ServiceControlFilter::log(const Http::HeaderMap* request_headers,
                               const Http::HeaderMap* response_headers,
                               const Http::HeaderMap* response_trailers,
                               const StreamInfo::StreamInfo& stream_info) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!handler_) {
    if (!request_headers) return;
    handler_ = std::move(factory_.createHandler(*request_headers, stream_info));
  }

  handler_->callReport(request_headers, response_headers, response_trailers);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
