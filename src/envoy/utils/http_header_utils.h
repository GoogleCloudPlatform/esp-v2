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

#ifndef UTILS_H_
#define UTILS_H_

#include <string>

#include "common/http/utility.h"
#include "envoy/http/header_map.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// Returns HTTP header value if the header is found, otherwise empty string
std::string extractHeader(const Envoy::Http::HeaderMap& headers,
                          const Envoy::Http::LowerCaseString& header);

// Get the HTTP method to be used for the request. This method understands the
// x-http-method-override header and if present, returns the
// x-http-method-override method. Otherwise, the actual HTTP method is returned.
std::string getRequestHTTPMethodWithOverride(
    const std::string& originalMethod, const Envoy::Http::HeaderMap& headers);

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy

#endif  // UTILS_H_
