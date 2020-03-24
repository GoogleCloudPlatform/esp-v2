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

#include "common/http/headers.h"
#include "src/envoy/utils/filter_state_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {

using Envoy::Http::FilterHeadersStatus;

Filter::Filter(FilterConfigSharedPtr config) : config_(config) {}

FilterHeadersStatus Filter::decodeHeaders(
    Envoy::Http::RequestHeaderMap& headers, bool) {
  const auto& filter_state = *decoder_callbacks_->streamInfo().filterState();
  absl::string_view operation =
      utils::getStringFilterState(filter_state, utils::kOperation);
  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    ENVOY_LOG(debug, "No operation found from DynamicMetadata");
    return FilterHeadersStatus::Continue;
  }

  ENVOY_LOG(debug, "Found operation: {}", operation);
  const auto* rule = config_->findRule(operation);
  if (rule == nullptr) {
    ENVOY_LOG(debug, "No backend routing rule found for operation {}",
              operation);
    return FilterHeadersStatus::Continue;
  }

  if (headers.Path() == nullptr) {
    ENVOY_LOG(debug, "No path header in request");
    return FilterHeadersStatus::Continue;
  }
  const absl::string_view original_path =
      headers.Path()->value().getStringView();
  ENVOY_LOG(debug, "backend routing for operation {}, original path: {}",
            operation, original_path);
  std::string newPath;

  if (rule->is_const_address()) {  // CONSTANT_ADDRESS
    absl::string_view queryParamFromPathParam =
        utils::getStringFilterState(filter_state, utils::kQueryParams);
    const auto originalPath = std::string(original_path);
    std::size_t originalQueryParamPos = originalPath.find('?');
    if (originalQueryParamPos != std::string::npos) {
      // has query parameters in original url
      const std::string& originalQueryParam =
          originalPath.substr(originalQueryParamPos);
      newPath = absl::StrCat(rule->path_prefix(), originalQueryParam);
      if (!queryParamFromPathParam.empty()) {
        absl::StrAppend(&newPath, "&", queryParamFromPathParam);
      }
    } else {
      newPath = rule->path_prefix();
      if (!queryParamFromPathParam.empty()) {
        absl::StrAppend(&newPath, "?", queryParamFromPathParam);
      }
    }
    config_->stats().constant_address_request_.inc();
    ENVOY_LOG(debug,
              "constant address backend routing for operation {}, new path: {}",
              operation, newPath);
  } else {  // APPEND_PATH_TO_ADDRESS
    newPath = absl::StrCat(rule->path_prefix(), original_path);
    config_->stats().append_path_to_address_request_.inc();
    ENVOY_LOG(
        debug,
        "append path to address backend routing for operation {}, new path: {}",
        operation, newPath);
  }
  const auto& pathField = Envoy::Http::Headers::get().Path;
  headers.remove(pathField);
  headers.addCopy(pathField, newPath);

  return FilterHeadersStatus::Continue;
}

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
