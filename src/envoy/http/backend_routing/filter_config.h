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

#include "api/envoy/http/backend_routing/config.pb.h"
#include "common/common/empty_string.h"
#include "common/common/logger.h"
#include "envoy/server/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {

/**
 * All stats for the backend routing filter. @see stats_macros.h
 */

#define ALL_BACKEND_ROUTING_FILTER_STATS(COUNTER) \
  COUNTER(append_path_to_address_request)         \
  COUNTER(constant_address_request)               \
  COUNTER(denied_by_no_path)                      \
  COUNTER(denied_by_no_operation)                 \
  COUNTER(allowed_by_no_configured_rules)

/**
 * Wrapper struct for backend routing filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_BACKEND_ROUTING_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for ESPv2 backend routing filter.
class FilterConfig : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  FilterConfig(const ::google::api::envoy::http::backend_routing::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Envoy::Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())) {
    for (const auto& rule : proto_config_.rules()) {
      if (rule.path_translation() ==
          ::google::api::envoy::http::backend_routing::BackendRoutingRule::
              PATH_TRANSLATION_UNSPECIFIED) {
        throw Envoy::ProtoValidationException(
            "Path translation for BackendRouting rule must be specified", rule);
      }
      if (rule.path_prefix() == Envoy::EMPTY_STRING) {
        throw Envoy::ProtoValidationException("Path prefix cannot be empty",
                                              rule);
      }
      if (!Envoy::Http::validHeaderString(rule.path_prefix())) {
        throw Envoy::ProtoValidationException(
            "Path prefix contains invalid characters", rule);
      }
      backend_routing_map_[rule.operation()] = &rule;
    }
  }

  const ::google::api::envoy::http::backend_routing::BackendRoutingRule*
  findRule(absl::string_view operation) const {
    const auto it = backend_routing_map_.find(operation);
    if (it == backend_routing_map_.end()) {
      return nullptr;
    }
    return it->second;
  }

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix,
                            Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "backend_routing.";
    return {ALL_BACKEND_ROUTING_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  // The config proto
  ::google::api::envoy::http::backend_routing::FilterConfig proto_config_;
  // The stats
  FilterStats stats_;
  // The map from operation to rule.
  absl::flat_hash_map<
      std::string,
      const ::google::api::envoy::http::backend_routing::BackendRoutingRule*>
      backend_routing_map_;
};

using FilterConfigSharedPtr = std::shared_ptr<FilterConfig>;

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
