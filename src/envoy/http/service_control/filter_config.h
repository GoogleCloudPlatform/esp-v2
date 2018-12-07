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
#include "envoy/thread_local/thread_local.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/token_cache.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ThreadLocalCache : public ThreadLocal::ThreadLocalObject {
 public:
  // Load the config from envoy config.
  ThreadLocalCache(
      const ::google::api_proxy::envoy::http::service_control::FilterConfig&
          config,
      Upstream::ClusterManager& cm, TimeSource& time_source) {
    for (const auto& service : config.services()) {
      token_cache_map_[service.service_name()] = std::unique_ptr<TokenCache>(
          new TokenCache(cm, time_source, service.token_cluster()));
    }
  }

  TokenCache* getTokenCacheByServiceName(const std::string& service_name) {
    return token_cache_map_[service_name].get();
  }

 private:
  std::unordered_map<std::string, std::unique_ptr<TokenCache>> token_cache_map_;
};

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

// The Envoy filter config for Cloud ESF service control client.
class FilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfig(
      const ::google::api_proxy::envoy::http::service_control::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())),
        cm_(context.clusterManager()),
        random_(context.random()),
        tls_(context.threadLocal().allocateSlot()),
        builder_({"endpoints_log"}, proto_config_.service_name(),
                 proto_config_.service_config_id()) {
    tls_->set([this](Event::Dispatcher& dispatcher)
                  -> ThreadLocal::ThreadLocalObjectSharedPtr {
      return std::make_shared<ThreadLocalCache>(proto_config_, cm_,
                                                dispatcher.timeSystem());
    });
  }

  const ::google::api_proxy::envoy::http::service_control::FilterConfig&
  config() const {
    return proto_config_;
  }

  Upstream::ClusterManager& cm() { return cm_; }
  Runtime::RandomGenerator& random() { return random_; }
  ::google::api_proxy::service_control::RequestBuilder& builder() {
    return builder_;
  }

  // Get per-thread cache object.
  ThreadLocalCache& getCache() const {
    return tls_->getTyped<ThreadLocalCache>();
  }

  ServiceControlFilterStats& stats() { return stats_; }

 private:
  ServiceControlFilterStats generateStats(const std::string& prefix,
                                          Stats::Scope& scope) {
    const std::string final_prefix = prefix + "service_control.";
    return {ALL_SERVICE_CONTROL_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  // The proto config.
  ::google::api_proxy::envoy::http::service_control::FilterConfig proto_config_;
  // The stats for the filter.
  ServiceControlFilterStats stats_;
  Upstream::ClusterManager& cm_;
  Runtime::RandomGenerator& random_;
  // Thread local slot to store per-thread cache
  ThreadLocal::SlotPtr tls_;
  ::google::api_proxy::service_control::RequestBuilder builder_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
