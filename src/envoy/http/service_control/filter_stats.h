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
#include "google/protobuf/stubs/status.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

/**
 * General service control filter stats.
 * For description of each stat, @see the README.md for this filter.
 * @see stats_macros.h
 */
#define FILTER_STATS(COUNTER, HISTOGRAM) \
  COUNTER(allowed)                       \
  COUNTER(allowed_control_plane_fault)   \
  COUNTER(denied)                        \
  COUNTER(denied_control_plane_fault)    \
  COUNTER(denied_consumer_blocked)       \
  COUNTER(denied_consumer_error)         \
  COUNTER(denied_consumer_quota)         \
  COUNTER(denied_producer_error)         \
  HISTOGRAM(request_time, Milliseconds)  \
  HISTOGRAM(backend_time, Milliseconds)  \
  HISTOGRAM(overhead_time, Milliseconds)

/**
 * Service control call status stats.
 * These match the canonical RPC status codes.
 * https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
 * @see stats_macros.h
 */
#define CALL_STATUS_STATS(COUNTER) \
  COUNTER(OK)                      \
  COUNTER(CANCELLED)               \
  COUNTER(UNKNOWN)                 \
  COUNTER(INVALID_ARGUMENT)        \
  COUNTER(DEADLINE_EXCEEDED)       \
  COUNTER(NOT_FOUND)               \
  COUNTER(ALREADY_EXISTS)          \
  COUNTER(PERMISSION_DENIED)       \
  COUNTER(RESOURCE_EXHAUSTED)      \
  COUNTER(FAILED_PRECONDITION)     \
  COUNTER(ABORTED)                 \
  COUNTER(OUT_OF_RANGE)            \
  COUNTER(UNIMPLEMENTED)           \
  COUNTER(INTERNAL)                \
  COUNTER(UNAVAILABLE)             \
  COUNTER(DATA_LOSS)               \
  COUNTER(UNAUTHENTICATED)

/**
 * Wrapper struct for general service control filter stats. @see stats_macros.h
 */
struct FilterStats {
  FILTER_STATS(GENERATE_COUNTER_STRUCT, GENERATE_HISTOGRAM_STRUCT);
};

/**
 * Wrapper struct for service control call status stats. @see stats_macros.h
 */
struct CallStatusStats {
  CALL_STATUS_STATS(GENERATE_COUNTER_STRUCT);
};

/**
 * Wrapper struct for all the stats structs of service control filter .
 */
struct ServiceControlFilterStats {
  // The general filter stats.
  FilterStats filter_;
  // The stats of service control check call status.
  CallStatusStats check_;
  // The stats of service control allocate quota call status.
  CallStatusStats allocate_quota_;
  // The stats of service control report call status.
  CallStatusStats report_;

  // Collect service control call status.
  static void collectCallStatus(CallStatusStats& filter_stats,
                                const absl::StatusCode& code);

  // Create a stat struct.
  static ServiceControlFilterStats create(const std::string& prefix,
                                          Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "service_control.";

    return {{FILTER_STATS(POOL_COUNTER_PREFIX(scope, final_prefix),
                          POOL_HISTOGRAM_PREFIX(scope, final_prefix))},
            {CALL_STATUS_STATS(
                POOL_COUNTER_PREFIX(scope, final_prefix + "check."))},
            {CALL_STATUS_STATS(
                POOL_COUNTER_PREFIX(scope, final_prefix + "allocate_quota."))},
            {CALL_STATUS_STATS(
                POOL_COUNTER_PREFIX(scope, final_prefix + "report."))}};
  }
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
