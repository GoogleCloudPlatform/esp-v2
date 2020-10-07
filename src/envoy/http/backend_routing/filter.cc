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

#include "src/envoy/http/backend_routing/filter.h"

#include <string>

#include "absl/strings/string_view.h"
#include "common/common/assert.h"
#include "common/http/headers.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {

namespace {
struct RcDetailsValues {
  // The request doesn't contain PATH header.
  const std::string MissingPath = "backend_routing_bad_request{MISSING_PATH}";

  // The path contains fragment identifiers `#`.
  const std::string FragmentIdentifier =
      "backend_routing_bad_request{PATH_WITH_FRAGMENT_IDENTIFIER}";

  // Missing operation in internal filter state.
  const std::string UndefinedRequest = "backend_routing_undefined_request";
};
using RcDetails = Envoy::ConstSingleton<RcDetailsValues>;
}  // namespace

using Envoy::Http::FilterHeadersStatus;
using ::espv2::api::envoy::v9::http::backend_routing::BackendRoutingRule;

Filter::Filter(FilterConfigSharedPtr config) : config_(config) {}

FilterHeadersStatus Filter::decodeHeaders(
    Envoy::Http::RequestHeaderMap& headers, bool) {
  if (headers.Path() == nullptr) {
    // NOTE: this shouldn't happen in practice because Path Matcher filter would
    // have already rejected the request.
    config_->stats().denied_by_no_path_.inc();
    rejectRequest(Envoy::Http::Code::BadRequest, "No path in request headers",
                  RcDetails::get().MissingPath);
    return FilterHeadersStatus::StopIteration;
  }

  const auto& filter_state = *decoder_callbacks_->streamInfo().filterState();
  absl::string_view operation =
      utils::getStringFilterState(filter_state, utils::kFilterStateOperation);
  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    config_->stats().denied_by_no_operation_.inc();
    rejectRequest(
        Envoy::Http::Code::InternalServerError,
        absl::StrCat("Request `", utils::readHeaderEntry(headers.Method()), " ",
                     utils::readHeaderEntry(headers.Path()),
                     "` is not defined by this API."),
        RcDetails::get().UndefinedRequest);
    return FilterHeadersStatus::StopIteration;
  }

  const auto* rule = config_->findRule(operation);
  if (rule == nullptr) {
    // By design, we only want to apply the filter to operations that are in the
    // configuration. Otherwise, let it pass through (not a dynamic routing
    // request).
    config_->stats().allowed_by_no_configured_rules_.inc();
    ENVOY_LOG(debug,
              "Allow request to pass through, as filter is not configured for "
              "operation: {}",
              operation);
    return FilterHeadersStatus::Continue;
  }

  absl::string_view original_path = headers.Path()->value().getStringView();
  std::string new_path;

  // Reject requests with fragment identifiers. They should never be sent to
  // servers, and it breaks how we handle path translation (query params
  // appended incorrectly).
  if (absl::StrContains(original_path, "#")) {
    config_->stats().denied_by_invalid_path_.inc();
    rejectRequest(Envoy::Http::Code::BadRequest,
                  "Path cannot contain fragment identifier (#)",
                  RcDetails::get().FragmentIdentifier);
    return FilterHeadersStatus::StopIteration;
  }

  switch (rule->path_translation()) {
    case BackendRoutingRule::CONSTANT_ADDRESS:
      new_path = translateConstPath(rule->path_prefix(), original_path);
      config_->stats().constant_address_request_.inc();
      ENVOY_LOG(debug,
                "constant address backend routing for operation {}"
                ", original path: {}, new path: {}",
                operation, original_path, new_path);
      break;

    case BackendRoutingRule::APPEND_PATH_TO_ADDRESS:
      new_path = translateAppendPath(rule->path_prefix(), original_path);
      config_->stats().append_path_to_address_request_.inc();
      ENVOY_LOG(debug,
                "append path to address backend routing for operation {}"
                ", original path: {}, new path: {}",
                operation, original_path, new_path);
      break;

    default:
      NOT_REACHED_GCOVR_EXCL_LINE;
  }

  if (!headers.EnvoyOriginalPath()) {
    headers.setEnvoyOriginalPath(headers.getPathValue());
  }
  headers.setPath(new_path);
  return FilterHeadersStatus::Continue;
}

void Filter::rejectRequest(Envoy::Http::Code code, absl::string_view error_msg,
                           absl::string_view details) {
  ENVOY_LOG(debug, "{}", error_msg);
  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     details);
  decoder_callbacks_->streamInfo().setResponseFlag(
      Envoy::StreamInfo::ResponseFlag::NoRouteFound);
}

// Replace the original path with the constant path prefix.
// If any variable bindings were extracted, append them as query params.
std::string Filter::translateConstPath(absl::string_view prefix,
                                       absl::string_view original_path) {
  const auto& filter_state = *decoder_callbacks_->streamInfo().filterState();
  const absl::string_view extracted_query_params =
      utils::getStringFilterState(filter_state, utils::kFilterStateQueryParams);

  const auto original_path_str = std::string(original_path);
  std::size_t originalQueryParamPos = original_path_str.find('?');
  if (originalQueryParamPos == std::string::npos) {
    // No query param in original request.
    std::string new_path = std::string(prefix);
    if (!extracted_query_params.empty()) {
      // Add extracted variable bindings.
      absl::StrAppend(&new_path, "?", extracted_query_params);
    }
    return new_path;
  }

  // Has query parameters in original request.
  const std::string originalQueryParam =
      original_path_str.substr(originalQueryParamPos);
  std::string new_path = absl::StrCat(prefix, originalQueryParam);
  if (!extracted_query_params.empty()) {
    // Append extracted variable bindings.
    absl::StrAppend(&new_path, "&", extracted_query_params);
  }
  return new_path;
}

// Just append the original request path to the configured path prefix.
// Extracted variable bindings should not be attached as query params.
// If the original path has query params, they will be included.
std::string Filter::translateAppendPath(absl::string_view prefix,
                                        absl::string_view original_path) {
  return absl::StrCat(prefix, original_path);
}

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
