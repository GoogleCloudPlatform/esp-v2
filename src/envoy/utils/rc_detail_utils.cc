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

#include "rc_detail_utils.h"

#include <iostream>

namespace espv2 {
namespace envoy {
namespace utils {

std::string generateRcDetails(const std::string& filter_name,
                              const std::string& error_type,
                              const std::string& error_detail) {
  if (error_detail.length() > 0) {
    return absl::StrCat(filter_name, "_", error_type, "{", error_detail, "}");
  }
  return absl::StrCat(filter_name, "_", error_type);
}
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
