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

#include "common/http/utility.h"
#include "common/protobuf/utility.h"

#include <string>

namespace Envoy {
namespace Extensions {
namespace Utils {

constexpr char kPathMatcherFilterName[] = "envoy.filters.http.path_matcher";

void setOperationToMetadata(::envoy::api::v2::core::Metadata& metadata, const std::string& operation);
const std::string& getOperationFromMetadata(const ::envoy::api::v2::core::Metadata& metadata,
                                            const std::string& default_value);

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy

#endif  // METADATA_UTILS_H_