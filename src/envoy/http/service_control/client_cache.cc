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

#include "src/envoy/http/service_control/client_cache.h"

#include "common/tracing/http_tracer_impl.h"
#include "src/api_proxy/service_control/request_builder.h"

using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

using ::espv2::api_proxy::service_control::CheckResponseInfo;
using ::google::api::servicecontrol::v1::AllocateQuotaRequest;
using ::google::api::servicecontrol::v1::AllocateQuotaResponse;
using ::google::api::servicecontrol::v1::CheckRequest;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::ReportRequest;
using ::google::api::servicecontrol::v1::ReportResponse;

using ::google::service_control_client::CheckAggregationOptions;
using ::google::service_control_client::QuotaAggregationOptions;
using ::google::service_control_client::ReportAggregationOptions;
using ::google::service_control_client::ServiceControlClientOptions;
using ::google::service_control_client::TransportDoneFunc;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace {

// Default config for check aggregator
constexpr uint32_t kCheckAggregationEntries = 10000;
// Check doesn't support quota yet. It is safe to increase
// the cache life of check results.
// Cache life is 5 minutes. It will be refreshed every minute.
constexpr uint32_t kCheckAggregationFlushIntervalMs = 60000;
constexpr uint32_t kCheckAggregationExpirationMs = 300000;

// Default config for quota aggregator
constexpr uint32_t kQuotaAggregationEntries = 10000;
constexpr uint32_t kQuotaAggregationFlushIntervalMs = 1000;

// Default config for report aggregator
constexpr uint32_t kReportAggregationEntries = 10000;
constexpr uint32_t kReportAggregationFlushIntervalMs = 1000;

// The default connection timeout for check requests.
constexpr uint32_t kCheckDefaultTimeoutInMs = 1000;
// The default connection timeout for allocate quota requests.
constexpr uint32_t kAllocateQuotaDefaultTimeoutInMs = 1000;
// The default connection timeout for report requests.
constexpr uint32_t kReportDefaultTimeoutInMs = 2000;

// The default number of retries for check calls.
constexpr uint32_t kCheckDefaultNumberOfRetries = 3;
// The default number of retries for allocate quota calls.
// Allocate quota has fail_open policy, retry once is enough.
constexpr uint32_t kAllocateQuotaDefaultNumberOfRetries = 1;
// The default number of retries for report calls.
constexpr uint32_t kReportDefaultNumberOfRetries = 5;

// The default value for network_fail_open flag.
constexpr bool kDefaultNetworkFailOpen = true;

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
  EnvoyPeriodicTimer(Envoy::Event::Dispatcher& dispatcher, int interval_ms,
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
  Envoy::Event::TimerPtr timer_;
};

}  // namespace

void ClientCache::InitHttpRequestSetting(const FilterConfig& filter_config) {
  if (!filter_config.has_sc_calling_config()) {
    network_fail_open_ = kDefaultNetworkFailOpen;
    check_timeout_ms_ = kCheckDefaultTimeoutInMs;
    quota_timeout_ms_ = kAllocateQuotaDefaultTimeoutInMs;
    report_timeout_ms_ = kReportDefaultTimeoutInMs;
    check_retries_ = kCheckDefaultNumberOfRetries;
    quota_retries_ = kAllocateQuotaDefaultNumberOfRetries;
    report_retries_ = kReportDefaultNumberOfRetries;
    return;
  }
  const auto& sc_calling_config = filter_config.sc_calling_config();
  network_fail_open_ = sc_calling_config.has_network_fail_open()
                           ? sc_calling_config.network_fail_open().value()
                           : true;
  check_timeout_ms_ = sc_calling_config.has_check_timeout_ms()
                          ? sc_calling_config.check_timeout_ms().value()
                          : kCheckDefaultTimeoutInMs;
  quota_timeout_ms_ = sc_calling_config.has_quota_timeout_ms()
                          ? sc_calling_config.quota_timeout_ms().value()
                          : kAllocateQuotaDefaultTimeoutInMs;
  report_timeout_ms_ = sc_calling_config.has_report_timeout_ms()
                           ? sc_calling_config.report_timeout_ms().value()
                           : kReportDefaultTimeoutInMs;

  check_retries_ = sc_calling_config.has_check_retries()
                       ? sc_calling_config.check_retries().value()
                       : kCheckDefaultNumberOfRetries;
  quota_retries_ = sc_calling_config.has_quota_retries()
                       ? sc_calling_config.quota_retries().value()
                       : kAllocateQuotaDefaultNumberOfRetries;
  report_retries_ = sc_calling_config.has_report_retries()
                        ? sc_calling_config.report_retries().value()
                        : kReportDefaultNumberOfRetries;
}

