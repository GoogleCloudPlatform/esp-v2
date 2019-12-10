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

#include "api/envoy/http/backend_auth/config.pb.h"
#include "common/common/logger.h"
#include "src/envoy/http/backend_auth/config_parser.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

/**
 * All stats for the backend auth filter. @see stats_macros.h
 */

// clang-format off
#define ALL_BACKEND_AUTH_FILTER_STATS(COUNTER)     \
  COUNTER(token_added)                             \
// clang-format on

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

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
