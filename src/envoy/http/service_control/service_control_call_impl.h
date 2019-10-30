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

#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "envoy/server/filter_config.h"
#include "envoy/thread_local/thread_local.h"
#include "envoy/upstream/cluster_manager.h"
#include "google/api/service.pb.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/client_cache.h"
#include "src/envoy/http/service_control/service_control_call.h"
#include "src/envoy/utils/service_account_token.h"
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
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config,
      Upstream::ClusterManager& cm, Envoy::TimeSource& time_source,
      Event::Dispatcher& dispatcher)
      : client_cache_(
            config, filter_config, cm, time_source, dispatcher,
            [this]() -> const std::string& { return sc_token(); },
            [this]() -> const std::string& { return quota_token(); }) {}

  void set_sc_token(TokenSharedPtr sc_token) { sc_token_ = sc_token; }
  const std::string& sc_token() const {
    return (sc_token_) ? *sc_token_ : empty_token();
  }

  void set_quota_token(TokenSharedPtr quota_token) {
    quota_token_ = quota_token;
  }
  const std::string& quota_token() const {
    return (quota_token_) ? *quota_token_ : empty_token();
  }

  const std::string& empty_token() const {
    static const std::string* const kEmptyToken = new std::string;
    return *kEmptyToken;
  }

  ClientCache& client_cache() { return client_cache_; }

 private:
  TokenSharedPtr token_;
  TokenSharedPtr sc_token_;
  TokenSharedPtr quota_token_;
  ClientCache client_cache_;
};

class ServiceControlCallImpl : public ServiceControlCall,
                               public Logger::Loggable<Logger::Id::filter> {
 public:
  ServiceControlCallImpl(
      const ::google::api::envoy::http::service_control::Service& config,
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config,
      Server::Configuration::FactoryContext& context);

  void callCheck(const ::google::api_proxy::service_control::CheckRequestInfo&
                     request_info,
                 Envoy::Tracing::Span& parent_span,
                 CheckDoneFunc on_done) override;

  void callQuota(const ::google::api_proxy::service_control::QuotaRequestInfo&
                     request_info,
                 QuotaDoneFunc on_done) override;

  void callReport(const ::google::api_proxy::service_control::ReportRequestInfo&
                      request_info) override;

 private:
  // Get thread local cache object.
  ThreadLocalCache& getTLCache() const {
    return tls_->getTyped<ThreadLocalCache>();
  }

  const ::google::api::envoy::http::service_control::Service& config_;
  const ::google::api::envoy::http::service_control::FilterConfig&
      filter_config_;
  std::unique_ptr<::google::api_proxy::service_control::RequestBuilder>
      request_builder_;
  Utils::TokenSubscriberPtr token_sub_ptr_;
  Utils::ServiceAccountTokenPtr sc_token_gen_ptr_;
  Utils::ServiceAccountTokenPtr quota_token_gen_ptr_;
  ThreadLocal::SlotPtr tls_;
};  // namespace ServiceControl

class ServiceControlCallFactoryImpl : public ServiceControlCallFactory {
 public:
  explicit ServiceControlCallFactoryImpl(
      Server::Configuration::FactoryContext& context)
      : context_(context) {}

  ServiceControlCallPtr create(
      const ::google::api::envoy::http::service_control::Service& config,
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config) override {
    return std::make_unique<ServiceControlCallImpl>(config, filter_config,
                                                    context_);
  }

 private:
  Server::Configuration::FactoryContext& context_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
