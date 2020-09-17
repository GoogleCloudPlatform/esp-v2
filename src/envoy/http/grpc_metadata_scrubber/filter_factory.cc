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

#include "api/envoy/v8/http/grpc_metadata_scrubber/config.pb.h"
#include "api/envoy/v8/http/grpc_metadata_scrubber/config.pb.validate.h"
#include "src/envoy/http/grpc_metadata_scrubber/filter.h"

#include "envoy/registry/registry.h"
#include "extensions/filters/http/common/factory_base.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace grpc_metadata_scrubber {

constexpr char kGrpcScrubberFilterName[] =
    "com.google.espv2.filters.http.grpc_metadata_scrubber";

/**
 * Config registration for ESPv2 grpc scrubber filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v8::http::grpc_metadata_scrubber::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(kGrpcScrubberFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v8::http::grpc_metadata_scrubber::
          FilterConfig&,
      const std::string& stats_prefix,
      Envoy::Server::Configuration::FactoryContext& context) override {
    auto filter_config = std::make_shared<FilterConfig>(stats_prefix, context);
    return [filter_config](
               Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>(filter_config);
      callbacks.addStreamEncoderFilter(
          Envoy::Http::StreamEncoderFilterSharedPtr(filter));
    };
  }
};
/**
 * Static registration for the filter. @see RegisterFactory.
 */
static Envoy::Registry::RegisterFactory<
    FilterFactory, Envoy::Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace grpc_metadata_scrubber
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
