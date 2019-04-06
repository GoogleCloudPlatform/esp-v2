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

#include "api/envoy/http/service_control/config.pb.h"
#include "envoy/thread_local/thread_local.h"
#include "google/api/service.pb.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/client_cache.h"
#include "src/envoy/http/service_control/service_control_call.h"
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
    static const std::string* const kEmptyToken = new std::string;
    return (token_) ? *token_ : *kEmptyToken;
  }

  ClientCache& client_cache() { return client_cache_; }

 private:
  TokenSharedPtr token_;
  ClientCache client_cache_;
};

class ServiceControlCallImpl : public ServiceControlCall,
                               public Utils::TokenSubscriber::Callback,
                               public Logger::Loggable<Logger::Id::filter> {
 public:
  ServiceControlCallImpl(
      const ::google::api::envoy::http::service_control::Service& config,
      Server::Configuration::FactoryContext& context);

  void callCheck(
      const ::google::api_proxy::service_control::CheckRequestInfo& request,
      CheckDoneFunc on_done) override;

  void callReport(const ::google::api_proxy::service_control::ReportRequestInfo&
                      request) override;

 private:
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

  const ::google::api::envoy::http::service_control::Service& config_;
  std::unique_ptr<::google::api_proxy::service_control::RequestBuilder>
      request_builder_;
  ThreadLocal::SlotPtr tls_;
  Utils::TokenSubscriber token_subscriber_;
};

class ServiceControlCallFactoryImpl : public ServiceControlCallFactory {
 public:
  ServiceControlCallFactoryImpl(Server::Configuration::FactoryContext& context)
      : context_(context) {}

  ServiceControlCallPtr create(
      const ::google::api::envoy::http::service_control::Service& config)
      override {
    return std::make_unique<ServiceControlCallImpl>(config, context_);
  }

 private:
  Server::Configuration::FactoryContext& context_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
