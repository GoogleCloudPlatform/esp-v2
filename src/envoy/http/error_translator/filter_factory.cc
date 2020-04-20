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

#include "api/envoy/http/error_translator/config.pb.h"
#include "api/envoy/http/error_translator/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/error_translator/filter.h"
#include "src/envoy/http/error_translator/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

const std::string FilterName = "envoy.filters.http.error_translator";

/**
 * Config registration for ESPv2 backend routing filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::google::api::envoy::http::error_translator::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(FilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::google::api::envoy::http::error_translator::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Envoy::Server::Configuration::FactoryContext& context) override {
    auto filter_config =
        std::make_shared<FilterConfig>(proto_config, stats_prefix, context);
    return [filter_config](
               Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>(filter_config);
      callbacks.addStreamEncoderFilter(
          Envoy::Http::StreamEncoderFilterSharedPtr(filter));
    };
  }
};
/**
 * Static registration for the rate limit filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    FilterFactory, Envoy::Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
