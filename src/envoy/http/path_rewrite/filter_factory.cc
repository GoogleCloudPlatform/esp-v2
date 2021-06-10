// Copyright 2020 Google LLC
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

#include "api/envoy/v9/http/path_rewrite/config.pb.h"
#include "api/envoy/v9/http/path_rewrite/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "source/extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/path_rewrite/config_parser_impl.h"
#include "src/envoy/http/path_rewrite/filter.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

/**
 * Config registration for ESPv2 path rewrite filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v9::http::path_rewrite::FilterConfig,
          ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig> {
 public:
  FilterFactory() : FactoryBase(kFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v9::http::path_rewrite::FilterConfig&,
      const std::string& stats_prefix,
      Envoy::Server::Configuration::FactoryContext& context) override {
    auto filter_config =
        std::make_shared<FilterConfig>(stats_prefix, context.scope());
    return [filter_config](
               Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>(filter_config);
      callbacks.addStreamDecoderFilter(
          Envoy::Http::StreamDecoderFilterSharedPtr(filter));
    };
  }

  Envoy::Router::RouteSpecificFilterConfigConstSharedPtr
  createRouteSpecificFilterConfigTyped(
      const ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig&
          per_route,
      Envoy::Server::Configuration::ServerFactoryContext&,
      Envoy::ProtobufMessage::ValidationVisitor&) override {
    auto parser = std::make_unique<ConfigParserImpl>(per_route);
    return std::make_shared<PerRouteFilterConfig>(std::move(parser));
  }
};

/**
 * Static registration for the rate limit filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    FilterFactory, Envoy::Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
