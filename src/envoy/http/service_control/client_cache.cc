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

#include "src/envoy/http/service_control/client_cache.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/http_call.h"

using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

using ::google::api::servicecontrol::v1::AllocateQuotaRequest;
using ::google::api::servicecontrol::v1::AllocateQuotaResponse;
using ::google::api::servicecontrol::v1::CheckRequest;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::ReportRequest;
using ::google::api::servicecontrol::v1::ReportResponse;
using ::google::api_proxy::service_control::CheckResponseInfo;

using ::google::service_control_client::CheckAggregationOptions;
using ::google::service_control_client::QuotaAggregationOptions;
using ::google::service_control_client::ReportAggregationOptions;
using ::google::service_control_client::ServiceControlClient;
using ::google::service_control_client::ServiceControlClientOptions;
using ::google::service_control_client::TransportDoneFunc;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

// Default config for check aggregator
const int kCheckAggregationEntries = 10000;
// Check doesn't support quota yet. It is safe to increase
// the cache life of check results.
// Cache life is 5 minutes. It will be refreshed every minute.
const int kCheckAggregationFlushIntervalMs = 60000;
const int kCheckAggregationExpirationMs = 300000;

// Default config for quota aggregator
const int kQuotaAggregationEntries = 10000;
const int kQuotaAggregationFlushIntervalMs = 1000;

// Default config for report aggregator
const int kReportAggregationEntries = 10000;
const int kReportAggregationFlushIntervalMs = 1000;

// The default connection timeout for check requests.
const int kCheckDefaultTimeoutInMs = 5000;

// Generates CheckAggregationOptions.
CheckAggregationOptions getCheckAggregationOptions() {
  return CheckAggregationOptions(kCheckAggregationEntries,
                                 kCheckAggregationFlushIntervalMs,
                                 kCheckAggregationExpirationMs);
}

// Generates QuotaAggregationOptions.
QuotaAggregationOptions getQuotaAggregationOptions() {
  return QuotaAggregationOptions(kQuotaAggregationEntries,
                                 kQuotaAggregationFlushIntervalMs);
}

// Generates ReportAggregationOptions.
ReportAggregationOptions getReportAggregationOptions() {
  return ReportAggregationOptions(kReportAggregationEntries,
                                  kReportAggregationFlushIntervalMs);
}

// A timer object to wrap PeriodicTimer
class EnvoyPeriodicTimer
    : public ::google::service_control_client::PeriodicTimer {
 public:
  EnvoyPeriodicTimer(Event::Dispatcher& dispatcher, int interval_ms,
                     std::function<void()> callback)
      : interval_ms_(interval_ms),
        callback_(callback),
        timer_(dispatcher.createTimer([this]() { call(); })) {
    timer_->enableTimer(std::chrono::milliseconds(interval_ms_));
  }

  void call() {
    callback_();
    timer_->enableTimer(std::chrono::milliseconds(interval_ms_));
  }

  // Cancels the timer.
  virtual void Stop() override { timer_.reset(); }

 private:
  int interval_ms_;
  std::function<void()> callback_;
  Event::TimerPtr timer_;
};

}  // namespace

