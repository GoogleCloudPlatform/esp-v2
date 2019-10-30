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

#include "src/api_proxy/path_matcher/variable_binding_utils.h"

#include "absl/strings/match.h"

namespace google {
namespace api_proxy {
namespace path_matcher {

const std::string VariableBindingsToQueryParameters(
    const std::vector<VariableBinding>& variable_bindings,
    const absl::flat_hash_map<std::string, std::string>& snake_to_json) {
  std::string query_params;
  for (size_t i = 0; i < variable_bindings.size(); i++) {
    const VariableBinding& variable_binding = variable_bindings[i];
    for (size_t j = 0; j < variable_binding.field_path.size(); j++) {
      const std::string& segment = variable_binding.field_path[j];

      // If the segment has JSON name, use JSON name instead.
      if (absl::StrContains(segment, "_")) {
        auto json_name_it = snake_to_json.find(segment);
        if (json_name_it != snake_to_json.end()) {
          query_params.append(json_name_it->second);
        } else {
          query_params.append(segment);
        }
      } else {
        query_params.append(segment);
      }

      if (j < variable_binding.field_path.size() - 1) {
        query_params.append(".");
      }
    }

    query_params.append("=");
    query_params.append(variable_binding.value);
    if (i < variable_bindings.size() - 1) {
      query_params.append("&");
    }
  }
  return query_params;
}

}  // namespace path_matcher
}  // namespace api_proxy
}  // namespace google
