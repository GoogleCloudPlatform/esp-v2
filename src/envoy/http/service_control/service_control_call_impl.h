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
#include "src/envoy/token/token_subscriber_factory_impl.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// Use shared_ptr to do atomic token update.
typedef std::shared_ptr<std::string> TokenSharedPtr;

// The scope for Service Control API
constexpr char kServiceControlScope[] =
    "https://www.googleapis.com/auth/servicecontrol";

class ThreadLocalCache : public Envoy::ThreadLocal::ThreadLocalObject {
 public:
  ThreadLocalCache(
      const ::espv2::api::envoy::http::service_control::Service& config,
      const ::espv2::api::envoy::http::service_control::FilterConfig&
          filter_config,
      Envoy::Upstream::ClusterManager& cm, Envoy::TimeSource& time_source,
      Envoy::Event::Dispatcher& dispatcher)
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
  TokenSharedPtr sc_token_;
  TokenSharedPtr quota_token_;
  ClientCache client_cache_;
};

typedef std::shared_ptr<
    ::espv2::api::envoy::http::service_control::FilterConfig>
    FilterConfigProtoSharedPtr;

class ServiceControlCallImpl
    : public ServiceControlCall,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  ServiceControlCallImpl(
      FilterConfigProtoSharedPtr proto_config,
      const ::espv2::api::envoy::http::service_control::Service& config,
      Envoy::Server::Configuration::FactoryContext& context);

  CancelFunc callCheck(
      const ::espv2::api_proxy::service_control::CheckRequestInfo& request_info,
      Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done) override;

  void callQuota(
      const ::espv2::api_proxy::service_control::QuotaRequestInfo& request_info,
      QuotaDoneFunc on_done) override;

  void callReport(const ::espv2::api_proxy::service_control::ReportRequestInfo&
                      request_info) override;

 private:
  // Get thread local cache object.
  ThreadLocalCache& getTLCache() const {
    return tls_->getTyped<ThreadLocalCache>();
  }

  void createImdsTokenSub();
  void createTokenGen();
  void createIamTokenSub();

  const ::espv2::api::envoy::http::service_control::FilterConfig&
      filter_config_;
  std::unique_ptr<::espv2::api_proxy::service_control::RequestBuilder>
      request_builder_;

  const token::TokenSubscriberFactoryImpl token_subscriber_factory_;

  // Token subscriber used to fetch access token from imds for service control
  token::TokenSubscriberPtr imds_token_sub_;

  // Access Token for iam server
  std::string access_token_for_iam_;
  // Token subscriber used to fetch access token from imds for accessing iam
  token::TokenSubscriberPtr access_token_sub_;
  // Token subscriber used to fetch access token from iam for service control
  token::TokenSubscriberPtr iam_token_sub_;

  token::ServiceAccountTokenPtr sc_token_gen_;
  token::ServiceAccountTokenPtr quota_token_gen_;
  Envoy::ThreadLocal::SlotPtr tls_;
};  // namespace ServiceControl

class ServiceControlCallFactoryImpl : public ServiceControlCallFactory {
 public:
  explicit ServiceControlCallFactoryImpl(
      FilterConfigProtoSharedPtr proto_config,
      Envoy::Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config), context_(context) {}

  ServiceControlCallPtr create(
      const ::espv2::api::envoy::http::service_control::Service& config)
      override {
    return std::make_unique<ServiceControlCallImpl>(proto_config_, config,
                                                    context_);
  }

 private:
  FilterConfigProtoSharedPtr proto_config_;
  Envoy::Server::Configuration::FactoryContext& context_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
