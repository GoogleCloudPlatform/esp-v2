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
#include "envoy/event/dispatcher.h"
#include "envoy/tracing/http_tracer.h"
#include "envoy/upstream/cluster_manager.h"
#include "include/service_control_client.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/filter_stats.h"
#include "src/envoy/http/service_control/http_call.h"
#include "src/envoy/http/service_control/service_control_callback_func.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// The class to cache check and batch report.
class ClientCache : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  ClientCache(
      const ::google::api::envoy::http::service_control::Service& config,
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config,
      ServiceControlFilterStats& filter_stats,
      Envoy::Upstream::ClusterManager& cm, Envoy::TimeSource& time_source,
      Envoy::Event::Dispatcher& dispatcher,
      std::function<const std::string&()> sc_token_fn,
      std::function<const std::string&()> quota_token_fn);

  ~ClientCache() { destruct_mode_ = true; };

  CancelFunc callCheck(
      const ::google::api::servicecontrol::v1::CheckRequest& request,
      Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done);

  void callQuota(
      const ::google::api::servicecontrol::v1::AllocateQuotaRequest& request,
      QuotaDoneFunc on_done);

  void callReport(
      const ::google::api::servicecontrol::v1::ReportRequest& request);

 private:
  void initHttpRequestSetting(
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config);

  void collectCallStatus(CallStatusStats& filter_stats,
                         const ::google::protobuf::util::error::Code& code);

  template <class Response>
  static ::google::protobuf::util::Status processScCallTransportStatus(
      const ::google::protobuf::util::Status& status, Response* resp,
      const std::string& body);

  const ::google::api::envoy::http::service_control::Service& config_;

  // Filter statistics. When service control client is destroyed in worker
  // thread, filter_stats_ may have already been destructed in the main thread,
  // so don't collect stats at this point.
  ServiceControlFilterStats& filter_stats_;

  // Whether the client_cache is being destructed.
  bool destruct_mode_;

  bool network_fail_open_;

  // the configurable timeouts
  uint32_t check_timeout_ms_;
  uint32_t report_timeout_ms_;
  uint32_t quota_timeout_ms_;

  // the configurable retries
  uint32_t check_retries_;
  uint32_t report_retries_;
  uint32_t quota_retries_;

  // the http call factories
  std::unique_ptr<HttpCallFactory> check_call_factory_;
  std::unique_ptr<HttpCallFactory> quota_call_factory_;
  std::unique_ptr<HttpCallFactory> report_call_factory_;

  // When service control client is destroyed, it will flush out some batched
  // reports and call report_transport_func to send them. Since
  // report_transport_func is using some member variables, placing the client_
  // as the last one to make sure it is destroyed first.
  std::unique_ptr<::google::service_control_client::ServiceControlClient>
      client_;

  // Used to retrieve the current time for tracing.
  Envoy::TimeSource& time_source_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
