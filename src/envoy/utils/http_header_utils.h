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

#include "common/http/utility.h"
#include "envoy/http/header_map.h"

namespace espv2 {
namespace envoy {
namespace utils {

// Returns HTTP header value if the entry is set, otherwise empty string.
absl::string_view readHeaderEntry(const Envoy::Http::HeaderEntry* entry);

// Returns HTTP header value if the header is found, otherwise empty string.
absl::string_view extractHeader(const Envoy::Http::HeaderMap& headers,
                                const Envoy::Http::LowerCaseString& header);

// Get the HTTP method to be used for the request. This method understands the
// x-http-method-override header and if present, returns the
// x-http-method-override method. Otherwise, the actual HTTP method is returned.
absl::string_view getRequestHTTPMethodWithOverride(
    absl::string_view originalMethod, const Envoy::Http::HeaderMap& headers);

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
