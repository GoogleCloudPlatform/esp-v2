// Copyright 2019 Google Cloud Platform Proxy Authors
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

#ifndef FILTER_STATE_UTILS_H_
#define FILTER_STATE_UTILS_H_

#include <string>

#include "envoy/stream_info/filter_state.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// Data names in `FilterState` set by Path Matcher filter:
constexpr char kOperation[] = "envoy.filters.http.path_matcher.operation";
constexpr char kQueryParams[] = "envoy.filters.http.path_matcher.query_params";

// Sets a read only string value in the filter state.
void setStringFilterState(Envoy::StreamInfo::FilterState& filter_state,
                          absl::string_view data_name, absl::string_view value);

// Returns a string_view from filter state.
// Returns an empty string_view if the value is not found.
absl::string_view getStringFilterState(
    const Envoy::StreamInfo::FilterState& filter_state,
    absl::string_view data_name);

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy

#endif  // FILTER_STATE_UTILS_H_
