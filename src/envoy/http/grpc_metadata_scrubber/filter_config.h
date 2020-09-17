// Copyright 2020 Google LLC
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

#include "envoy/server/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace grpc_metadata_scrubber {

/**
 * All stats for the grpc metadata scrubber filter. @see stats_macros.h
 */

// clang-format off
#define ALL_GRPC_METADATA_SCRUBBER_FILTER_STATS(COUNTER)     \
  COUNTER(all)                                 \
  COUNTER(removed)
// clang-format on

/**
 * Wrapper struct for grpc metadata scrubber filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_GRPC_METADATA_SCRUBBER_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for ESPv2 grpc metadata scrubber filter.
class FilterConfig {
 public:
  FilterConfig(const std::string& stats_prefix,
               Envoy::Server::Configuration::FactoryContext& context)
      : stats_(generateStats(stats_prefix, context.scope())) {}

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix,
                            Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "grpc_metadata_scrubber.";
    return {ALL_GRPC_METADATA_SCRUBBER_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  FilterStats stats_;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

}  // namespace grpc_metadata_scrubber
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