ClientCache::ClientCache(
    const ::google::api::envoy::http::service_control::Service& config,
    Upstream::ClusterManager& cm, Event::Dispatcher& dispatcher,
    std::function<const std::string&()> token_fn)
    : config_(config),
      cm_(cm),
      dispatcher_(dispatcher),
      token_fn_(token_fn),
      network_fail_open_(config.network_fail_open()) {
  ServiceControlClientOptions options(getCheckAggregationOptions(),
                                      getQuotaAggregationOptions(),
                                      getReportAggregationOptions());

  options.check_transport = [this](const CheckRequest& request,
                                   CheckResponse* response,
                                   TransportDoneFunc on_done) {
    const std::string& token = token_fn_();
    if (token.empty()) {
      on_done(
          Status(Code::UNAUTHENTICATED,
                 std::string("Missing access token for service control call")));
      return;
    }
    auto* call = HttpCall::create(cm_, config_.service_control_uri());
    call->call(
        config_.service_name() + ":check", token, request,
        [this, response, on_done](const Status& status,
                                  const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            response->ParseFromString(body);
            ENVOY_LOG(
                error,
                "Failed to call check, error: {}, str body: {}, pb body: {}",
                status.ToString(), body, response->DebugString());
          }
          on_done(status);
        });
  };

  options.quota_transport = [this](const AllocateQuotaRequest& request,
                                   AllocateQuotaResponse* response,
                                   TransportDoneFunc on_done) {
    const std::string& token = token_fn_();
    if (token.empty()) {
      on_done(
          Status(Code::UNAUTHENTICATED,
                 std::string("Missing access token for service control call")));
      return;
    }
    auto* call = HttpCall::create(cm_, config_.service_control_uri());
    call->call(config_.service_name() + ":allocateQuota", token, request,
               [this, response, on_done](const Status& status,
                                         const std::string& body) {
                 if (status.ok()) {
                   // Handle 200 response
                   if (!response->ParseFromString(body)) {
                     on_done(Status(Code::INVALID_ARGUMENT,
                                    std::string("Invalid response")));
                     return;
                   }
                 } else {
                   response->ParseFromString(body);
                   ENVOY_LOG(error,
                             "Failed to call allocateQuota, error: {}, str "
                             "body: {}, pb body: {}",
                             status.ToString(), body, response->DebugString());
                 }
                 on_done(status);
               });
  };

  options.report_transport = [this](const ReportRequest& request,
                                    ReportResponse* response,
                                    TransportDoneFunc on_done) {
    const std::string& token = token_fn_();
    if (token.empty()) {
      on_done(
          Status(Code::UNAUTHENTICATED,
                 std::string("Missing access token for service control call")));
      return;
    }
    auto* call = HttpCall::create(cm_, config_.service_control_uri());
    call->call(
        config_.service_name() + ":report", token, request,
        [this, response, on_done](const Status& status,
                                  const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            response->ParseFromString(body);
            ENVOY_LOG(
                error,
                "Failed to call report, error: {}, str body: {}, pb body: {}",
                status.ToString(), body, response->DebugString());
          }
          on_done(status);
        });
  };

  options.periodic_timer = [this](int interval_ms,
                                  std::function<void()> callback)
      -> std::unique_ptr<::google::service_control_client::PeriodicTimer> {
    return std::unique_ptr<::google::service_control_client::PeriodicTimer>(
        new EnvoyPeriodicTimer(dispatcher_, interval_ms, callback));
  };

  client_ = ::google::service_control_client::CreateServiceControlClient(
      config_.service_name(), config_.service_config_id(), options);
}

void ClientCache::callCheck(
    const CheckRequest& request,
    std::function<void(const Status&, const CheckResponseInfo&)> on_done) {
  CheckResponse* response = new CheckResponse;
  client_->Check(request, response,
                 [this, response, on_done](const Status& status) {
                   CheckResponseInfo response_info;
                   if (status.ok()) {
                     Status status = ::google::api_proxy::service_control::
                         RequestBuilder::ConvertCheckResponse(
                             *response, config_.service_name(), &response_info);
                     on_done(status, response_info);
                   } else {
                     if (network_fail_open_) {
                       on_done(Status::OK, response_info);
                     } else {
                       on_done(status, response_info);
                     }
                   }
                   delete response;
                 });
}

void ClientCache::callQuota(
    const ::google::api::servicecontrol::v1::AllocateQuotaRequest& request,
    std::function<void(const ::google::protobuf::util::Status& status)>
        on_done) {
  AllocateQuotaResponse* response = new AllocateQuotaResponse;
  client_->Quota(
      request, response, [this, response, on_done](const Status& status) {
        if (status.ok()) {
          on_done(::google::api_proxy::service_control::RequestBuilder::
                      ConvertAllocateQuotaResponse(*response,
                                                   config_.service_name()));
        } else {
          on_done(Status(static_cast<google::protobuf::util::error::Code>(
                             status.error_code()),
                         status.error_message()));
        }
        delete response;
      });
}

void ClientCache::callReport(const ReportRequest& request) {
  ReportResponse* response = new ReportResponse;
  client_->Report(request, response,
                  [response](const Status&) { delete response; });
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
