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

#include "src/envoy/http/trace_context/filter.h"

#include <string>

#include "envoy/http/header_map.h"
#include "source/common/http/headers.h"
#include "source/common/http/utility.h"
#include "source/common/singleton/const_singleton.h"
#include "source/common/tracing/http_tracer_impl.h"
#include "src/api_proxy/tracing/trace_context_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

using Envoy::Http::FilterHeadersStatus;
using Envoy::Http::RequestHeaderMap;

FilterHeadersStatus Filter::decodeHeaders(RequestHeaderMap& headers, bool) {
  Envoy::Tracing::Span& active_span = decoder_callbacks_->activeSpan();
  const auto& headers_singleton = TraceContextHeadersSingleton::get();

  // Force the tracer to inject context so that the "traceparent" header is 
  // populated in this header map if it hasn't been already.
  Envoy::Tracing::UpstreamContext upstream_context;
  Envoy::Tracing::HttpTraceContext trace_context(headers);
  active_span.injectContext(trace_context, upstream_context);

  auto traceparent_header = headers.get(headers_singleton.Traceparent);
  if (!traceparent_header.empty()) {
    absl::string_view traceparent = traceparent_header[0]->value().getStringView();

    for (int format : config_->outgoing_contexts()) {
      switch (format) {
        case ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT: {
          absl::optional<std::string> cloud_trace = 
              espv2::api_proxy::tracing::TraceContextUtils::TraceParentToXCloudTraceContext(traceparent);
          if (cloud_trace.has_value()) {
            headers.setCopy(headers_singleton.CloudTraceContext, cloud_trace.value());
          }
          break;
        }
        case ::envoy::v12::http::trace_context::GRPC_TRACE_BIN: {
          absl::optional<std::string> grpc_trace = 
              espv2::api_proxy::tracing::TraceContextUtils::TraceParentToGrpcTraceBin(traceparent);
          if (grpc_trace.has_value()) {
            headers.setCopy(headers_singleton.GrpcTraceBin, grpc_trace.value());
          }
          break;
        }
        default:
          break;
      }
    }
  }

  return FilterHeadersStatus::Continue;
}

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
