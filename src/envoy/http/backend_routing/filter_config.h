// Copyright 2019 Google Cloud Platform Proxy Authors
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
#include "common/common/logger.h"
#include "envoy/server/filter_config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendRouting {

/**
 * All stats for the backend routing filter. @see stats_macros.h
 */

// clang-format off
#define ALL_BACKEND_ROUTING_FILTER_STATS(COUNTER)     \
  COUNTER(append_path_to_address_request)             \
  COUNTER(constant_address_request)                   \
// clang-format on

/**
 * Wrapper struct for backend routing filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_BACKEND_ROUTING_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for API Proxy backend routing filter.
class FilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfig(const ::google::api::envoy::http::backend_routing::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())) {}

  const ::google::api::envoy::http::backend_routing::FilterConfig& config() const {
    return proto_config_;
  }

  FilterStats& stats() { return stats_; }

 private:
  FilterStats generateStats(const std::string& prefix, Stats::Scope& scope) {
    const std::string final_prefix = prefix + "backend_routing.";
    return {ALL_BACKEND_ROUTING_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  ::google::api::envoy::http::backend_routing::FilterConfig proto_config_;
  FilterStats stats_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace BackendRouting
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

