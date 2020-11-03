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
#include "common/http/header_utility.h"
#include "common/http/headers.h"

namespace espv2 {
namespace envoy {
namespace utils {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

namespace {
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
  const auto result =
      Envoy::Http::HeaderUtility::getAllOfHeaderAsString(headers, header);
  if (!result.result().has_value()) {
    return Envoy::EMPTY_STRING;
  }
  return result.result().value();
}

bool handleHttpMethodOverride(Envoy::Http::RequestHeaderMap& headers) {
  const auto entry = headers.get(kHttpMethodOverrideHeader);
  if (entry.empty()) {
    return false;
  }

  // Override can be confusing while debugging, log it.
  absl::string_view method_original = headers.Method()->value().getStringView();
  absl::string_view method_override = entry[0]->value().getStringView();
  ENVOY_LOG_MISC(debug, "Original :method = {}, x-http-method-override = {}",
                 method_original, method_override);

  // Move the header.
  headers.setMethod(method_override);
  headers.remove(kHttpMethodOverrideHeader);
  return true;
}

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
