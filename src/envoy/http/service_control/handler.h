// Copyright 2019 Google Cloud Platform Proxy Authors
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

#include "envoy/buffer/buffer.h"
#include "envoy/common/pure.h"
#include "envoy/http/header_map.h"
#include "envoy/stream_info/stream_info.h"
#include "src/api_proxy/service_control/request_info.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ServiceControlHandler {
 public:
  virtual ~ServiceControlHandler() = default;

  class CheckDoneCallback {
   public:
    virtual ~CheckDoneCallback() = default;
    virtual void onCheckDone(const ::google::protobuf::util::Status&) PURE;
  };

  // Make an async check call.
  // The headers could be modified by adding some.
  virtual void callCheck(Http::HeaderMap& headers,
                         Envoy::Tracing::Span& parent_span,
                         CheckDoneCallback& callback) PURE;

  // Make a report call.
  virtual void callReport(const Http::HeaderMap* request_headers,
                          const Http::HeaderMap* response_headers,
                          const Http::HeaderMap* response_trailers) PURE;

  // Collect decode data, if the stream report interval has passed,
  // make an intermediate report call for long-lived gRPC streaming.
  virtual void collectDecodeData(Buffer::Instance& request_data,
                                 std::chrono::system_clock::time_point now =
                                     std::chrono::system_clock::now()) PURE;

  // Collect encode data, if the stream report interval has passed,
  // make an intermediate report call for long-lived gRPC streaming.
  virtual void collectEncodeData(Buffer::Instance& response_data,
                                 std::chrono::system_clock::time_point now =
                                     std::chrono::system_clock::now()) PURE;
};
typedef std::unique_ptr<ServiceControlHandler> ServiceControlHandlerPtr;

class ServiceControlHandlerFactory {
 public:
  virtual ~ServiceControlHandlerFactory() = default;

  virtual ServiceControlHandlerPtr createHandler(
      const Http::HeaderMap& headers,
      const StreamInfo::StreamInfo& stream_info) const PURE;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
