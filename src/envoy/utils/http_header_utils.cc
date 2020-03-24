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

#include "src/envoy/utils/http_header_utils.h"
#include "common/common/empty_string.h"

namespace espv2 {
namespace envoy {
namespace utils {

namespace {
// TODO(kyuc): refactor it to be safe, move it to a class or make the type char*
const Envoy::Http::LowerCaseString kHttpMethodOverrideHeader{
    "x-http-method-override"};
}  // namespace

absl::string_view readHeaderEntry(const Envoy::Http::HeaderEntry* entry) {
  if (entry) {
    return entry->value().getStringView();
  }
  return Envoy::EMPTY_STRING;
}

absl::string_view extractHeader(const Envoy::Http::HeaderMap& headers,
                                const Envoy::Http::LowerCaseString& header) {
  const auto* entry = headers.get(header);
  return readHeaderEntry(entry);
}

absl::string_view getRequestHTTPMethodWithOverride(
    absl::string_view originalMethod, const Envoy::Http::HeaderMap& headers) {
  const auto* entry = headers.get(kHttpMethodOverrideHeader);
  if (entry) {
    return entry->value().getStringView();
  }
  return originalMethod;
}

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
