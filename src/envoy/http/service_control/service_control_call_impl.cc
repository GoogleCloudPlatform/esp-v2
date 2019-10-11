#include <memory>

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

#include "src/api_proxy/service_control/logs_metrics_loader.h"
#include "src/envoy/http/service_control/service_control_call_impl.h"

using Envoy::Extensions::Utils::ServiceAccountToken;
using Envoy::Extensions::Utils::TokenSubscriber;
using ::google::api::envoy::http::common::AccessToken;
using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::api::envoy::http::service_control::Service;
using ::google::api_proxy::service_control::LogsMetricsLoader;
using ::google::api_proxy::service_control::RequestBuilder;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

namespace {
// The service_control service name. used for as audience to generate JWT token.
constexpr char kServiceControlService[] =
    "/google.api.servicecontrol.v1.ServiceController";

// The quota_control service name. used for as audience to generate JWT token.
constexpr char kQuotaControlService[] =
    "/google.api.servicecontrol.v1.QuotaController";

}  // namespace

ServiceControlCallImpl::ServiceControlCallImpl(
    const Service& config, const FilterConfig& filter_config,
    Server::Configuration::FactoryContext& context)
    : config_(config),
      filter_config_(filter_config),
      tls_(context.threadLocal().allocateSlot()) {
  tls_->set([this, &cm = context.clusterManager(),
             &time_source = context.timeSource()](Event::Dispatcher& dispatcher)
                -> ThreadLocal::ThreadLocalObjectSharedPtr {
    return std::make_shared<ThreadLocalCache>(config_, filter_config_, cm,
                                              time_source, dispatcher);
  });

  switch (filter_config.access_token().token_type_case()) {
    case AccessToken::kRemoteToken: {
      const std::string& token_uri =
          filter_config.access_token().remote_token().uri();
      const std::string& token_cluster =
          filter_config.access_token().remote_token().cluster();
      token_sub_ptr_ = std::make_unique<TokenSubscriber>(
          context, token_cluster, token_uri,
          /*json_response=*/true, [this](const std::string& token) {
            TokenSharedPtr new_token = std::make_shared<std::string>(token);
            tls_->runOnAllThreads([this, new_token]() {
              tls_->getTyped<ThreadLocalCache>().set_sc_token(new_token);
              tls_->getTyped<ThreadLocalCache>().set_quota_token(new_token);
            });
          });
    } break;
    case AccessToken::kServiceAccountSecret: {
      const std::string service_control_auidence =
          filter_config.service_control_uri().uri() + kServiceControlService;
      sc_token_gen_ptr_ = std::make_unique<ServiceAccountToken>(
          context,
          filter_config.access_token().service_account_secret().inline_string(),
          service_control_auidence, [this](const std::string& token) {
            TokenSharedPtr new_token = std::make_shared<std::string>(token);
            tls_->runOnAllThreads([this, new_token]() {
              tls_->getTyped<ThreadLocalCache>().set_sc_token(new_token);
            });
          });

      const std::string quota_audience =
          filter_config.service_control_uri().uri() + kQuotaControlService;
      quota_token_gen_ptr_ = std::make_unique<ServiceAccountToken>(
          context,
          filter_config.access_token().service_account_secret().inline_string(),
          quota_audience, [this](const std::string& token) {
            TokenSharedPtr new_token = std::make_shared<std::string>(token);
            tls_->runOnAllThreads([this, new_token]() {
              tls_->getTyped<ThreadLocalCache>().set_quota_token(new_token);
            });
          });
    } break;
    default:
      ENVOY_LOG(error, "No access token set!");
      break;
  }

  if (config.has_service_config()) {
    ::google::api::Service origin_service;
    if (!config.service_config().UnpackTo(&origin_service)) {
      throw ProtoValidationException("Invalid service config", config);
    }

    std::set<std::string> logs, metrics, labels;
    (void)LogsMetricsLoader::Load(origin_service, &logs, &metrics, &labels);
    request_builder_.reset(new RequestBuilder(logs, metrics, labels,
                                              config.service_name(),
                                              config.service_config_id()));
  } else {
    request_builder_.reset(new RequestBuilder(
        {"endpoints_log"}, config.service_name(), config.service_config_id()));
  }
}  // namespace ServiceControl

void ServiceControlCallImpl::callCheck(
    const ::google::api_proxy::service_control::CheckRequestInfo& request_info,
    Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done) {
  ::google::api::servicecontrol::v1::CheckRequest request;
  (void)request_builder_->FillCheckRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending check : {}", request.DebugString());
  getTLCache().client_cache().callCheck(request, parent_span, on_done);
}

void ServiceControlCallImpl::callQuota(
    const ::google::api_proxy::service_control::QuotaRequestInfo& request_info,
    QuotaDoneFunc on_done) {
  ::google::api::servicecontrol::v1::AllocateQuotaRequest request;
  (void)request_builder_->FillAllocateQuotaRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending allocateQuota : {}", request.DebugString());
  getTLCache().client_cache().callQuota(request, on_done);
}

void ServiceControlCallImpl::callReport(
    const ::google::api_proxy::service_control::ReportRequestInfo&
        request_info) {
  ::google::api::servicecontrol::v1::ReportRequest request;
  (void)request_builder_->FillReportRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending report : {}", request.DebugString());
  getTLCache().client_cache().callReport(request);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
