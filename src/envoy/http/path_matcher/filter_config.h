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

#include <unordered_map>

#include "absl/container/flat_hash_map.h"
#include "api/envoy/v9/http/path_matcher/config.pb.h"
#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "envoy/server/filter_config.h"
#include "src/api_proxy/path_matcher/path_matcher.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {

/**
 * All stats for the path matcher filter. @see stats_macros.h
 */

// clang-format off
#define ALL_PATH_MATCHER_FILTER_STATS(COUNTER)     \
  COUNTER(allowed)                                 \
  COUNTER(denied)
// clang-format on

/**
 * Wrapper struct for path matcher filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_PATH_MATCHER_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for ESPv2 path matcher filter.
class FilterConfig : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  FilterConfig(const ::espv2::api::envoy::v9::http::path_matcher::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Envoy::Server::Configuration::FactoryContext& context);

  const ::espv2::api::envoy::v9::http::path_matcher::PathMatcherRule* findRule(
      const std::string& http_method, const std::string& path) const {
    return path_matcher_->Lookup(http_method, path);
  }

  const ::espv2::api::envoy::v9::http::path_matcher::PathMatcherRule* findRule(
      const std::string& http_method, const std::string& path,
      std::vector<espv2::api_proxy::path_matcher::VariableBinding>*
          variable_bindings) const {
    return path_matcher_->Lookup(http_method, path, variable_bindings);
  }

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix,
                            Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "path_matcher.";
    return {ALL_PATH_MATCHER_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig proto_config_;
  ::espv2::api_proxy::path_matcher::PathMatcherPtr<
      const ::espv2::api::envoy::v9::http::path_matcher::PathMatcherRule*>
      path_matcher_;

  FilterStats stats_;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
