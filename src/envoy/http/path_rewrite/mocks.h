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
#include "gmock/gmock.h"
#include "src/envoy/http/path_rewrite/config_parser.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

class MockConfigParser : public ConfigParser {
 public:
  MOCK_METHOD(bool, rewrite,
              (absl::string_view origin_path, std::string& new_path), (const));
};

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
