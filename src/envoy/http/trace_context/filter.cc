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

Envoy::Http::FilterHeadersStatus Filter::decodeHeaders(
    Envoy::Http::RequestHeaderMap& headers, bool) {
  ENVOY_LOG(trace, "TraceContextFilter::decodeHeaders EXECUTING");

  const auto& const_headers = TraceContextHeadersSingleton::get();

  Envoy::Tracing::Span& active_span = decoder_callbacks_->activeSpan();
  std::string trace_id = active_span.getTraceId();
  std::string span_id = active_span.getSpanId();

  if (!trace_id.empty() && !span_id.empty()) {
    std::string fake_traceparent = "00-" + trace_id + "-" + span_id + "-01";

    for (int format : config_->outgoing_contexts()) {
      switch (format) {
        case ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT: {
          absl::optional<std::string> cloud_trace =
              espv2::api_proxy::tracing::TraceContextUtils::
                  TraceParentToXCloudTraceContext(fake_traceparent);
          if (cloud_trace.has_value()) {
            headers.setCopy(const_headers.CloudTraceContext,
                            cloud_trace.value());
          }
          break;
        }
        case ::envoy::v12::http::trace_context::GRPC_TRACE_BIN: {
          absl::optional<std::string> grpc_trace = espv2::api_proxy::tracing::
              TraceContextUtils::TraceParentToGrpcTraceBin(fake_traceparent);
          if (grpc_trace.has_value()) {
            headers.setCopy(const_headers.GrpcTraceBin, grpc_trace.value());
          }
          break;
        }
      }
    }
  }

  bool wants_traceparent = false;
  for (int format : config_->outgoing_contexts()) {
    if (format == ::envoy::v12::http::trace_context::TRACE_CONTEXT) {
      wants_traceparent = true;
    }
  }

  if (!wants_traceparent) {
    headers.remove(const_headers.Traceparent);
    auto original = headers.get(const_headers.OriginalTraceparent);
    if (!original.empty()) {
      headers.setCopy(const_headers.Traceparent,
                      original[0]->value().getStringView());
    }
  }

  headers.remove(const_headers.OriginalTraceparent);

  return Envoy::Http::FilterHeadersStatus::Continue;
}

Envoy::Http::FilterHeadersStatus Filter::encodeHeaders(
    Envoy::Http::ResponseHeaderMap&, bool) {
  ENVOY_LOG(trace, "TraceContextFilter::encodeHeaders EXECUTING");
  return Envoy::Http::FilterHeadersStatus::Continue;
}
}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
