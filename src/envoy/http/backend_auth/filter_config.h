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

#include "api/envoy/v6/http/backend_auth/config.pb.h"
#include "common/common/logger.h"
#include "src/envoy/http/backend_auth/config_parser.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

/**
 * All stats for the backend auth filter. @see stats_macros.h
 */
#define ALL_BACKEND_AUTH_FILTER_STATS(COUNTER) \
  COUNTER(denied_by_no_operation)              \
  COUNTER(denied_by_no_token)                  \
  COUNTER(allowed_by_no_configured_rules)      \
  COUNTER(token_added)

/**
 * Wrapper struct for backend auth filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_BACKEND_AUTH_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

class FilterConfig {
 public:
  virtual ~FilterConfig() = default;

  virtual FilterStats& stats() PURE;

  virtual const FilterConfigParser& cfg_parser() const PURE;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
