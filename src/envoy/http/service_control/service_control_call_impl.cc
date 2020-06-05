#include <memory>

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

#include "src/api_proxy/service_control/logs_metrics_loader.h"
#include "src/envoy/http/service_control/service_control_call_impl.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

using ::espv2::api::envoy::http::common::AccessToken;
using ::espv2::api::envoy::http::service_control::FilterConfig;
using ::espv2::api::envoy::http::service_control::Service;
using ::espv2::api_proxy::service_control::LogsMetricsLoader;
using ::espv2::api_proxy::service_control::RequestBuilder;
using token::ServiceAccountTokenGenerator;
using token::TokenSubscriber;
using token::TokenType;

namespace {
// The service_control service name. used for as audience to generate JWT token.
constexpr char kServiceControlService[] =
    "/google.api.servicecontrol.v1.ServiceController";

// The quota_control service name. used for as audience to generate JWT token.
constexpr char kQuotaControlService[] =
    "/google.api.servicecontrol.v1.QuotaController";

}  // namespace

void ServiceControlCallImpl::createImdsTokenSub() {
  const std::string& token_cluster = filter_config_.imds_token().cluster();
  const std::string& token_uri = filter_config_.imds_token().uri();
  imds_token_sub_ = token_subscriber_factory_.createImdsTokenSubscriber(
      TokenType::AccessToken, token_cluster, token_uri,
      [this](absl::string_view token) {
        TokenSharedPtr new_token = std::make_shared<std::string>(token);
        tls_->runOnAllThreads([this, new_token]() {
          tls_->getTyped<ThreadLocalCache>().set_sc_token(new_token);
          tls_->getTyped<ThreadLocalCache>().set_quota_token(new_token);
        });
      });
}

void ServiceControlCallImpl::createTokenGen() {
  const std::string service_control_auidence =
      filter_config_.service_control_uri().uri() + kServiceControlService;
  sc_token_gen_ = token_subscriber_factory_.createServiceAccountTokenGenerator(
      filter_config_.service_account_secret().inline_string(),
      service_control_auidence, [this](const std::string& token) {
        TokenSharedPtr new_token = std::make_shared<std::string>(token);
        tls_->runOnAllThreads([this, new_token]() {
          tls_->getTyped<ThreadLocalCache>().set_sc_token(new_token);
        });
      });

  const std::string quota_audience =
      filter_config_.service_control_uri().uri() + kQuotaControlService;
  quota_token_gen_ =
      token_subscriber_factory_.createServiceAccountTokenGenerator(
          filter_config_.service_account_secret().inline_string(),
          quota_audience, [this](const std::string& token) {
            TokenSharedPtr new_token = std::make_shared<std::string>(token);
            tls_->runOnAllThreads([this, new_token]() {
              tls_->getTyped<ThreadLocalCache>().set_quota_token(new_token);
            });
          });
}

void ServiceControlCallImpl::createIamTokenSub() {
  switch (filter_config_.iam_token().access_token().token_type_case()) {
    case AccessToken::kRemoteToken: {
      const std::string& cluster =
          filter_config_.iam_token().access_token().remote_token().cluster();
      const std::string& uri =
          filter_config_.iam_token().access_token().remote_token().uri();
      access_token_sub_ = token_subscriber_factory_.createImdsTokenSubscriber(
          TokenType::AccessToken, cluster, uri,
          [this](absl::string_view access_token) {
            access_token_for_iam_ = std::string(access_token);
          });
      break;
    }
    default: {
      throw Envoy::EnvoyException(
          "Not support getting access token for iam server by "
          "service account file");
    }
  }
  const std::string& token_cluster =
      filter_config_.iam_token().iam_uri().cluster();
  const std::string& token_uri = filter_config_.iam_token().iam_uri().uri();
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  scopes.Add(kServiceControlScope);
  iam_token_sub_ = token_subscriber_factory_.createIamTokenSubscriber(
      TokenType::AccessToken, token_cluster, token_uri,
      [this](absl::string_view token) {
        TokenSharedPtr new_token = std::make_shared<std::string>(token);
        tls_->runOnAllThreads([this, new_token]() {
          tls_->getTyped<ThreadLocalCache>().set_sc_token(new_token);
          tls_->getTyped<ThreadLocalCache>().set_quota_token(new_token);
        });
      },
      filter_config_.iam_token().delegates(), scopes,
      [this]() { return access_token_for_iam_; });
}

ServiceControlCallImpl::ServiceControlCallImpl(
    FilterConfigProtoSharedPtr proto_config, const Service& config,
    ServiceControlFilterStats& filter_stats,
    Envoy::Server::Configuration::FactoryContext& context)
    : filter_config_(*proto_config),
      token_subscriber_factory_(context),
      tls_(context.threadLocal().allocateSlot()) {
  // Pass shared_ptr of proto_config to the function capture so that
  // it will not be released when the function is called.
  tls_->set([proto_config, &config, &filter_stats,
             &cm = context.clusterManager(),
             &time_source =
                 context.timeSource()](Envoy::Event::Dispatcher& dispatcher)
                -> Envoy::ThreadLocal::ThreadLocalObjectSharedPtr {
    return std::make_shared<ThreadLocalCache>(
        config, *proto_config, filter_stats, cm, time_source, dispatcher);
  });

  switch (filter_config_.access_token_case()) {
    case FilterConfig::kImdsToken: {
      createImdsTokenSub();
    } break;
    case FilterConfig::kServiceAccountSecret: {
      createTokenGen();
    } break;
    case FilterConfig::kIamToken: {
      createIamTokenSub();
    } break;
    default:
      ENVOY_LOG(error, "No access token set!");
      break;
  }

  if (config.has_service_config()) {
    std::set<std::string> logs, metrics, labels;
    (void)LogsMetricsLoader::Load(config.service_config(), &logs, &metrics,
                                  &labels);
    request_builder_.reset(new RequestBuilder(logs, metrics, labels,
                                              config.service_name(),
                                              config.service_config_id()));
  } else {
    request_builder_.reset(new RequestBuilder(
        {"endpoints_log"}, config.service_name(), config.service_config_id()));
  }
}  // namespace ServiceControl

CancelFunc ServiceControlCallImpl::callCheck(
    const ::espv2::api_proxy::service_control::CheckRequestInfo& request_info,
    Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done) {
  ::google::api::servicecontrol::v1::CheckRequest request;
  (void)request_builder_->FillCheckRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending check : {}", request.DebugString());
  return getTLCache().client_cache().callCheck(request, parent_span, on_done);
}

void ServiceControlCallImpl::callQuota(
    const ::espv2::api_proxy::service_control::QuotaRequestInfo& request_info,
    QuotaDoneFunc on_done) {
  ::google::api::servicecontrol::v1::AllocateQuotaRequest request;
  (void)request_builder_->FillAllocateQuotaRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending allocateQuota : {}", request.DebugString());
  getTLCache().client_cache().callQuota(request, on_done);
}

void ServiceControlCallImpl::callReport(
    const ::espv2::api_proxy::service_control::ReportRequestInfo&
        request_info) {
  ::google::api::servicecontrol::v1::ReportRequest request;
  (void)request_builder_->FillReportRequest(request_info, &request);
  ENVOY_LOG(debug, "Sending report : {}", request.DebugString());
  getTLCache().client_cache().callReport(request);
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
