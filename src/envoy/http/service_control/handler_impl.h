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

#pragma once

#include <chrono>
#include <string>

#include "common/common/logger.h"
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
  ServiceControlHandlerImpl(const Http::HeaderMap& headers,
                            const StreamInfo::StreamInfo& stream_info,
                            const std::string& uuid,
                            const FilterConfigParser& cfg_parser,
                            std::chrono::system_clock::time_point now =
                                std::chrono::system_clock::now());
  virtual ~ServiceControlHandlerImpl();

  void callCheck(Http::HeaderMap& headers, CheckDoneCallback& callback);

  void callReport(const Http::HeaderMap* request_headers,
                  const Http::HeaderMap* response_headers,
                  const Http::HeaderMap* response_trailers);

  void collectDecodeData(Buffer::Instance& request_data,
                         std::chrono::system_clock::time_point now =
                             std::chrono::system_clock::now());

 private:
  void fillOperationInfo(
      ::google::api_proxy::service_control::OperationInfo& info,
      std::chrono::system_clock::time_point now =
          std::chrono::system_clock::now());
  void prepareReportRequest(
      ::google::api_proxy::service_control::ReportRequestInfo& info);
  void finishCallReport(
      const ::google::api_proxy::service_control::ReportRequestInfo& info);

  bool isConfigured() const { return require_ctx_ != nullptr; }

  bool isCheckRequired() const {
    return !require_ctx_->config().api_key().allow_without_api_key();
  }

  bool hasApiKey() const { return !api_key_.empty(); }

  void onCheckResponse(
      Http::HeaderMap& headers, const ::google::protobuf::util::Status& status,
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

  // This flag is used to mark if the request is aborted before the check
  // callback is returned.
  std::shared_ptr<bool> aborted_;
  uint64_t request_header_size_;

  // Intermediate data for reporting on streaming.
  ::google::api_proxy::service_control::StreamingRequestInfo streaming_info_;
  // Interval timer for sending intermittent reports.
  std::chrono::system_clock::time_point last_reported_;
};

class ServiceControlHandlerFactoryImpl : public ServiceControlHandlerFactory {
 public:
  ServiceControlHandlerFactoryImpl(Runtime::RandomGenerator& random,
                                   const FilterConfigParser& cfg_parser)
      : random_(random), cfg_parser_(cfg_parser) {}

  ServiceControlHandlerPtr createHandler(
      const Http::HeaderMap& headers,
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
