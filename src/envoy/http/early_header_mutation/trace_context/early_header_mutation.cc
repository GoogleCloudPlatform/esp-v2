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

  // Strip any unsolicited client-supplied stash header to prevent spoofing.
  headers.remove(const_headers.x_espv2_original_traceparent);

  auto original_traceparent = headers.get(const_headers.traceparent);
  if (!original_traceparent.empty()) {
    headers.setCopy(const_headers.x_espv2_original_traceparent,
                    original_traceparent[0]->value().getStringView());
  }

  bool has_traceparent = false;
  for (const auto& format : config_->incoming_contexts()) {
    if (format == ::envoy::v12::http::trace_context::TRACE_CONTEXT) {
      has_traceparent = true;
      break;
    }
  }

  // Evaluate incoming trace contexts in the exact order configured by the
  // user (matching legacy OpenCensus precedence semantics).
  bool found_valid_context = false;
  for (const auto& format : config_->incoming_contexts()) {
    switch (format) {
      case ::envoy::v12::http::trace_context::TRACE_CONTEXT: {
        if (!headers.get(const_headers.traceparent).empty()) {
          found_valid_context = true;
        }
        break;
      }
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
