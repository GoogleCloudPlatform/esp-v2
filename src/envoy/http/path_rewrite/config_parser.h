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

#include "envoy/common/pure.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

class ConfigParser {
 public:
  virtual ~ConfigParser() = default;

  // If return false, fails to generate new path due to:
  // origin_path doesn't match with the url_template in the const_path.
  virtual bool rewrite(absl::string_view origin_path, std::string& new_path) const PURE;
};

using ConstConfigParserPtr = std::unique_ptr<const ConfigParser>;

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
