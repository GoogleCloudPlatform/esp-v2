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

#include "api/envoy/v7/http/service_control/config.pb.h"
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

// Forward declare friend class to test private functions.
namespace test {
class ClientCacheCheckResponseTest;
class ClientCacheCheckResponseErrorTypeTest;
class ClientCacheQuotaResponseTest;
class ClientCacheQuotaResponseErrorTypeTest;
class ClientCacheHttpRequestTest;
}  // namespace test

// The class to cache check and batch report.
class ClientCache : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  ClientCache(
      const ::espv2::api::envoy::v7::http::service_control::Service& config,
      const ::espv2::api::envoy::v7::http::service_control::FilterConfig&
          filter_config,
      const std::string& stats_prefix, Envoy::Stats::Scope& scope,
      Envoy::Upstream::ClusterManager& cm, Envoy::TimeSource& time_source,
      Envoy::Event::Dispatcher& dispatcher,
      std::function<const std::string&()> sc_token_fn,
      std::function<const std::string&()> quota_token_fn);

  CancelFunc callCheck(
      const ::google::api::servicecontrol::v1::CheckRequest& request,
      Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done);

  void callQuota(
      const ::google::api::servicecontrol::v1::AllocateQuotaRequest& request,
      QuotaDoneFunc on_done);

  void callReport(
      const ::google::api::servicecontrol::v1::ReportRequest& request);

 private:
  friend class test::ClientCacheCheckResponseTest;
  friend class test::ClientCacheCheckResponseErrorTypeTest;
  friend class test::ClientCacheQuotaResponseTest;
  friend class test::ClientCacheQuotaResponseErrorTypeTest;
  friend class test::ClientCacheHttpRequestTest;

  // Increments the corresponding stat for the given error type.
  void collectScResponseErrorStats(
      ::espv2::api_proxy::service_control::ScResponseErrorType error_type);

  // Ownership of CheckResponse is passed to this function.
  // The function will always call CheckDoneFunc.
  void handleCheckResponse(
      const ::google::protobuf::util::Status& http_status,
      ::google::api::servicecontrol::v1::CheckResponse* response,
      CheckDoneFunc on_done);

  // Ownership of AllocateQuotaResponse is passed to this function.
  // The function will always call QuotaDoneFunction.
  void handleQuotaOnDone(
      const ::google::protobuf::util::Status& http_status,
      ::google::api::servicecontrol::v1::AllocateQuotaResponse* response,
      QuotaDoneFunc on_done);

  void initHttpRequestSetting(
      const ::espv2::api::envoy::v7::http::service_control::FilterConfig&
          filter_config);

  void collectCallStatus(CallStatusStats& filter_stats,
                         const ::google::protobuf::util::error::Code& code);

  template <class Response>
  static ::google::protobuf::util::Status processScCallTransportStatus(
      const ::google::protobuf::util::Status& status, Response* resp,
      const std::string& body);

  const ::espv2::api::envoy::v7::http::service_control::Service& config_;

  // Filter statistics.
  ServiceControlFilterStats filter_stats_;

  // network fail policy
  bool network_fail_open_;

  // the configurable timeouts
  uint32_t check_timeout_ms_;
  uint32_t report_timeout_ms_;
  uint32_t quota_timeout_ms_;

  // the configurable retries
  uint32_t check_retries_;
  uint32_t report_retries_;
  uint32_t quota_retries_;

  // Used to retrieve the current time for tracing.
  Envoy::TimeSource& time_source_;

  // The http call factories. On destruction, they automatically cancel all
  // pending RPCs. These should always be close to the last member variables in
  // the class to mitigate use-after-free of other class members (destructor
  // ordering).
  std::unique_ptr<HttpCallFactory> check_call_factory_;
  std::unique_ptr<HttpCallFactory> quota_call_factory_;
  std::unique_ptr<HttpCallFactory> report_call_factory_;

  // The main caching client. On destruction, some cached requests are flushed,
  // calling the transports and making more http calls. Therefore, this should
  // always be the last member of the class (so it's destructed first).
  std::unique_ptr<::google::service_control_client::ServiceControlClient>
      client_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