ClientCache::ClientCache(
    const ::google::api::envoy::http::service_control::Service& config,
    const FilterConfig& filter_config, Envoy::Upstream::ClusterManager& cm,
    Envoy::TimeSource& time_source, Envoy::Event::Dispatcher& dispatcher,
    std::function<const std::string&()> sc_token_fn,
    std::function<const std::string&()> quota_token_fn)
    : config_(config), time_source_(time_source) {
  ServiceControlClientOptions options(getCheckAggregationOptions(),
                                      getQuotaAggregationOptions(),
                                      getReportAggregationOptions());

  InitHttpRequestSetting(filter_config);
  check_call_factory_ = std::make_unique<HttpCallFactory>(
      cm, dispatcher, filter_config.service_control_uri(),
      config_.service_name() + ":check", sc_token_fn, check_timeout_ms_,
      check_retries_, time_source, "Service Control remote call: Check");
  quota_call_factory_ = std::make_unique<HttpCallFactory>(
      cm, dispatcher, filter_config.service_control_uri(),
      config_.service_name() + ":allocateQuota", quota_token_fn,
      quota_timeout_ms_, quota_retries_, time_source,
      "Service Control remote call: Allocate Quota");
  report_call_factory_ = std::make_unique<HttpCallFactory>(
      cm, dispatcher, filter_config.service_control_uri(),
      config_.service_name() + ":report", sc_token_fn, report_timeout_ms_,
      report_retries_, time_source, "Service Control remote call: Report");

  options.check_transport = [this](const CheckRequest& request,
                                   CheckResponse* response,
                                   TransportDoneFunc on_done) {
    // Don't support tracing on this transport
    auto& null_span = Envoy::Tracing::NullSpan::instance();
    auto* call = check_call_factory_->createHttpCall(
        request, null_span,
        [response, on_done](const Status& status, const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            ENVOY_LOG(error, "Failed to call check, error: {}, str body: {}",
                      status.ToString(), body);
          }
          on_done(status);
        });
    call->call();
  };

  options.quota_transport = [this](const AllocateQuotaRequest& request,
                                   AllocateQuotaResponse* response,
                                   TransportDoneFunc on_done) {
    // Don't support tracing on this transport
    auto& null_span = Envoy::Tracing::NullSpan::instance();
    auto* call = quota_call_factory_->createHttpCall(
        request, null_span,
        [response, on_done](const Status& status, const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            ENVOY_LOG(error,
                      "Failed to call allocateQuota, error: {}, str body: {}",
                      status.ToString(), body);
          }
          on_done(status);
        });
    call->call();
  };

  options.report_transport = [this](const ReportRequest& request,
                                    ReportResponse* response,
                                    TransportDoneFunc on_done) {
    // Don't support tracing on this transport
    auto& null_span = Envoy::Tracing::NullSpan::instance();
    auto* call = report_call_factory_->createHttpCall(
        request, null_span,
        [response, on_done](const Status& status, const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            ENVOY_LOG(error, "Failed to call report, error: {}, str body: {}",
                      status.ToString(), body);
          }
          on_done(status);
        });
    call->call();
  };

  options.periodic_timer = [&dispatcher](int interval_ms,
                                         std::function<void()> callback)
      -> std::unique_ptr<::google::service_control_client::PeriodicTimer> {
    return std::unique_ptr<::google::service_control_client::PeriodicTimer>(
        new EnvoyPeriodicTimer(dispatcher, interval_ms, callback));
  };

  client_ = ::google::service_control_client::CreateServiceControlClient(
      config_.service_name(), config_.service_config_id(), options);
}

CancelFunc ClientCache::callCheck(
    const CheckRequest& request, Envoy::Tracing::Span& parent_span,
    std::function<void(const Status&, const CheckResponseInfo&)> on_done) {
  CancelFunc cancel_fn;
  auto check_transport = [this, &parent_span, &cancel_fn](
                             const CheckRequest& request,
                             CheckResponse* response,
                             TransportDoneFunc on_done) {
    auto* call = check_call_factory_->createHttpCall(
        request, parent_span,
        [response, on_done](const Status& status, const std::string& body) {
          if (status.ok()) {
            // Handle 200 response
            if (!response->ParseFromString(body)) {
              on_done(Status(Code::INVALID_ARGUMENT,
                             std::string("Invalid response")));
              return;
            }
          } else {
            ENVOY_LOG(error, "Failed to call check, error: {}, str body: {}",
                      status.ToString(), body);
          }
          on_done(status);
        });
    call->call();
    cancel_fn = [call]() { call->cancel(); };
  };

  parent_span.log(time_source_.systemTime(),
                  "Service Control cache query: Check");

  auto* response = new CheckResponse;
  client_->Check(
      request, response,
      [this, response, on_done](const Status& status) {
        CheckResponseInfo response_info;
        if (status.ok()) {
          Status converted_status = ::espv2::api_proxy::service_control::
              RequestBuilder::ConvertCheckResponse(
                  *response, config_.service_name(), &response_info);
          on_done(converted_status, response_info);
        } else if (status.error_code() == Code::UNAVAILABLE) {
          // Envoy::Grpc::httpToGrpcStatus() is called at http_call.cc at
          // HttpCallImpl::onSuccess to map http_code to grpc_code.
          // All 5xx server error codes have been mapped to Code::UNAVAILABLE.
          // network_fail_open only applies to 5xx server error codes.
          if (network_fail_open_) {
            ENVOY_LOG(debug,
                      "service control check fails, but the request is allowed "
                      "due to network_fail_open policy.");
            on_done(Status::OK, response_info);
          } else {
            // Preserve the original 5xx error code in the response back.
            on_done(status, response_info);
          }
        } else {
          // This is not caused by a client request error, so translate non-5xx
          // error codes to 500 Internal Server Error. Error message contains
          // details on the original error (including the original HTTP status
          // code).
          Status converted(Code::INTERNAL, status.error_message());
          on_done(converted, response_info);
        }
        delete response;
      },
      check_transport);
  return cancel_fn;
}

void ClientCache::callQuota(
    const ::google::api::servicecontrol::v1::AllocateQuotaRequest& request,
    std::function<void(const ::google::protobuf::util::Status& status)>
        on_done) {
  auto* response = new AllocateQuotaResponse;
  client_->Quota(
      request, response, [this, response, on_done](const Status& status) {
        if (status.ok()) {
          on_done(::espv2::api_proxy::service_control::RequestBuilder::
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
  auto* response = new ReportResponse;
  client_->Report(request, response,
                  [response](const Status&) { delete response; });
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
