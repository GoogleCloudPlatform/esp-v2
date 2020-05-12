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
 * All stats for the service control filter. @see stats_macros.h
 */

// clang-format off

#define SERVICE_CONTROL_FILTER_STATS(COUNTER, HISTOGRAM) \
  COUNTER(allowed)                                           \
  COUNTER(denied)                                            \
  HISTOGRAM(request_time, Milliseconds)                      \
  HISTOGRAM(backend_time, Milliseconds)                      \
  HISTOGRAM(overhead_time, Milliseconds)

// TODO(taoxuy): add macro function to init [check|quota|report]_count_STATUS
#define SERVICE_CONTROL_CHECK_STATS(COUNTER, HISTOGRAM) \
  COUNTER(check_count_OK)                               \
  COUNTER(check_count_CANCELLED)                        \
  COUNTER(check_count_UNKNOWN)                          \
  COUNTER(check_count_INVALID_ARGUMENT)                 \
  COUNTER(check_count_DEADLINE_EXCEEDED)                \
  COUNTER(check_count_NOT_FOUND)                        \
  COUNTER(check_count_ALREADY_EXISTS)                   \
  COUNTER(check_count_PERMISSION_DENIED)                \
  COUNTER(check_count_RESOURCE_EXHAUSTED)               \
  COUNTER(check_count_FAILED_PRECONDITION)              \
  COUNTER(check_count_ABORTED)                          \
  COUNTER(check_count_OUT_OF_RANGE)                     \
  COUNTER(check_count_UNIMPLEMENTED)                    \
  COUNTER(check_count_INTERNAL)                         \
  COUNTER(check_count_UNAVAILABLE)                      \
  COUNTER(check_count_DATA_LOSS)                        \
  COUNTER(check_count_UNAUTHENTICATED)

#define SERVICE_CONTROL_QUOTA_STATS(COUNTER, HISTOGRAM) \
  COUNTER(quota_count_OK)                               \
  COUNTER(quota_count_CANCELLED)                        \
  COUNTER(quota_count_UNKNOWN)                          \
  COUNTER(quota_count_INVALID_ARGUMENT)                 \
  COUNTER(quota_count_DEADLINE_EXCEEDED)                \
  COUNTER(quota_count_NOT_FOUND)                        \
  COUNTER(quota_count_ALREADY_EXISTS)                   \
  COUNTER(quota_count_PERMISSION_DENIED)                \
  COUNTER(quota_count_RESOURCE_EXHAUSTED)               \
  COUNTER(quota_count_FAILED_PRECONDITION)              \
  COUNTER(quota_count_ABORTED)                          \
  COUNTER(quota_count_OUT_OF_RANGE)                     \
  COUNTER(quota_count_UNIMPLEMENTED)                    \
  COUNTER(quota_count_INTERNAL)                         \
  COUNTER(quota_count_UNAVAILABLE)                      \
  COUNTER(quota_count_DATA_LOSS)                        \
  COUNTER(quota_count_UNAUTHENTICATED)

#define SERVICE_CONTROL_REPORT_STATS(COUNTER, HISTOGRAM) \
  COUNTER(report_count_OK)                               \
  COUNTER(report_count_CANCELLED)                        \
  COUNTER(report_count_UNKNOWN)                          \
  COUNTER(report_count_INVALID_ARGUMENT)                 \
  COUNTER(report_count_DEADLINE_EXCEEDED)                \
  COUNTER(report_count_NOT_FOUND)                        \
  COUNTER(report_count_ALREADY_EXISTS)                   \
  COUNTER(report_count_PERMISSION_DENIED)                \
  COUNTER(report_count_RESOURCE_EXHAUSTED)               \
  COUNTER(report_count_FAILED_PRECONDITION)              \
  COUNTER(report_count_ABORTED)                          \
  COUNTER(report_count_OUT_OF_RANGE)                     \
  COUNTER(report_count_UNIMPLEMENTED)                    \
  COUNTER(report_count_INTERNAL)                         \
  COUNTER(report_count_UNAVAILABLE)                      \
  COUNTER(report_count_DATA_LOSS)                        \
  COUNTER(report_count_UNAUTHENTICATED)

// clang-format on

/**
 * Wrapper struct for service control filter stats. @see stats_macros.h
 */
struct ServiceControlFilterStats {
  SERVICE_CONTROL_FILTER_STATS(GENERATE_COUNTER_STRUCT,
                               GENERATE_HISTOGRAM_STRUCT);
  SERVICE_CONTROL_CHECK_STATS(GENERATE_COUNTER_STRUCT,
                              GENERATE_HISTOGRAM_STRUCT);
  SERVICE_CONTROL_QUOTA_STATS(GENERATE_COUNTER_STRUCT,
                              GENERATE_HISTOGRAM_STRUCT);
  SERVICE_CONTROL_REPORT_STATS(GENERATE_COUNTER_STRUCT,
                               GENERATE_HISTOGRAM_STRUCT)
  // Collect check call status.
  static void collectCheckStatus(
      ServiceControlFilterStats& filter_stats,
      const ::google::protobuf::util::error::Code& code);

// Collect allocateQuota call status.
  static void collectQuotaStatus(
      ServiceControlFilterStats& filter_stats,
      const ::google::protobuf::util::error::Code& code);

  // Collect report call status.
  static void collectReportStatus(
      ServiceControlFilterStats& filter_stats,
      const ::google::protobuf::util::error::Code& code);
};

class ServiceControlFilterStatBase {
 public:
  ServiceControlFilterStatBase(const std::string& prefix,
                               Envoy::Stats::Scope& scope)
      : stats_(generateStats(prefix, scope)) {}

  ServiceControlFilterStats& stats() { return stats_; }

 private:
  static ServiceControlFilterStats generateStats(const std::string& prefix,
                                                 Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "service_control.";
    // clang-format off
    return {
        SERVICE_CONTROL_FILTER_STATS(
            POOL_COUNTER_PREFIX(scope, final_prefix),
            POOL_HISTOGRAM_PREFIX(scope, final_prefix))
        SERVICE_CONTROL_CHECK_STATS(
            POOL_COUNTER_PREFIX(scope, final_prefix),
            POOL_HISTOGRAM_PREFIX(scope, final_prefix))
        SERVICE_CONTROL_QUOTA_STATS(
            POOL_COUNTER_PREFIX(scope, final_prefix),
            POOL_HISTOGRAM_PREFIX(scope, final_prefix))
        SERVICE_CONTROL_REPORT_STATS(
            POOL_COUNTER_PREFIX(scope, final_prefix),
            POOL_HISTOGRAM_PREFIX(scope, final_prefix))};
    // clang-format on
  }

  // The stats for the filter.
  ServiceControlFilterStats stats_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
