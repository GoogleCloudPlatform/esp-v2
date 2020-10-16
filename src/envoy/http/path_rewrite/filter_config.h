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

#include "envoy/stats/scope.h"
#include "src/envoy/http/path_rewrite/config_parser.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

// The filter name.
constexpr const char kFilterName[] =
    "com.google.espv2.filters.http.path_rewrite";

/**
 * All stats for the backend auth filter. @see stats_macros.h
 */
#define ALL_PATH_REWRITE_FILTER_STATS(COUNTER) \
  COUNTER(path_changed)                        \
  COUNTER(path_not_changed)                    \
  COUNTER(denied_by_no_path)                   \
  COUNTER(denied_by_invalid_path)              \
  COUNTER(denied_by_oversize_path)             \
  COUNTER(denied_by_no_route)                  \
  COUNTER(denied_by_url_template_mismatch)

/**
 * Wrapper struct for backend auth filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_PATH_REWRITE_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

class FilterConfig {
 public:
  FilterConfig(const std::string& stats_prefix, Envoy::Stats::Scope& scope)
      : stats_(generateStats(stats_prefix, scope)) {}

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix,
                            Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "path_rewrite.";
    return {ALL_PATH_REWRITE_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  // The stats
  FilterStats stats_;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

class PerRouteFilterConfig : public Envoy::Router::RouteSpecificFilterConfig {
 public:
  PerRouteFilterConfig(ConfigParserPtr config_parser)
      : config_parser_(std::move(config_parser)) {}

  const ConfigParser& config_parser() const { return *config_parser_; }

 private:
  ConfigParserPtr config_parser_;
};

using PerRouteFilterConfigSharedPtr = std::shared_ptr<PerRouteFilterConfig>;

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
