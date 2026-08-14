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

#include "api/envoy/v12/http/trace_context/config.pb.h"
#include "envoy/http/filter.h"
#include "envoy/http/header_map.h"
#include "source/common/common/logger.h"
#include "source/extensions/filters/http/common/pass_through_filter.h"

#include "source/common/singleton/const_singleton.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

class TraceContextHeaders {
 public:
  const Envoy::Http::LowerCaseString Traceparent{"traceparent"};
  const Envoy::Http::LowerCaseString CloudTraceContext{"x-cloud-trace-context"};
  const Envoy::Http::LowerCaseString GrpcTraceBin{"grpc-trace-bin"};
};
using TraceContextHeadersSingleton = Envoy::ConstSingleton<TraceContextHeaders>;

class Filter : public Envoy::Http::PassThroughDecoderFilter,
               public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  Filter(std::shared_ptr<const ::envoy::v12::http::trace_context::TraceContextForwardedConfig> config)
      : config_(std::move(config)) {}

  // Envoy::Http::StreamDecoderFilter
  Envoy::Http::FilterHeadersStatus decodeHeaders(Envoy::Http::RequestHeaderMap&,
                                                 bool) override;

 private:
  std::shared_ptr<const ::envoy::v12::http::trace_context::TraceContextForwardedConfig> config_;
};

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
