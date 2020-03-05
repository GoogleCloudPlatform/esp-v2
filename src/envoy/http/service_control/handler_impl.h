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
#include "envoy/http/header_map.h"
#include "envoy/http/query_params.h"
#include "envoy/runtime/runtime.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/handler.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The request handler to call Check and Report
class ServiceControlHandlerImpl : public Logger::Loggable<Logger::Id::filter>,
                                  public ServiceControlHandler {
 public:
  ServiceControlHandlerImpl(const Http::RequestHeaderMap& headers,
                            const StreamInfo::StreamInfo& stream_info,
                            const std::string& uuid,
                            const FilterConfigParser& cfg_parser,
                            std::chrono::system_clock::time_point now =
                                std::chrono::system_clock::now());
  ~ServiceControlHandlerImpl() override;

  void callCheck(Http::RequestHeaderMap& headers,
                 Envoy::Tracing::Span& parent_span,
                 CheckDoneCallback& callback) override;

  void callReport(const Http::RequestHeaderMap* request_headers,
                  const Http::ResponseHeaderMap* response_headers,
                  const Http::ResponseTrailerMap* response_trailers,
                  std::chrono::system_clock::time_point now) override;

  void tryIntermediateReport(
      std::chrono::system_clock::time_point now) override;

  void processResponseHeaders(
      const Http::ResponseHeaderMap& response_headers) override;

  void onDestroy() override;

 private:
  void callQuota();

  void fillOperationInfo(
      ::google::api_proxy::service_control::OperationInfo& info);
  void prepareReportRequest(
      ::google::api_proxy::service_control::ReportRequestInfo& info);

  bool isConfigured() const { return require_ctx_ != nullptr; }

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
      Http::RequestHeaderMap& headers,
      const ::google::protobuf::util::Status& status,
      const ::google::api_proxy::service_control::CheckResponseInfo&
          response_info);

  // The filter config parser.
  const FilterConfigParser& cfg_parser_;

  // The metadata for the request
  const StreamInfo::StreamInfo& stream_info_;

  // The matched requirement
  const RequirementContext* require_ctx_{};

  std::string path_;
  std::string http_method_;
  std::string uuid_;
  std::string api_key_;

  CheckDoneCallback* check_callback_{};
  ::google::api_proxy::service_control::CheckResponseInfo check_response_info_;
  ::google::protobuf::util::Status check_status_;

  CancelFunc cancel_fn_;
  bool on_check_done_called_;
  uint64_t request_header_size_;
  uint64_t response_header_size_;

  // The frontend protocol only for intermediate reports.
  ::google::api_proxy::service_control::protocol::Protocol frontend_protocol_;

  // If true, it is a grpc and need to send multiple reports.
  bool is_grpc_;
  // If true, this is the first report.
  bool is_first_report_;
  // Interval timer for sending intermediate reports.
  std::chrono::system_clock::time_point last_reported_;
};

class ServiceControlHandlerFactoryImpl : public ServiceControlHandlerFactory {
 public:
  ServiceControlHandlerFactoryImpl(Runtime::RandomGenerator& random,
                                   const FilterConfigParser& cfg_parser)
      : random_(random), cfg_parser_(cfg_parser) {}

  ServiceControlHandlerPtr createHandler(
      const Http::RequestHeaderMap& headers,
      const StreamInfo::StreamInfo& stream_info) const override {
    return std::make_unique<ServiceControlHandlerImpl>(
        headers, stream_info, random_.uuid(), cfg_parser_);
  }

 private:
  // Random object.
  Runtime::RandomGenerator& random_;
  // The filter config parser.
  const FilterConfigParser& cfg_parser_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
