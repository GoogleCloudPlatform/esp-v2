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
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/http/backend_auth/filter_config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {
class MockFilterConfigParser : public FilterConfigParser {
 public:
  MOCK_METHOD(absl::string_view, getAudience, (absl::string_view operation),
              (const));

  MOCK_METHOD(const TokenSharedPtr, getJwtToken, (absl::string_view audience),
              (const));
};

class MockFilterConfig : public FilterConfig {
 public:
  MOCK_METHOD(const FilterConfigParser&, cfg_parser, (), (const));

  MOCK_METHOD(FilterStats&, stats, (), ());
};
}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy