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

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "src/envoy/http/service_control/handler.h"
#include "src/envoy/http/service_control/service_control_call.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class MockServiceControlHandler : public ServiceControlHandler {
 public:
  MOCK_METHOD3(callCheck,
               void(Http::HeaderMap& headers, Envoy::Tracing::Span& parent_span,
                    CheckDoneCallback& callback));

  MOCK_METHOD4(callReport, void(const Http::HeaderMap* request_headers,
                                const Http::HeaderMap* response_headers,
                                const Http::HeaderMap* response_trailers,
                                std::chrono::system_clock::time_point now));

  MOCK_METHOD1(tryIntermediateReport,
               void(std::chrono::system_clock::time_point now));

  MOCK_METHOD1(processResponseHeaders,
               void(const Http::HeaderMap& response_headers));

  MOCK_METHOD0(onDestroy, void());
};

class MockServiceControlHandlerFactory : public ServiceControlHandlerFactory {
 public:
  ServiceControlHandlerPtr createHandler(
      const Http::HeaderMap& headers,
      const StreamInfo::StreamInfo& stream_info) const override {
    return ServiceControlHandlerPtr{createHandler_(headers, stream_info)};
  }

  MOCK_CONST_METHOD2(
      createHandler_,
      ServiceControlHandler*(const Http::HeaderMap& headers,
                             const StreamInfo::StreamInfo& stream_info));
};

class MockServiceControlCall : public ServiceControlCall {
 public:
  MOCK_METHOD3(
      callCheck,
      CancelFunc(
          const ::google::api_proxy::service_control::CheckRequestInfo& request,
          Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done));

  MOCK_METHOD2(
      callQuota,
      void(const ::google::api_proxy::service_control::QuotaRequestInfo& info,
           QuotaDoneFunc on_done));

  MOCK_METHOD1(
      callReport,
      void(const ::google::api_proxy::service_control::ReportRequestInfo&
               request));
};

class MockServiceControlCallFactory : public ServiceControlCallFactory {
 public:
  ServiceControlCallPtr create(
      const ::google::api::envoy::http::service_control::Service& config)
      override {
    return ServiceControlCallPtr{create_(config)};
  }

  MOCK_CONST_METHOD1(
      create_,
      ServiceControlCall*(
          const ::google::api::envoy::http::service_control::Service& config));
};

class MockCheckDoneCallback : public ServiceControlHandler::CheckDoneCallback {
 public:
  MockCheckDoneCallback() {}
  ~MockCheckDoneCallback() {}

  MOCK_METHOD1(onCheckDone, void(const ::google::protobuf::util::Status&));
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
