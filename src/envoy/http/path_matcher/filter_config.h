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
#include "absl/types/optional.h"
#include "api/envoy/v6/http/path_matcher/config.pb.h"
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
  FilterConfig(const ::espv2::api::envoy::v6::http::path_matcher::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Envoy::Server::Configuration::FactoryContext& context);

  const std::string* findOperation(const std::string& http_method,
                                   const std::string& path) const {
    return path_matcher_->Lookup(http_method, path);
  }

  const std::string* findOperation(
      const std::string& http_method, const std::string& path,
      std::vector<espv2::api_proxy::path_matcher::VariableBinding>*
          variable_bindings) const {
    return path_matcher_->Lookup(http_method, path, variable_bindings);
  }

  // Returns whether an operation needs path parameter extraction.
  // If needed, it will also return a map of snake to json segment conversions.
  //
  // NOTE: path parameter extraction is only needed when backend rule path
  // translation is CONSTANT_ADDRESS.
  const ::espv2::api::envoy::v6::http::path_matcher::
      PathParameterExtractionRule*
      needParameterExtraction(const std::string& operation) const {
    auto operation_it = path_param_extractions_.find(operation);
    if (operation_it == path_param_extractions_.end()) {
      return nullptr;
    }

    // Relies on pointer safety in absl::flat_hash_map.
    // The map will never change after filter config is generated, so it should
    // be safe.
    return &(operation_it->second);
  }

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix,
                            Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "path_matcher.";
    return {ALL_PATH_MATCHER_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  ::espv2::api::envoy::v6::http::path_matcher::FilterConfig proto_config_;
  ::espv2::api_proxy::path_matcher::PathMatcherPtr<const std::string*>
      path_matcher_;

  // Map from operation id to a PathParameterExtractionRule.
  // Only stores the operations that need path param extraction.
  absl::flat_hash_map<
      std::string,
      ::espv2::api::envoy::v6::http::path_matcher::PathParameterExtractionRule>
      path_param_extractions_;

  FilterStats stats_;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
