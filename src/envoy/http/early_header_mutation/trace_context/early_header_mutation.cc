#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation.h"
#include "src/api_proxy/tracing/trace_context_utils.h"
#include "envoy/http/header_map.h"
#include "source/common/common/logger.h"
#include "source/common/singleton/const_singleton.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

class TraceContextHeaders {
 public:
  const Envoy::Http::LowerCaseString x_cloud_trace_context{"x-cloud-trace-context"};
  const Envoy::Http::LowerCaseString grpc_trace_bin{"grpc-trace-bin"};
  const Envoy::Http::LowerCaseString traceparent{"traceparent"};
};

using TraceContextHeadersSingleton = Envoy::ConstSingleton<TraceContextHeaders>;

bool TraceContextEarlyHeaderMutation::mutate(Envoy::Http::RequestHeaderMap& headers,
                                             const Envoy::StreamInfo::StreamInfo&) const {
  const auto& const_headers = TraceContextHeadersSingleton::get();

  // If W3C traceparent already exists, it has absolute precedence.
  // We simply skip translation and delete any stale legacy headers.
  if (!headers.get(const_headers.traceparent).empty()) {
    headers.remove(const_headers.x_cloud_trace_context);
    headers.remove(const_headers.grpc_trace_bin);
    return true;
  }

  // Iterate over incoming contexts configured by the control plane.
  for (const auto& format : config_->incoming_contexts()) {
    switch (format) {
      case ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT: {
        auto result = headers.get(const_headers.x_cloud_trace_context);
        if (!result.empty()) {
          auto val = espv2::api_proxy::tracing::TraceContextUtils::XCloudTraceContextToTraceParent(
              result[0]->value().getStringView());
          if (val.has_value()) {
            headers.setReferenceKey(const_headers.traceparent, val.value());
          }
        }
        break;
      }
      case ::envoy::v12::http::trace_context::GRPC_TRACE_BIN: {
        auto result = headers.get(const_headers.grpc_trace_bin);
        if (!result.empty()) {
          auto val = espv2::api_proxy::tracing::TraceContextUtils::GrpcTraceBinToTraceParent(
              result[0]->value().getStringView());
          if (val.has_value()) {
            headers.setReferenceKey(const_headers.traceparent, val.value());
          }
        }
        break;
      }
      default:
        break;
    }
    if (!headers.get(const_headers.traceparent).empty()) {
      break;
    }
  }

  // Always aggressively delete legacy headers regardless of whether translation succeeded
  // to avoid leaking stale tracing contexts.
  headers.remove(const_headers.x_cloud_trace_context);
  headers.remove(const_headers.grpc_trace_bin);

  return true;
}

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
