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

#include <chrono>
#include <string>

#include "common/common/logger.h"
#include "common/grpc/codec.h"
#include "common/grpc/common.h"
#include "envoy/buffer/buffer.h"
#include "envoy/common/random_generator.h"
#include "envoy/common/time.h"
#include "envoy/http/header_map.h"
#include "envoy/http/query_params.h"
#include "envoy/runtime/runtime.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/handler.h"
#include "src/envoy/utils/http_header_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// The request handler to call Check and Report
class ServiceControlHandlerImpl
    : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter>,
      public ServiceControlHandler {
 public:
  ServiceControlHandlerImpl(const Envoy::Http::RequestHeaderMap& headers,
                            const Envoy::StreamInfo::StreamInfo& stream_info,
                            const std::string& uuid,
                            const FilterConfigParser& cfg_parser,
                            Envoy::TimeSource& timeSource,
                            ServiceControlFilterStats& filter_stats);
  ~ServiceControlHandlerImpl() override;

  void callCheck(Envoy::Http::RequestHeaderMap& headers,
                 Envoy::Tracing::Span& parent_span,
                 CheckDoneCallback& callback) override;

  void callReport(
      const Envoy::Http::RequestHeaderMap* request_headers,
      const Envoy::Http::ResponseHeaderMap* response_headers,
      const Envoy::Http::ResponseTrailerMap* response_trailers) override;

  void fillFilterState(::Envoy::StreamInfo::FilterState& filter_state) override;

  void onDestroy() override;

 private:
  absl::string_view getOperationFromPerRoute(
      const Envoy::StreamInfo::StreamInfo& stream_info);

  void callQuota();

  void fillOperationInfo(
      ::espv2::api_proxy::service_control::OperationInfo& info);
  void prepareReportRequest(
      ::espv2::api_proxy::service_control::ReportRequestInfo& info);

  bool isConfigured() const {
    return require_ctx_ != cfg_parser_.non_match_rqm_ctx();
  }

  bool isQuotaRequired() const {
    return !require_ctx_->config().skip_service_control() &&
           !require_ctx_->config().metric_costs().empty();
  }

  bool isCheckRequired() const {
    return !require_ctx_->config().api_key().allow_without_api_key() &&
           !require_ctx_->config().skip_service_control();
  }

  bool isReportRequired() const {
    return !require_ctx_->config().skip_service_control();
  }

  bool hasApiKey() const { return !api_key_.empty(); }

  void onCheckResponse(
      Envoy::Http::RequestHeaderMap& headers,
      const ::google::protobuf::util::Status& status,
      const ::espv2::api_proxy::service_control::CheckResponseInfo&
          response_info);

  // The filter config parser.
  const FilterConfigParser& cfg_parser_;

  // The metadata for the request
  const Envoy::StreamInfo::StreamInfo& stream_info_;

  // timeSource
  Envoy::TimeSource& time_source_;

  // The matched requirement
  const RequirementContext* require_ctx_{};

  std::string path_;
  std::string http_method_;
  std::string uuid_;
  std::string api_key_;

  // The name of headers to send consumer info
  const Envoy::Http::LowerCaseString consumer_type_header_;
  const Envoy::Http::LowerCaseString consumer_number_header_;

  CheckDoneCallback* check_callback_{};
  ::espv2::api_proxy::service_control::CheckResponseInfo check_response_info_;
  ::google::protobuf::util::Status check_status_;

  // The response code detail.
  std::string rc_detail_;

  CancelFunc cancel_fn_;
  bool on_check_done_called_;

  // If true, it is a grpc and need to send multiple reports.
  bool is_grpc_;

  // Filter statistics.
  ServiceControlFilterStats& filter_stats_;
};

class ServiceControlHandlerFactoryImpl : public ServiceControlHandlerFactory {
 public:
  ServiceControlHandlerFactoryImpl(Envoy::Random::RandomGenerator& random,
                                   const FilterConfigParser& cfg_parser,
                                   Envoy::TimeSource& time_source)
      : random_(random), cfg_parser_(cfg_parser), time_source_(time_source) {}

  ServiceControlHandlerPtr createHandler(
      const Envoy::Http::RequestHeaderMap& headers,
      const Envoy::StreamInfo::StreamInfo& stream_info,
      ServiceControlFilterStats& filter_stats) const override {
    return std::make_unique<ServiceControlHandlerImpl>(
        headers, stream_info, random_.uuid(), cfg_parser_, time_source_,
        filter_stats);
  }

 private:
  // Random object.
  Envoy::Random::RandomGenerator& random_;
  // The filter config parser.
  const FilterConfigParser& cfg_parser_;
  // The timeSource
  Envoy::TimeSource& time_source_;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
