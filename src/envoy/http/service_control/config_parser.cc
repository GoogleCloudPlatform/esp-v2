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

using ::google::api::envoy::http::service_control::FilterConfig;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

FilterConfigParser::FilterConfigParser(const FilterConfig& config,
                                       ServiceControlCallFactory& factory)
    : config_(config) {
  for (const auto& service : config_.services()) {
    service_map_.emplace(
        service.service_name(),
        ServiceContextPtr(new ServiceContext(service, config_, factory)));
  }

  if (service_map_.size() < static_cast<size_t>(config_.services_size())) {
    throw ProtoValidationException("Duplicated service names", config_);
  }

  for (const auto& requirement : config_.requirements()) {
    const auto service_it = service_map_.find(requirement.service_name());
    if (service_it == service_map_.end()) {
      throw ProtoValidationException("Invalid service name", requirement);
    }
    requirements_map_.emplace(requirement.operation_name(),
                              RequirementContextPtr(new RequirementContext(
                                  requirement, *service_it->second)));
  }

  if (requirements_map_.size() <
      static_cast<size_t>(config_.requirements_size())) {
    throw ProtoValidationException("Duplicated operation names", config_);
  }

  // The default places to extract api-key
  default_api_keys_.add_locations()->set_query("key");
  default_api_keys_.add_locations()->set_query("api_key");
  default_api_keys_.add_locations()->set_header("x-api-key");
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
