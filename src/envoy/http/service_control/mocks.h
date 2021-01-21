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
#include "src/envoy/http/service_control/http_call.h"
#include "src/envoy/http/service_control/service_control_call.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

class MockServiceControlHandler : public ServiceControlHandler {
 public:
  MOCK_METHOD(void, callCheck,
              (Envoy::Http::RequestHeaderMap & headers,
               Envoy::Tracing::Span& parent_span, CheckDoneCallback& callback),
              (override));

  MOCK_METHOD(void, callReport,
              (const Envoy::Http::RequestHeaderMap* request_headers,
               const Envoy::Http::ResponseHeaderMap* response_headers,
               const Envoy::Http::ResponseTrailerMap* response_trailers,
               const Envoy::Tracing::Span& parent_span),
              (override));

  MOCK_METHOD(void, onDestroy, (), (override));

  MOCK_METHOD(void, fillFilterState,
              (::Envoy::StreamInfo::FilterState & filter_state), (override));
};

class MockServiceControlHandlerFactory : public ServiceControlHandlerFactory {
 public:
  MOCK_METHOD(ServiceControlHandlerPtr, createHandler,
              (const Envoy::Http::RequestHeaderMap& headers,
               const Envoy::StreamInfo::StreamInfo& stream_info,
               ServiceControlFilterStats& filter_stats),
              (const, override));
};

class MockServiceControlCall : public ServiceControlCall {
 public:
  MOCK_METHOD(
      CancelFunc, callCheck,
      (const ::espv2::api_proxy::service_control::CheckRequestInfo& request,
       Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done),
      (override));

  MOCK_METHOD(
      void, callQuota,
      (const ::espv2::api_proxy::service_control::QuotaRequestInfo& info,
       QuotaDoneFunc on_done),
      (override));

  MOCK_METHOD(
      void, callReport,
      (const ::espv2::api_proxy::service_control::ReportRequestInfo& request),
      (override));
};

class MockServiceControlCallFactory : public ServiceControlCallFactory {
 public:
  MOCK_METHOD(
      ServiceControlCallPtr, create,
      (const ::espv2::api::envoy::v9::http::service_control::Service& config),
      (override));
};

class MockCheckDoneCallback : public ServiceControlHandler::CheckDoneCallback {
 public:
  MockCheckDoneCallback() {}
  ~MockCheckDoneCallback() {}

  MOCK_METHOD(void, onCheckDone,
              (const ::google::protobuf::util::Status&, absl::string_view),
              (override));
};

class MockHttpCall : public HttpCall {
 public:
  MockHttpCall() {}
  ~MockHttpCall() {}

  MOCK_METHOD(void, cancel, (), (override));
  MOCK_METHOD(void, call, (), (override));
};

class MockHttpCallFactory : public HttpCallFactory {
 public:
  MockHttpCallFactory() {}
  ~MockHttpCallFactory() {}

  MOCK_METHOD(HttpCall*, createHttpCall,
              (const Envoy::Protobuf::Message& body,
               Envoy::Tracing::Span& parent_span, HttpCall::DoneFunc on_done));
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
