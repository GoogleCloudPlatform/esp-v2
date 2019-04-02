// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "envoy/server/filter_config.h"
#include "src/envoy/http/service_control/config_parser.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

/**
 * All stats for the service control filter. @see stats_macros.h
 */

// clang-format off
#define ALL_SERVICE_CONTROL_FILTER_STATS(COUNTER)     \
  COUNTER(allowed)                                    \
  COUNTER(denied)
// clang-format on

/**
 * Wrapper struct for service control filter stats. @see stats_macros.h
 */
struct ServiceControlFilterStats {
  ALL_SERVICE_CONTROL_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};
// The Envoy filter config for API Proxy service control client.
class ServiceControlFilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  ServiceControlFilterConfig(
      const ::google::api::envoy::http::service_control::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())),
        cm_(context.clusterManager()),
        random_(context.random()),
        config_parser_(proto_config_, context) {
    // The default places to extract api-key
    default_api_keys_.add_locations()->set_query("key");
    default_api_keys_.add_locations()->set_query("api_key");
    default_api_keys_.add_locations()->set_header("x-api-key");
  }

  const ::google::api::envoy::http::service_control::FilterConfig& proto()
      const {
    return proto_config_;
  }

  Upstream::ClusterManager& cm() { return cm_; }
  Runtime::RandomGenerator& random() const { return random_; }
  ServiceControlFilterStats& stats() { return stats_; }
  const FilterConfigParser& cfg_parser() const { return config_parser_; }
  const ::google::api::envoy::http::service_control::APIKeyRequirement&
  default_api_keys() const {
    return default_api_keys_;
  }

 private:
  ServiceControlFilterStats generateStats(const std::string& prefix,
                                          Stats::Scope& scope) {
    const std::string final_prefix = prefix + "service_control.";
    return {ALL_SERVICE_CONTROL_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  // The proto config.
  ::google::api::envoy::http::service_control::FilterConfig proto_config_;
  // The stats for the filter.
  ServiceControlFilterStats stats_;
  Upstream::ClusterManager& cm_;
  Runtime::RandomGenerator& random_;
  FilterConfigParser config_parser_;

  ::google::api::envoy::http::service_control::APIKeyRequirement
      default_api_keys_;
};

typedef std::shared_ptr<ServiceControlFilterConfig> FilterConfigSharedPtr;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
