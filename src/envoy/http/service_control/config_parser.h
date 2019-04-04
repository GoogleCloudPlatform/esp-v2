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

#ifndef ENVOY_SERVICE_CONTROL_RULE_PARSER_H
#define ENVOY_SERVICE_CONTROL_RULE_PARSER_H

#include <unordered_map>

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/requirement.pb.h"
#include "envoy/thread_local/thread_local.h"
#include "google/api/service.pb.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/client_cache.h"
#include "src/envoy/utils/token_subscriber.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// Use shared_ptr to do atomic token update.
typedef std::shared_ptr<std::string> TokenSharedPtr;

class ThreadLocalCache : public ThreadLocal::ThreadLocalObject {
 public:
  ThreadLocalCache(
      const ::google::api::envoy::http::service_control::Service& config,
      Upstream::ClusterManager& cm, Event::Dispatcher& dispatcher)
      : client_cache_(config, cm, dispatcher,
                      [this]() -> const std::string& { return token(); }) {}

  void set_token(TokenSharedPtr token) { token_ = token; }

  const std::string& token() const {
    static std::string empty_str;
    return (token_) ? *token_ : empty_str;
  }

  ClientCache& client_cache() { return client_cache_; }

 private:
  TokenSharedPtr token_;
  ClientCache client_cache_;
};

class ServiceContext : public Utils::TokenSubscriber::Callback {
 public:
  ServiceContext(
      const ::google::api::envoy::http::service_control::Service& proto_config,
      Server::Configuration::FactoryContext& context);

  const ::google::api::envoy::http::service_control::Service& config() const {
    return filter_service_;
  }

  const ::google::api_proxy::service_control::RequestBuilder& builder() const {
    return *request_builder_;
  }

  // Get thread local cache object.
  ThreadLocalCache& getTLCache() const {
    return tls_->getTyped<ThreadLocalCache>();
  }

  // Utils::TokenSubscriber::Callback function
  void onTokenUpdate(const std::string& token) override {
    TokenSharedPtr new_token = std::make_shared<std::string>(token);
    tls_->runOnAllThreads([this, new_token]() {
      tls_->getTyped<ThreadLocalCache>().set_token(new_token);
    });
  }

 private:
  // The simplified service config defined in Envoy filter
  const ::google::api::envoy::http::service_control::Service& filter_service_;
  // The original service config, but only some fields are copied.
  ::google::api::Service origin_service_;
  std::unique_ptr<::google::api_proxy::service_control::RequestBuilder>
      request_builder_;
  ThreadLocal::SlotPtr tls_;
  Utils::TokenSubscriber token_subscriber_;
};
typedef std::unique_ptr<ServiceContext> ServiceContextPtr;

class RequirementContext {
 public:
  RequirementContext(
      const ::google::api::envoy::http::service_control::Requirement& config,
      const ServiceContext& service_ctx)
      : config_(config), service_ctx_(service_ctx) {}

  const ::google::api::envoy::http::service_control::Requirement& config()
      const {
    return config_;
  }

  const ServiceContext& service_ctx() const { return service_ctx_; }

 private:
  const ::google::api::envoy::http::service_control::Requirement& config_;
  const ServiceContext& service_ctx_;
};
typedef std::unique_ptr<RequirementContext> RequirementContextPtr;

class FilterConfigParser {
 public:
  FilterConfigParser(
      const ::google::api::envoy::http::service_control::FilterConfig& config,
      Server::Configuration::FactoryContext& context);

  const RequirementContext* FindRequirement(absl::string_view operation) const {
    const auto requirement_it = requirements_map_.find(operation);
    if (requirement_it == requirements_map_.end()) {
      return nullptr;
    }
    return requirement_it->second.get();
  }

 private:
  // Operation name to RequirementContext map.
  absl::flat_hash_map<std::string, RequirementContextPtr> requirements_map_;
  // Service name to ServiceContext map.
  absl::flat_hash_map<std::string, ServiceContextPtr> service_map_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

#endif  // ENVOY_SERVICE_CONTROL_RULE_PARSER_H
