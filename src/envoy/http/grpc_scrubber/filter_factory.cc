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

#include "api/envoy/v8/http/grpc_scrubber/config.pb.h"
#include "api/envoy/v8/http/grpc_scrubber/config.pb.validate.h"
#include "src/envoy/http/grpc_scrubber/filter.h"

#include "envoy/registry/registry.h"
#include "extensions/filters/http/common/factory_base.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace grpc_scrubber {

constexpr char kGrpcScrubberFilterName[] =
    "com.google.espv2.filters.http.grpc_scrubber";

/**
 * Config registration for ESPv2 grpc scrubber filter.
 */
class FilterFactory
    : public Envoy::Extensions::HttpFilters::Common::FactoryBase<
          ::espv2::api::envoy::v8::http::grpc_scrubber::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(kGrpcScrubberFilterName) {}

 private:
  Envoy::Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::espv2::api::envoy::v8::http::grpc_scrubber::FilterConfig&,
      const std::string&,
      Envoy::Server::Configuration::FactoryContext&) override {
    return [](Envoy::Http::FilterChainFactoryCallbacks& callbacks) -> void {
      auto filter = std::make_shared<Filter>();
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

}  // namespace grpc_scrubber
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
