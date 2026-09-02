#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation.h"

#include "envoy/http/header_map.h"
#include "source/common/common/logger.h"
#include "source/common/singleton/const_singleton.h"
#include "src/api_proxy/tracing/trace_context_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

class TraceContextHeaders {
 public:
  const Envoy::Http::LowerCaseString x_cloud_trace_context{
      "x-cloud-trace-context"};
  const Envoy::Http::LowerCaseString grpc_trace_bin{"grpc-trace-bin"};
  const Envoy::Http::LowerCaseString traceparent{"traceparent"};
  const Envoy::Http::LowerCaseString x_espv2_original_traceparent{
      "x-espv2-original-traceparent"};
};

using TraceContextHeadersSingleton = Envoy::ConstSingleton<TraceContextHeaders>;

bool TraceContextEarlyHeaderMutation::mutate(
    Envoy::Http::RequestHeaderMap& headers,
    const Envoy::StreamInfo::StreamInfo&) const {
  const auto& const_headers = TraceContextHeadersSingleton::get();

  auto original_traceparent = headers.get(const_headers.traceparent);
  if (!original_traceparent.empty()) {
    headers.setCopy(const_headers.x_espv2_original_traceparent,
                    original_traceparent[0]->value().getStringView());
  }

  bool has_cloud_trace = false;
  bool has_grpc_trace = false;
  bool has_traceparent = false;
  for (const auto& format : config_->incoming_contexts()) {
    if (format == ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT) {
      has_cloud_trace = true;
    }
    if (format == ::envoy::v12::http::trace_context::GRPC_TRACE_BIN) {
      has_grpc_trace = true;
    }
    if (format == ::envoy::v12::http::trace_context::TRACE_CONTEXT) {
      has_traceparent = true;
    }
  }

  bool found_valid_context = false;

  // 1. W3C traceparent takes absolute priority over legacy headers if
  // configured and present.
  if (has_traceparent && !headers.get(const_headers.traceparent).empty()) {
    found_valid_context = true;
  } else {
    // 2. If W3C traceparent isn't active, fallback to evaluating other
    // configured contexts in priority order.
    for (const auto& format : config_->incoming_contexts()) {
      if (format == ::envoy::v12::http::trace_context::TRACE_CONTEXT) {
        continue;  // Already handled above
      }

      switch (format) {
        case ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT: {
          auto result = headers.get(const_headers.x_cloud_trace_context);
          if (!result.empty()) {
            auto val = espv2::api_proxy::tracing::TraceContextUtils::
                XCloudTraceContextToTraceParent(
                    result[0]->value().getStringView());
            if (val.has_value()) {
              headers.setCopy(const_headers.traceparent, val.value());
              found_valid_context = true;
            }
          }
          break;
        }
        case ::envoy::v12::http::trace_context::GRPC_TRACE_BIN: {
          auto result = headers.get(const_headers.grpc_trace_bin);
          if (!result.empty()) {
            auto val = espv2::api_proxy::tracing::TraceContextUtils::
                GrpcTraceBinToTraceParent(result[0]->value().getStringView());
            if (val.has_value()) {
              headers.setCopy(const_headers.traceparent, val.value());
              found_valid_context = true;
            }
          }
          break;
        }
        default:
          break;
      }
      if (found_valid_context) {
        break;
      }
    }
  }

  // Cleanup unused legacy headers to prevent downstream confusion
  if (has_cloud_trace) {
    headers.remove(const_headers.x_cloud_trace_context);
  }
  if (has_grpc_trace) {
    headers.remove(const_headers.grpc_trace_bin);
  }

  // If no traceparent was securely resolved, force strip any forged/stale
  // values.
  if (!found_valid_context && !has_traceparent) {
    headers.remove(const_headers.traceparent);
  }

  return true;
}

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
