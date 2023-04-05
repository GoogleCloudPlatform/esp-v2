// Copyright 2023 Google LLC
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

#include "api/envoy/v12/http/header_sanitizer/config.pb.h"
#include "api/envoy/v12/http/header_sanitizer/config.pb.validate.h"
#include "envoy/registry/registry.h"
#include "source/extensions/filters/http/common/factory_base.h"
#include "src/envoy/http/header_sanitizer/filter.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace header_sanitizer {

constexpr const char kFilterName[] =
    "com.google.espv2.filters.http.header_sanitizer";

/**
 * Config registration for ESPv2 header sanitizer filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v12::http::header_sanitizer::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(kFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v12::http::header_sanitizer::FilterConfig&,
      const std::string&,
      Envoy::Server::Configuration::FactoryContext&) override {
    return [](Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>();
      callbacks.addStreamDecoderFilter(filter);
    };
  }
};
/**
 * Static registration for the filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    FilterFactory, Envoy::Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace header_sanitizer
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
