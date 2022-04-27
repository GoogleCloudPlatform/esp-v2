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

#include "api/envoy/v11/http/backend_auth/config.pb.h"
#include "api/envoy/v11/http/backend_auth/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "source/extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/backend_auth/config_parser_impl.h"
#include "src/envoy/http/backend_auth/filter.h"
#include "src/envoy/http/backend_auth/filter_config_impl.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

/**
 * Config registration for ESPv2 backend auth filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v11::http::backend_auth::FilterConfig,
          ::espv2::api::envoy::v11::http::backend_auth::PerRouteFilterConfig> {
 public:
  FilterFactory() : FactoryBase(kFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v11::http::backend_auth::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Envoy::Server::Configuration::FactoryContext& context) override {
    auto filter_config =
        std::make_shared<FilterConfigImpl>(proto_config, stats_prefix, context);
    return [filter_config](
               Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>(filter_config);
      callbacks.addStreamDecoderFilter(
          Envoy::Http::StreamDecoderFilterSharedPtr(filter));
    };
  }

  Envoy::Router::RouteSpecificFilterConfigConstSharedPtr
  createRouteSpecificFilterConfigTyped(
      const ::espv2::api::envoy::v11::http::backend_auth::PerRouteFilterConfig&
          per_route,
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

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
