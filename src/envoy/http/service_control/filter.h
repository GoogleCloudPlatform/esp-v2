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

#include "common/common/logger.h"
#include "envoy/access_log/access_log.h"
#include "envoy/http/filter.h"
#include "envoy/http/header_map.h"
#include "extensions/filters/http/common/pass_through_filter.h"
#include "src/envoy/http/service_control/filter_stats.h"
#include "src/envoy/http/service_control/handler.h"

#include <string>

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// The Envoy filter for ESPv2 service control client.
class ServiceControlFilter
    : public Envoy::Http::PassThroughDecoderFilter,
      public Envoy::AccessLog::Instance,
      public ServiceControlHandler::CheckDoneCallback,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  ServiceControlFilter(ServiceControlFilterStats& stats,
                       const ServiceControlHandlerFactory& factory)
      : stats_(stats), factory_(factory) {}

  void onDestroy() override;

  // Envoy::Http::StreamDecoderFilter
  Envoy::Http::FilterHeadersStatus decodeHeaders(
      Envoy::Http::RequestHeaderMap& headers, bool) override;
  Envoy::Http::FilterDataStatus decodeData(Envoy::Buffer::Instance& data,
                                           bool end_stream) override;
  Envoy::Http::FilterTrailersStatus decodeTrailers(
      Envoy::Http::RequestTrailerMap&) override;

  // Called when the request is completed.
  void log(const Envoy::Http::RequestHeaderMap* request_headers,
           const Envoy::Http::ResponseHeaderMap* response_headers,
           const Envoy::Http::ResponseTrailerMap* response_trailers,
           const Envoy::StreamInfo::StreamInfo& stream_info) override;

  // For Handler::CheckDoneCallback, called when callCheck() is done
  void onCheckDone(const ::google::protobuf::util::Status& status) override;

 private:
  void rejectRequest(Envoy::Http::Code code, absl::string_view error_msg);

  ServiceControlFilterStats& stats_;
  const ServiceControlHandlerFactory& factory_;

  // The service control request handler
  std::unique_ptr<ServiceControlHandler> handler_;

  // The state of the request.
  enum State { Init, Calling, Responded, Complete };
  State state_ = Init;
  // Mark if request has been stopped.
  bool stopped_ = false;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
