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

#include "src/envoy/utils/metadata_utils.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

Http::FilterHeadersStatus Filter::decodeHeaders(Http::HeaderMap& headers,
                                                bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  const std::string& operation = Utils::getStringMetadata(
      decoder_callbacks_->streamInfo().dynamicMetadata(), Utils::kOperation);

  // TODO(kyuc): the following check might not be necessary.
  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    ENVOY_LOG(debug, "No operation found from DynamicMetadata");
    rejectRequest(Http::Code(404), "Method does not exist.");
    return Http::FilterHeadersStatus::StopIteration;
  }

  handler_.reset(new Handler(headers, operation, config_));
  if (!handler_->isConfigured()) {
    rejectRequest(Http::Code(404), "Method does not exist.");
    return Http::FilterHeadersStatus::StopIteration;
  }

  if (!handler_->isCheckRequired()) {
    return Http::FilterHeadersStatus::Continue;
  }

  if (!handler_->hasApiKey()) {
    rejectRequest(Http::Code(401),
                  "Method doesn't allow unregistered callers (callers without "
                  "established identity). Please use API Key or other form of "
                  "API consumer identity to call this API.");
    return Http::FilterHeadersStatus::StopIteration;
  }

  state_ = Calling;
  stopped_ = false;

  // Make a check call
  handler_->callCheck(headers, *this, decoder_callbacks_->streamInfo());

  if (state_ == Complete) {
    return Http::FilterHeadersStatus::Continue;
  }
  ENVOY_LOG(debug, "Called ServiceControl filter : Stop");
  stopped_ = true;
  return Http::FilterHeadersStatus::StopIteration;
}

void Filter::onCheckDone(const ::google::protobuf::util::Status& status) {
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

void Filter::rejectRequest(Http::Code code, absl::string_view error_msg) {
  config_->stats().denied_.inc();
  state_ = Responded;

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

Http::FilterDataStatus Filter::decodeData(Buffer::Instance&, bool) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterDataStatus::StopIterationAndWatermark;
  }
  return Http::FilterDataStatus::Continue;
}

Http::FilterTrailersStatus Filter::decodeTrailers(Http::HeaderMap&) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (state_ == Calling) {
    return Http::FilterTrailersStatus::StopIteration;
  }
  return Http::FilterTrailersStatus::Continue;
}

void Filter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks& callbacks) {
  decoder_callbacks_ = &callbacks;
}

void Filter::log(const Http::HeaderMap* request_headers,
                 const Http::HeaderMap* response_headers,
                 const Http::HeaderMap* response_trailers,
                 const StreamInfo::StreamInfo& stream_info) {
  ENVOY_LOG(debug, "Called ServiceControl Filter : {}", __func__);
  if (!handler_) {
    if (!request_headers) return;

    // TODO(kyuc): double check if this stream_info is equivalent to the one
    // from decoder_callbacks_.
    const std::string& operation = Utils::getStringMetadata(
        stream_info.dynamicMetadata(), Utils::kOperation);
    handler_.reset(new Handler(*request_headers, operation, config_));
  }

  if (!handler_->isConfigured()) {
    return;
  }

  handler_->callReport(response_headers, response_trailers, stream_info);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
