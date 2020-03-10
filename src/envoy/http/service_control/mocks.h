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
  MOCK_METHOD(void, callCheck,
              (Http::RequestHeaderMap & headers,
               Envoy::Tracing::Span& parent_span, CheckDoneCallback& callback),
              (override));

  MOCK_METHOD(void, callReport,
              (const Http::RequestHeaderMap* request_headers,
               const Http::ResponseHeaderMap* response_headers,
               const Http::ResponseTrailerMap* response_trailers),
              (override));

  MOCK_METHOD(void, tryIntermediateReport, (), (override));

  MOCK_METHOD(void, processResponseHeaders,
              (const Http::ResponseHeaderMap& response_headers), (override));

  MOCK_METHOD(void, onDestroy, (), (override));
};

class MockServiceControlHandlerFactory : public ServiceControlHandlerFactory {
 public:
  MOCK_METHOD(ServiceControlHandlerPtr, createHandler,
              (const Http::RequestHeaderMap& headers,
               const StreamInfo::StreamInfo& stream_info),
              (const, override));
};

class MockServiceControlCall : public ServiceControlCall {
 public:
  MOCK_METHOD(
      CancelFunc, callCheck,
      (const ::google::api_proxy::service_control::CheckRequestInfo& request,
       Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done),
      (override));

  MOCK_METHOD(
      void, callQuota,
      (const ::google::api_proxy::service_control::QuotaRequestInfo& info,
       QuotaDoneFunc on_done),
      (override));

  MOCK_METHOD(
      void, callReport,
      (const ::google::api_proxy::service_control::ReportRequestInfo& request),
      (override));
};

class MockServiceControlCallFactory : public ServiceControlCallFactory {
 public:
  MOCK_METHOD(
      ServiceControlCallPtr, create,
      (const ::google::api::envoy::http::service_control::Service& config),
      (override));
};

class MockCheckDoneCallback : public ServiceControlHandler::CheckDoneCallback {
 public:
  MockCheckDoneCallback() {}
  ~MockCheckDoneCallback() {}

  MOCK_METHOD(void, onCheckDone, (const ::google::protobuf::util::Status&),
              (override));
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
