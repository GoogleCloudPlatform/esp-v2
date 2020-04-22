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

#include "src/envoy/http/error_translator/filter.h"

#include <string>

#include "absl/types/optional.h"
#include "common/buffer/buffer_impl.h"
#include "common/grpc/status.h"
#include "common/http/headers.h"
#include "src/envoy/utils/filter_state_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

using ::Envoy::Http::FilterDataStatus;
using ::Envoy::Http::FilterHeadersStatus;

Filter::Filter(FilterConfigSharedPtr config) : config_(config) {}

FilterHeadersStatus Filter::encodeHeaders(
    Envoy::Http::ResponseHeaderMap& headers, bool) {
  if (isUpstreamResponse()) {
    // Do not translate any response sent from the upstream.
    return FilterHeadersStatus::Continue;
  }

  if (headers.ContentType() != nullptr &&
      headers.ContentType()->value().getStringView() ==
          Envoy::Http::Headers::get().ContentTypeValues.Grpc) {
    is_grpc_response_ = true;
  }

  if (isEspv2FilterError() && is_grpc_response_) {
    // Do not translate for gRPC, body does not matter.
    // gRPC error is already in filter state as well, so nothing to do.
    return FilterHeadersStatus::Continue;
  }

  if (isEspv2FilterError()) {
    // We need to modify the response body, but keep filter state as is.
    ENVOY_LOG(debug, "Translating HTTP espv2 error headers");

    // Store a copy and scrub details if needed.
    error_ = utils::getErrorFilterState(
        *encoder_callbacks_->streamInfo().filterState());
    if (config_->shouldScrubDebugDetails()) {
      error_.clear_details();
    }

    // Translate to JSON.
    error_json_ = errorToJson(error_);

    // Update content headers.
    headers.setContentLength(error_json_.size());
    headers.setReferenceContentType(
        Envoy::Http::Headers::get().ContentTypeValues.Json);

    return FilterHeadersStatus::Continue;
  }

  if (is_grpc_response_) {
    // We keep the response body as is, but modify the filter state.
    // This will be used by the access log (SC Report).
    // This handles the trailers-only response by `sendLocalReply`.
    ENVOY_LOG(debug,
              "Translating gRPC non-espv2 (upstream envoy filter) "
              "trailers-only error response");

    if (headers.GrpcStatus() != nullptr) {
      const absl::string_view grpc_status =
          headers.GrpcStatus()->value().getStringView();
      uint32_t parsed;
      if (absl::SimpleAtoi(grpc_status, &parsed)) {
        error_.set_code(parsed);
      } else {
        error_.set_code(Envoy::Grpc::Status::WellKnownGrpcStatus::Internal);
      }
    } else {
      error_.set_code(Envoy::Grpc::Status::WellKnownGrpcStatus::Internal);
    }
    if (headers.GrpcMessage() != nullptr) {
      const std::string grpc_message =
          std::string(headers.GrpcMessage()->value().getStringView());
      error_.set_message(grpc_message);
    }

    // Store in filter state.
    utils::setErrorFilterState(*encoder_callbacks_->streamInfo().filterState(),
                               error_);

    return FilterHeadersStatus::Continue;
  }

  // Now we are handling `sendLocalReply` from non-espv2 filters.
  // We need to modify the response body and filter state.
  // TODO(nareddyt): Do we also need to check the content type is "plain/text"?
  ENVOY_LOG(debug,
            "Translating HTTP non-espv2 (upstream envoy filter) error headers");

  // Stop iteration because we will modify some headers based on the body
  // (example: content length). Expect encodeData to unblock this.
  headers_ = &headers;
  return FilterHeadersStatus::StopIteration;
}

FilterDataStatus Filter::encodeData(Envoy::Buffer::Instance& body,
                                    bool end_stream) {
  if (isUpstreamResponse()) {
    // Do not translate any response sent from the upstream.
    return FilterDataStatus::Continue;
  }

  if (is_grpc_response_) {
    // Do not translate body for gRPC. It is not used.
    return FilterDataStatus::Continue;
  }

  if (isEspv2FilterError()) {
    ENVOY_LOG(debug, "Translating HTTP espv2 error body");
    GOOGLE_DCHECK(!error_.message().empty());
    GOOGLE_DCHECK(!error_json_.empty());

    if (!end_stream) {
      // Keep buffering data until we have the full response.
      // TODO(nareddyt): Is there a better way to discard the remaining data? We
      // don't care about it.
      return FilterDataStatus::StopIterationAndBuffer;
    }

    // Replace the body with the JSON error.
    Envoy::Buffer::OwnedImpl json_body(error_json_);
    body.move(json_body);
    return FilterDataStatus::Continue;
  }

  // Now we are handling `sendLocalReply` from non-espv2 filters.
  // We need to modify the response body and filter state.
  ENVOY_LOG(debug,
            "Translating HTTP non-espv2 (upstream envoy filter) error body");
  GOOGLE_DCHECK_NE(headers_, nullptr);

  if (!end_stream) {
    // Keep buffering data until we have the full response.
    return FilterDataStatus::StopIterationAndBuffer;
  }

  // Craft the error. Translate to canonical status codes.
  const unsigned int http_code =
      encoder_callbacks_->streamInfo().responseCode().value_or(500);
  error_.set_code(Envoy::Grpc::Utility::httpToGrpcStatus(http_code));

  const std::string body_string = body.toString();
  error_.set_message(body_string);

  // Set in the filter state for the access log (SC Report).
  utils::setErrorFilterState(*encoder_callbacks_->streamInfo().filterState(),
                             error_);

  // Translate to json.
  error_json_ = errorToJson(error_);

  // Update content headers.
  headers_->setContentLength(error_json_.size());
  headers_->setReferenceContentType(
      Envoy::Http::Headers::get().ContentTypeValues.Json);

  // Replace the body with the JSON error.
  Envoy::Buffer::OwnedImpl json_body(error_json_);
  body.move(json_body);
  return FilterDataStatus::Continue;
}

bool Filter::isUpstreamResponse() {
  const absl::optional<std::string>& response_details =
      encoder_callbacks_->streamInfo().responseCodeDetails();
  if (!response_details) {
    return false;
  }
  return *response_details !=
         Envoy::StreamInfo::ResponseCodeDetails::get().ViaUpstream;
}

bool Filter::isEspv2FilterError() {
  return utils::hasErrorFilterState(
      *encoder_callbacks_->streamInfo().filterState());
}

std::string Filter::errorToJson(google::rpc::Status&) {
  // TODO(nareddyt): convert to JSON.
  return "";
}

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
