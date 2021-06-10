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

#include "api/envoy/v9/http/service_control/config.pb.h"
#include "api/envoy/v9/http/service_control/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "source/extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

/**
 * Config registration for ESPv2 service control filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v9::http::service_control::FilterConfig,
          ::espv2::api::envoy::v9::http::service_control::
              PerRouteFilterConfig> {
 public:
  FilterFactory() : FactoryBase(kFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v9::http::service_control::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Envoy::Server::Configuration::FactoryContext& context) override {
    auto filter_config = std::make_shared<ServiceControlFilterConfig>(
        proto_config, stats_prefix, context);
    return [filter_config](
               Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<ServiceControlFilter>(
          filter_config->stats(), filter_config->handler_factory());
      callbacks.addStreamDecoderFilter(
          Envoy::Http::StreamDecoderFilterSharedPtr(filter));
      callbacks.addAccessLogHandler(
          Envoy::AccessLog::InstanceSharedPtr(filter));
    };
  }

  Envoy::Router::RouteSpecificFilterConfigConstSharedPtr
  createRouteSpecificFilterConfigTyped(
      const ::espv2::api::envoy::v9::http::service_control::
          PerRouteFilterConfig& per_route,
      Envoy::Server::Configuration::ServerFactoryContext&,
      Envoy::ProtobufMessage::ValidationVisitor&) override {
    return std::make_shared<PerRouteFilterConfig>(per_route);
  }
};

/**
 * Static registration for the rate limit filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    FilterFactory, Envoy::Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
