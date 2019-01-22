// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include "src/envoy/http/service_control/config_parser.h"

#include "common/protobuf/utility.h"
#include "google/protobuf/stubs/logging.h"

using ::google::api::envoy::http::service_control::FilterConfig;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

FilterConfigParser::FilterConfigParser(
    const FilterConfig& config,
    Server::Configuration::FactoryContext& context) {
  for (const auto& service : config.services()) {
    service_map_[service.service_name()] =
        ServiceContextPtr(new ServiceContext(service, context));
  }

  ::google::api_proxy::path_matcher::PathMatcherBuilder<
      const RequirementContext*>
      pmb;
  for (const auto& rule : config.rules()) {
    const auto& pattern = rule.pattern();
    const auto& requirement = rule.requires();

    const auto service_it = service_map_.find(requirement.service_name());
    if (service_it == service_map_.end()) {
      throw ProtoValidationException("Invalid service name", requirement);
    }

    RequirementContextPtr require_ctx(
        new RequirementContext(requirement, *service_it->second));
    if (!pmb.Register(pattern.http_method(), pattern.uri_template(),
                      std::string(), require_ctx.get())) {
      throw ProtoValidationException("Duplicated pattern", pattern);
    }
    require_ctx_list_.push_back(std::move(require_ctx));
  }
  path_matcher_ = pmb.Build();
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
