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

#include "src/envoy/http/service_control/service_control_call_impl.h"

#include "src/api_proxy/service_control/logs_metrics_loader.h"

using ::google::api::envoy::http::service_control::Service;
using ::google::api_proxy::service_control::LogsMetricsLoader;
using ::google::api_proxy::service_control::RequestBuilder;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

namespace {

const char kDefaultTokenUrl[]{
    "http://metadata.google.internal/computeMetadata/v1/instance/"
    "service-accounts/default/token"};

}  // namespace

ServiceControlCallImpl::ServiceControlCallImpl(
    const Service& config, Server::Configuration::FactoryContext& context,
    const std::string& token_url)
    : config_(config),
      tls_(context.threadLocal().allocateSlot()),
      token_subscriber_(context, *this, config.token_cluster(),
                        token_url.empty() ? kDefaultTokenUrl : token_url,
                        /*json_response=*/true) {
  if (config.has_service_config()) {
    ::google::api::Service origin_service;
    if (!config.service_config().UnpackTo(&origin_service)) {
      throw ProtoValidationException("Invalid service config", config);
    }

    std::set<std::string> logs, metrics, labels;
    LogsMetricsLoader::Load(origin_service, &logs, &metrics, &labels);
    request_builder_.reset(new RequestBuilder(logs, metrics, labels,
                                              config.service_name(),
                                              config.service_config_id()));
  } else {
    request_builder_.reset(new RequestBuilder(
        {"endpoints_log"}, config.service_name(), config.service_config_id()));
  }

  tls_->set(
      [this, &cm = context.clusterManager()](Event::Dispatcher& dispatcher)
          -> ThreadLocal::ThreadLocalObjectSharedPtr {
        return std::make_shared<ThreadLocalCache>(config_, cm, dispatcher);
      });
}

void ServiceControlCallImpl::callCheck(
    const ::google::api_proxy::service_control::CheckRequestInfo& info,
    CheckDoneFunc on_done) {
  ::google::api::servicecontrol::v1::CheckRequest request;
  request_builder_->FillCheckRequest(info, &request);
  ENVOY_LOG(debug, "Sending check : {}", request.DebugString());
  getTLCache().client_cache().callCheck(request, on_done);
}

void ServiceControlCallImpl::callQuota(
    const ::google::api_proxy::service_control::QuotaRequestInfo& info,
    QuotaDoneFunc on_done) {
  ::google::api::servicecontrol::v1::AllocateQuotaRequest request;
  request_builder_->FillAllocateQuotaRequest(info, &request);
  ENVOY_LOG(debug, "Sending allocateQuota : {}", request.DebugString());
  getTLCache().client_cache().callQuota(request, on_done);
}

void ServiceControlCallImpl::callReport(
    const ::google::api_proxy::service_control::ReportRequestInfo& info) {
  ::google::api::servicecontrol::v1::ReportRequest request;
  request_builder_->FillReportRequest(info, &request);
  ENVOY_LOG(debug, "Sending report : {}", request.DebugString());
  getTLCache().client_cache().callReport(request);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
