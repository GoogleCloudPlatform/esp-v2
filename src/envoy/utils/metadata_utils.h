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

#ifndef METADATA_UTILS_H_
#define METADATA_UTILS_H_

#include <string>
#include <vector>

#include "common/http/utility.h"
#include "common/protobuf/utility.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

constexpr char kPathMatcherFilterName[] = "envoy.filters.http.path_matcher";

// TODO(kyuc): add unit tests.

// Field names of the Path Matcher filter metadata:
constexpr char kOperation[] = "operation";
constexpr char kQueryParams[] = "query_params";

// Sets a string value in the PathMatcher Metadata.
void setStringMetadata(::envoy::api::v2::core::Metadata& metadata,
                       const std::string& field_name, const std::string& value);

// Returns a string value in the PathMatcher Metadata.
// Returns an empty string if the value is not found.
const std::string& getStringMetadata(
    const ::envoy::api::v2::core::Metadata& metadata,
    const std::string& field_name);

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy

#endif  // METADATA_UTILS_H_
