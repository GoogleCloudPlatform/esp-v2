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

#include <string>

#include "common/common/logger.h"
#include "envoy/common/pure.h"
#include "envoy/http/header_map.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/filter_config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The request handler to call Check and Report
class Handler : public Logger::Loggable<Logger::Id::filter> {
 public:
  Handler(const Http::HeaderMap& headers, const std::string& operation,
          FilterConfigSharedPtr config);
  virtual ~Handler();

  // Return false if the request is not configured.
  bool isConfigured() const { return require_ctx_ != nullptr; }

  // Return true if Check is required.
  bool isCheckRequired() const {
    return !require_ctx_->config().api_key().allow_without_api_key();
  }

  bool hasApiKey() const { return !api_key_.empty(); }

  class CheckDoneCallback {
   public:
    virtual ~CheckDoneCallback() {}
    virtual void onCheckDone(
        const ::google::protobuf::util::Status& status) PURE;
  };
  // Make an async check call.
  // The headers could be modified by adding some.
  void callCheck(Http::HeaderMap& headers, CheckDoneCallback& callback,
                 const StreamInfo::StreamInfo& stream_info);

  // Make a report call.
  void callReport(const Http::HeaderMap* response_headers,
                  const Http::HeaderMap* response_trailers,
                  const StreamInfo::StreamInfo& stream_info);

 private:
  // Helper functions to extract API key.
  bool extractAPIKeyFromQuery(const Http::HeaderMap& headers,
                              const std::string& query);
  bool extractAPIKeyFromHeader(const Http::HeaderMap& headers,
                               const std::string& header);
  bool extractAPIKeyFromCookie(const Http::HeaderMap& headers,
                               const std::string& cookie);
  bool extractAPIKey(
      const Http::HeaderMap& headers,
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::envoy::http::service_control::APIKeyLocation>&
          locations);
  void fillOperationInfo(
      ::google::api_proxy::service_control::OperationInfo& info);
  void fillGCPInfo(
      ::google::api_proxy::service_control::ReportRequestInfo& info);
  void onCheckResponse(
      Http::HeaderMap& headers, const ::google::protobuf::util::Status& status,
      const ::google::api_proxy::service_control::CheckResponseInfo&
          response_info);

  // The filer config.
  FilterConfigSharedPtr config_;

  // The matched requirement
  const RequirementContext* require_ctx_{};

  // cached parsed query parameters
  bool params_parsed_{false};
  Http::Utility::QueryParams parsed_params_;

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
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
