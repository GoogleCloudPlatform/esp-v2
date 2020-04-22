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
#include "common/common/assert.h"
#include "common/grpc/common.h"
#include "common/grpc/status.h"
#include "common/http/headers.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/google/protobuf/util/json_util.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

using ::Envoy::Http::FilterDataStatus;
using ::Envoy::Http::FilterHeadersStatus;

Filter::Filter(FilterConfigSharedPtr config) : config_(config) {
  type_helper_ = std::make_unique<google::grpc::transcoding::TypeHelper>(
      Envoy::Protobuf::util::NewTypeResolverForDescriptorPool(
          Envoy::Grpc::Common::typeUrlPrefix(), &descriptor_pool_));

  print_options_.add_whitespace = true;
  print_options_.always_print_primitive_fields = true;
  print_options_.always_print_enums_as_ints = false;
  print_options_.preserve_proto_field_names = false;
}

FilterHeadersStatus Filter::encodeHeaders(
    Envoy::Http::ResponseHeaderMap& headers, bool) {
  if (isUpstreamResponse()) {
    // Do not translate any response sent from the upstream.
    return FilterHeadersStatus::Continue;
  }

  if (Envoy::Grpc::Common::hasGrpcContentType(headers)) {
    is_grpc_response_ = true;
  }

  if (isEspv2FilterError() && is_grpc_response_) {
    // gRPC error is already in filter state. No need to translate body, as it
    // is not used for gRPC errors. Hence, nothing to do.
    return FilterHeadersStatus::Continue;
  }

  if (is_grpc_response_) {
    // We keep the response body as is, but modify the filter state.
    // This will be used by the access log (SC Report).
    // This handles the trailers-only response by `sendLocalReply`.
    ENVOY_LOG(debug,
              "Translating gRPC non-espv2 trailers-only error response: {}",
              __func__);

    // Craft the error (using canonical status code from gRPC header).
    google::rpc::Status error;
    const absl::optional<Envoy::Grpc::Status::GrpcStatus> status =
        Envoy::Grpc::Common::getGrpcStatus(headers);
    if (status) {
      error.set_code(*status);
    } else {
      error.set_code(Envoy::Grpc::Status::WellKnownGrpcStatus::Internal);
    }

    const std::string grpc_message =
        Envoy::Grpc::Common::getGrpcMessage(headers);
    error.set_message(grpc_message);

    // Store in filter state.
    utils::setErrorFilterState(*encoder_callbacks_->streamInfo().filterState(),
                               error);

    return FilterHeadersStatus::Continue;
  }

  // Now we are handling `sendLocalReply` for HTTP from espv2 and non-espv2
  // filters. We need to modify the response body and filter state.
  ENVOY_LOG(debug,
            "Storing HTTP error headers for espv2 and non-espv2 errors: {}",
            __func__);

  // Stop iteration because we will modify some headers based on the body
  // (example: content length). Expect encodeData() to unblock this.
  headers_ = &headers;
  return FilterHeadersStatus::StopIteration;
}

FilterDataStatus Filter::encodeData(Envoy::Buffer::Instance& body,
                                    bool end_stream) {
  if (isUpstreamResponse()) {
    // Do not translate any response sent from the upstream.
    return FilterDataStatus::Continue;
  }

  google::rpc::Status error;
  RELEASE_ASSERT(!is_grpc_response_,
                 "sendLocalReply() for gRPC is trailers-only response, there "
                 "should be no body");
  RELEASE_ASSERT(end_stream,
                 "sendLocalReply() will not buffer data or send trailers");
  RELEASE_ASSERT(
      headers_ != nullptr,
      "encodeHeaders() should have stored headers, as they will be modified");

  if (isEspv2FilterError()) {
    // We need to scrub details from the error (in the filter state).
    ENVOY_LOG(debug, "Translating HTTP espv2 error: {}", __func__);

    error = utils::getErrorFilterState(
        *encoder_callbacks_->streamInfo().filterState());
    if (config_->shouldScrubDebugDetails()) {
      error.clear_details();
    }

  } else {
    // We need to craft the error and store it in the filter state.
    ENVOY_LOG(debug, "Translating HTTP non-espv2 error: {}", __func__);

    // Note: Translate to canonical status codes.
    const unsigned int http_code =
        encoder_callbacks_->streamInfo().responseCode().value_or(500);
    const std::string body_string = body.toString();
    error.set_code(Envoy::Grpc::Utility::httpToGrpcStatus(http_code));
    error.set_message(body_string);

    // Set in the filter state for the access log (SC Report).
    utils::setErrorFilterState(*encoder_callbacks_->streamInfo().filterState(),
                               error);
  }

  // Once the error is crafted, we translate to JSON.
  std::string error_json;
  const Envoy::ProtobufUtil::Status json_status =
      errorToJson(error, &error_json);
  if (!json_status.ok()) {
    // Don't modify the response on failure.
    return FilterDataStatus::Continue;
  }

  // Update content headers.
  headers_->setContentLength(error_json.size());
  headers_->setReferenceContentType(
      Envoy::Http::Headers::get().ContentTypeValues.Json);

  // Replace the body with the JSON error.
  Envoy::Buffer::OwnedImpl json_body(error_json);
  body.move(json_body);

  // Unblock the StopIteration in encodeHeaders().
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

Envoy::ProtobufUtil::Status Filter::errorToJson(google::rpc::Status& error,
                                                std::string* json_out) {
  return Envoy::ProtobufUtil::BinaryToJsonString(
      type_helper_->Resolver(),
      Envoy::Grpc::Common::typeUrl(error.GetDescriptor()->full_name()),
      error.SerializeAsString(), json_out, print_options_);
}

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
