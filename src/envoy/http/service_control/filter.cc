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

#include "src/envoy/http/service_control/filter.h"

#include <chrono>

#include "common/grpc/status.h"
#include "envoy/http/header_map.h"
#include "src/envoy/http/service_control/handler.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

void ServiceControlFilter::onDestroy() {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (handler_) {
    handler_->onDestroy();
  }
}

Envoy::Http::FilterHeadersStatus ServiceControlFilter::decodeHeaders(
    Envoy::Http::RequestHeaderMap& headers, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);

  if (!headers.Method()) {
    rejectRequest(Envoy::Http::Code::BadRequest,
                  "No method in request headers.",
                  utils::generateRcDetails(utils::kRcDetailFilterServiceControl,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorMissingMethod));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (!headers.Path()) {
    rejectRequest(Envoy::Http::Code::BadRequest, "No path in request headers.",
                  utils::generateRcDetails(utils::kRcDetailFilterServiceControl,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorMissingPath));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  if (utils::handleHttpMethodOverride(headers)) {
    // Update later filters that the HTTP method has changed by clearing the
    // route cache.
    ENVOY_LOG(debug, "HTTP method override occurred, recalculating route");
    decoder_callbacks_->clearRouteCache();
  }

  // Make sure route is calculated
  auto route = decoder_callbacks_->route();
  if (route == nullptr) {
    rejectRequest(
        Envoy::Http::Code::NotFound,
        absl::StrCat("Request `", utils::readHeaderEntry(headers.Method()), " ",
                     utils::readHeaderEntry(headers.Path()),
                     "` is not defined by this API."),
        utils::generateRcDetails(utils::kRcDetailFilterServiceControl,
                                 utils::kRcDetailErrorTypeUndefinedRequest));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  handler_ =
      factory_.createHandler(headers, decoder_callbacks_->streamInfo(), stats_);
  handler_->fillFilterState(*decoder_callbacks_->streamInfo().filterState());
  state_ = Calling;
  stopped_ = false;

  Envoy::Tracing::Span& parent_span = decoder_callbacks_->activeSpan();

  handler_->callCheck(headers, parent_span, *this);

  // If success happens synchronously, continue now.
  if (state_ == Complete) {
    return Envoy::Http::FilterHeadersStatus::Continue;
  }

  // Stop for now. If an async request is made, it will continue in onCheckDone.
  ENVOY_LOG(debug, "Called ServiceControl filter : Stop");
  stopped_ = true;
  return Envoy::Http::FilterHeadersStatus::StopIteration;
}

void ServiceControlFilter::onCheckDone(
    const ::google::protobuf::util::Status& status,
    absl::string_view rc_detail) {
  if (!status.ok()) {
    // protobuf::util::Status.error_code is the same as Envoy GrpcStatus
    // This cast is safe.
    auto http_code = Envoy::Grpc::Utility::grpcToHttpStatus(
        static_cast<Envoy::Grpc::Status::GrpcStatus>(status.error_code()));
    rejectRequest(static_cast<Envoy::Http::Code>(http_code), status.ToString(),
                  rc_detail);
    return;
  }

  stats_.filter_.allowed_.inc();
  state_ = Complete;
  if (stopped_) {
    decoder_callbacks_->continueDecoding();
  }
}

void ServiceControlFilter::rejectRequest(Envoy::Http::Code code,
                                         absl::string_view error_msg,
                                         absl::string_view rc_detail) {
  stats_.filter_.denied_.inc();
  state_ = Responded;

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     rc_detail);
  decoder_callbacks_->streamInfo().setResponseFlag(
      Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

Envoy::Http::FilterDataStatus ServiceControlFilter::decodeData(
    Envoy::Buffer::Instance&, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Envoy::Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Envoy::Http::FilterDataStatus::Continue;
}

Envoy::Http::FilterTrailersStatus ServiceControlFilter::decodeTrailers(
    Envoy::Http::RequestTrailerMap&) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Envoy::Http::FilterTrailersStatus::StopIteration;
  }
  return Envoy::Http::FilterTrailersStatus::Continue;
}

void ServiceControlFilter::log(
    const Envoy::Http::RequestHeaderMap* request_headers,
    const Envoy::Http::ResponseHeaderMap* response_headers,
    const Envoy::Http::ResponseTrailerMap* response_trailers,
    const Envoy::StreamInfo::StreamInfo& stream_info) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!handler_) {
    if (!request_headers) return;
    handler_ = factory_.createHandler(*request_headers, stream_info, stats_);
  }

  handler_->callReport(request_headers, response_headers, response_trailers);
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
