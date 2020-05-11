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
#define ALL_SERVICE_CONTROL_FILTER_STATS(COUNTER, HISTOGRAM)     \
  COUNTER(allowed)                                    \
  COUNTER(denied)                          \
  COUNTER(check_count_0) \
  COUNTER(check_count_1) \
  COUNTER(check_count_2) \
  COUNTER(check_count_3) \
  COUNTER(check_count_4) \
  COUNTER(check_count_5) \
  COUNTER(check_count_6) \
  COUNTER(check_count_7) \
  COUNTER(check_count_8) \
  COUNTER(check_count_9) \
  COUNTER(check_count_10) \
  COUNTER(check_count_11) \
  COUNTER(check_count_12) \
  COUNTER(check_count_13) \
  COUNTER(check_count_14) \
  COUNTER(check_count_15) \
  COUNTER(check_count_16) \
  COUNTER(report_count_0) \
  COUNTER(report_count_1) \
  COUNTER(report_count_2) \
  COUNTER(report_count_3) \
  COUNTER(report_count_4) \
  COUNTER(report_count_5) \
  COUNTER(report_count_6) \
  COUNTER(report_count_7) \
  COUNTER(report_count_8) \
  COUNTER(report_count_9) \
  COUNTER(report_count_10) \
  COUNTER(report_count_11) \
  COUNTER(report_count_12) \
  COUNTER(report_count_13) \
  COUNTER(report_count_14) \
  COUNTER(report_count_15) \
  COUNTER(report_count_16) \
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
  // Collect check call status.
  static void collectCheckStatus(
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
