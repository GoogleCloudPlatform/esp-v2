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

#include "api/envoy/http/path_matcher/config.pb.h"
#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "envoy/server/filter_config.h"
#include "src/api_proxy/path_matcher/path_matcher.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace PathMatcher {

/**
 * All stats for the path matcher filter. @see stats_macros.h
 */

// clang-format off
#define ALL_BACKEND_AUTH_FILTER_STATS(COUNTER)     \
  COUNTER(allowed)                                 \
  COUNTER(denied)
// clang-format on

/**
 * Wrapper struct for path matcher filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_BACKEND_AUTH_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for ESP V2 path matcher filter.
class FilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfig(const ::google::api::envoy::http::path_matcher::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())) {
    ::google::api_proxy::path_matcher::PathMatcherBuilder<const std::string*>
        pmb;
    for (const auto& rule : proto_config_.rules()) {
      if (!pmb.Register(rule.pattern().http_method(),
                        rule.pattern().uri_template(),
                        /*body_field_path=*/"", &rule.operation())) {
        throw ProtoValidationException("Duplicated pattern", rule.pattern());
      }
      if (rule.extract_path_parameters()) {
        path_params_operations_.insert(rule.operation());
      }
    }
    path_matcher_ = pmb.Build();

    for (const auto& segment_name : proto_config_.segment_names()) {
      snake_to_json_.emplace(segment_name.snake_name(),
                             segment_name.json_name());
    }
  }

  const std::string* FindOperation(const std::string& http_method,
                                   const std::string& path) const {
    return path_matcher_->Lookup(http_method, path);
  }

  const std::string* FindOperation(
      const std::string& http_method, const std::string& path,
      std::vector<google::api_proxy::path_matcher::VariableBinding>*
          variable_bindings) const {
    return path_matcher_->Lookup(http_method, path, variable_bindings);
  }

  // Returns whether an operation needs path parameter extraction.
  // NOTE: path parameter extraction is only needed when backend rule path
  // translation is CONSTANT_ADDRESS.
  bool NeedPathParametersExtraction(const std::string& operation) {
    auto operation_it = path_params_operations_.find(operation);
    return operation_it != path_params_operations_.end();
  }

  FilterStats& stats() { return stats_; }

  // Returns the mapping between snake-case segment name to JSON name.
  const absl::flat_hash_map<std::string, std::string>& snake_to_json() {
    return snake_to_json_;
  }

 private:
  FilterStats generateStats(const std::string& prefix, Stats::Scope& scope) {
    const std::string final_prefix = prefix + "path_matcher.";
    return {ALL_BACKEND_AUTH_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  ::google::api::envoy::http::path_matcher::FilterConfig proto_config_;
  ::google::api_proxy::path_matcher::PathMatcherPtr<const std::string*>
      path_matcher_;
  // Mapping between snake-case segment name to JSON name as specified in
  // `Service.types` (e.g. "foo_bar" -> "fooBar").
  absl::flat_hash_map<std::string, std::string> snake_to_json_;
  absl::flat_hash_set<std::string> path_params_operations_;
  FilterStats stats_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace PathMatcher
}  // namespace HttpFilters
}  // namespace Extensions
}  //  namespace Envoy
