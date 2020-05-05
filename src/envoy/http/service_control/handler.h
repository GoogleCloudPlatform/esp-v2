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

#include "envoy/buffer/buffer.h"
#include "envoy/common/pure.h"
#include "envoy/http/header_map.h"
#include "envoy/stream_info/stream_info.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/filter_stats.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

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
  virtual void callCheck(Envoy::Http::RequestHeaderMap& headers,
                         Envoy::Tracing::Span& parent_span,
                         CheckDoneCallback& callback) PURE;

  // Make a report call.
  virtual void callReport(
      const Envoy::Http::RequestHeaderMap* request_headers,
      const Envoy::Http::ResponseHeaderMap* response_headers,
      const Envoy::Http::ResponseTrailerMap* response_trailers) PURE;

  // If the stream report interval has passed,
  // make an intermediate report call for long-lived gRPC streaming.
  virtual void tryIntermediateReport() PURE;

  // Process the response header to get the information needed for sending
  // intermediate reports.
  virtual void processResponseHeaders(
      const Envoy::Http::ResponseHeaderMap& response_headers) PURE;

  // The request is about to be destroyed need to cancel all async requests.
  virtual void onDestroy() PURE;
};
using ServiceControlHandlerPtr = std::unique_ptr<ServiceControlHandler>;

class ServiceControlHandlerFactory {
 public:
  virtual ~ServiceControlHandlerFactory() = default;

  virtual ServiceControlHandlerPtr createHandler(
      const Envoy::Http::RequestHeaderMap& headers,
      const Envoy::StreamInfo::StreamInfo& stream_info,
      ServiceControlFilterStats& filter_stats) const PURE;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
