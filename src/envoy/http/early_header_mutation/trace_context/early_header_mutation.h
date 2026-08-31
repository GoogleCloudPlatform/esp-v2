#pragma once

#include "api/envoy/v12/http/trace_context/config.pb.h"
#include "envoy/http/early_header_mutation.h"
#include "envoy/http/header_map.h"
#include "source/common/common/logger.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

class TraceContextEarlyHeaderMutation
    : public Envoy::Http::EarlyHeaderMutation,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::http> {
 public:
  TraceContextEarlyHeaderMutation(
      std::shared_ptr<const ::envoy::v12::http::trace_context::TraceContextTranslatorConfig> config)
      : config_(std::move(config)) {}

  bool mutate(Envoy::Http::RequestHeaderMap& headers,
              const Envoy::StreamInfo::StreamInfo&) const override;

 private:
  std::shared_ptr<const ::envoy::v12::http::trace_context::TraceContextTranslatorConfig> config_;
};

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
