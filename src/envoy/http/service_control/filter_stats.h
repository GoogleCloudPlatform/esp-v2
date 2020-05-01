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

#include "envoy/stats/scope.h"
#include "envoy/stats/stats_macros.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

/**
 * All stats for the service control filter. @see stats_macros.h
 */

// clang-format off
#define ALL_SERVICE_CONTROL_FILTER_STATS(COUNTER, HISTOGRAM)     \
  COUNTER(allowed)                                    \
  COUNTER(denied) \
  HISTOGRAM(request_time, Milliseconds)  \
  HISTOGRAM(backend_time, Milliseconds)  \
  HISTOGRAM(overhead_time, Milliseconds)

// clang-format on

/**
 * Wrapper struct for service control filter stats. @see stats_macros.h
 */
struct ServiceControlFilterStats {
  ALL_SERVICE_CONTROL_FILTER_STATS(GENERATE_COUNTER_STRUCT,
                                   GENERATE_HISTOGRAM_STRUCT)
};

class ServiceControlFilterStatBase {
 public:
  ServiceControlFilterStatBase(const std::string& prefix,
                               Envoy::Stats::Scope& scope)
      : stats_(generateStats(prefix, scope)) {}

  ServiceControlFilterStats& stats() { return stats_; }

 private:
  ServiceControlFilterStats generateStats(const std::string& prefix,
                                          Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "service_control.";
    return {ALL_SERVICE_CONTROL_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix),
        POOL_HISTOGRAM_PREFIX(scope, final_prefix))};
  }

  // The stats for the filter.
  ServiceControlFilterStats stats_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
