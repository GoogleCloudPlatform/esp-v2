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
#pragma once

#include <string>
#include <vector>

#include "absl/container/flat_hash_map.h"
#include "absl/strings/str_split.h"
#include "src/api_proxy/path_matcher/path_matcher.h"

namespace google {
namespace api_proxy {
namespace path_matcher {

// Converts `VariableBinding`s to a query parameter string.
// For example, given the following `VariableBinding`s and
// snake-cased to JSON map {"foo_bar": "fooBar"}:
//  * {"foo_bar"}  : "42"
//  * {"foo", "bar"} : "42"
//  * {"a", "b", "c"}: "xyz"
// it returns "fooBar=42&foo.bar=42&a.b.c=xyz".
const std::string VariableBindingsToQueryParameters(
    const std::vector<google::api_proxy::path_matcher::VariableBinding>&
        variable_bindings,
    const absl::flat_hash_map<std::string, std::string>& snake_to_json);

}  // namespace path_matcher
}  // namespace api_proxy
}  // namespace google
