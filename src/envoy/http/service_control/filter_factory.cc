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

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/config.pb.validate.h"
#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/filter_config.h"

#include "envoy/registry/registry.h"
#include "extensions/filters/http/common/factory_base.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

const std::string FilterName = "envoy.filters.http.service_control";

/**
 * Config registration for ESP V2 service control filter.
 */
class FilterFactory
    : public Common::FactoryBase<
          ::google::api::envoy::http::service_control::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(FilterName) {}

 private:
  Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::google::api::envoy::http::service_control::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Server::Configuration::FactoryContext& context) override {
    auto filter_config = std::make_shared<ServiceControlFilterConfig>(
        proto_config, stats_prefix, context);
    return
        [filter_config](Http::FilterChainFactoryCallbacks& callbacks) -> void {
          auto filter = std::make_shared<ServiceControlFilter>(
              filter_config->stats(), filter_config->handler_factory());
          callbacks.addStreamFilter(Http::StreamFilterSharedPtr(filter));
          callbacks.addAccessLogHandler(AccessLog::InstanceSharedPtr(filter));
        };
  }
};

/**
 * Static registration for the rate limit filter. @see RegisterFactory.
 */
static Registry::RegisterFactory<
    FilterFactory, Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
