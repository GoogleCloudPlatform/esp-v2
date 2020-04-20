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

#include "common/http/headers.h"
#include "external/envoy/include/envoy/stream_info/_virtual_includes/stream_info_interface/envoy/stream_info/stream_info.h"
#include "src/envoy/utils/filter_state_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

using ::Envoy::Http::FilterHeadersStatus;

Filter::Filter(FilterConfigSharedPtr config) : config_(config) {}

// TODO(nareddyt): encodeHeaders or encodeData?
// Probably need both for `handleEnvoyFilterError`
FilterHeadersStatus Filter::encodeHeaders(Envoy::Http::ResponseHeaderMap&,
                                          bool) {
  if (utils::hasErrorFilterState(
          *encoder_callbacks_->streamInfo().filterState())) {
    // Enhanced handling of espv2 filter errors.
    handleEspv2FilterError();
    return FilterHeadersStatus::Continue;
  }

  if (encoder_callbacks_->streamInfo().responseCodeDetails() !=
      Envoy::StreamInfo::ResponseCodeDetails::get().ViaUpstream) {
    // Basic handling of non-espv2 upstream envoy filter errors.
    handleEnvoyFilterError();
    return FilterHeadersStatus::Continue;
  }

  // Normal reply, not an error.
  return FilterHeadersStatus::Continue;
}

void Filter::handleEspv2FilterError() {
  google::rpc::Status error = utils::getErrorFilterState(
      *encoder_callbacks_->streamInfo().filterState());

  if (config_->shouldScrubDebugDetails()) {
    // This is a copy, will not affect the filter state.
    error.clear_details();
  }

  // TODO(nareddyt): respond with JSON.
}

void Filter::handleEnvoyFilterError() {
  /*
   * TODO(nareddyt)
   *
   * - Read the envoy filter error text from:
   * -> Body for HTTP/JSON response
   * -> `grpc-message` header for gRPC response (maybe this can be ignored?)
   *
   * - Also need to get the status code from:
   * -> HTTP status header for HTTP/JSON response
   * -> `grpc-status` header for gRPC response (maybe this can be ignored?)
   *
   * - Create a basic google.rcp.Status from the message and code.
   *
   * - Set the google.rpc.Status in the filter state so access log can log it.
   *
   * - Respond with JSON.
   */
}

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
