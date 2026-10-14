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

#include "api/envoy/v12/http/trace_context/config.pb.h"
#include "api/envoy/v12/http/trace_context/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "source/extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/trace_context/filter.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

constexpr const char kFilterName[] =
    "com.google.espv2.filters.http.trace_context";

/**
 * Config registration for ESPv2 trace context filter.
 */
class UpstreamFilterFactory
    : public Envoy::Server::Configuration::UpstreamHttpFilterConfigFactory {
 public:
  std::string name() const override { return kFilterName; }

  absl::StatusOr<Envoy::Http::FilterFactoryCb> createFilterFactoryFromProto(
      const Envoy::Protobuf::Message& config,
      const std::string&,
      Envoy::Server::Configuration::UpstreamFactoryContext& context) override {
    auto message = Envoy::MessageUtil::downcastAndValidate<const ::envoy::v12::http::trace_context::TraceContextForwardedConfig&>(
        config, context.serverFactoryContext().messageValidationVisitor());
    auto shared_config = std::make_shared<::envoy::v12::http::trace_context::TraceContextForwardedConfig>(message);

    return [shared_config](Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>(shared_config);
      callbacks.addStreamFilter(filter);
    };
  }

  Envoy::ProtobufTypes::MessagePtr createEmptyConfigProto() override {
    return std::make_unique<::envoy::v12::http::trace_context::TraceContextForwardedConfig>();
  }
};

/**
 * Static registration for the filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    UpstreamFilterFactory, Envoy::Server::Configuration::UpstreamHttpFilterConfigFactory>
    register_upstream_;

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
