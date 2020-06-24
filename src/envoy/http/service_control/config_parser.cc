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

#include "src/envoy/http/service_control/config_parser.h"

#include "common/protobuf/utility.h"

using ::espv2::api::envoy::v6::http::service_control::FilterConfig;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace {

// The operation name for not matched requests.
const char kUnrecognizedOperation[] = "<Unknown Operation Name>";
}  // namespace

FilterConfigParser::FilterConfigParser(const FilterConfig& config,
                                       ServiceControlCallFactory& factory)
    : config_(config) {
  ServiceContext* first_srv_ctx = nullptr;
  for (const auto& service : config_.services()) {
    ServiceContext* srv_ctx = new ServiceContext(service, factory);
    if (first_srv_ctx == nullptr) {
      first_srv_ctx = srv_ctx;
    }
    service_map_.emplace(service.service_name(), ServiceContextPtr(srv_ctx));
  }
  if (first_srv_ctx == nullptr) {
    throw Envoy::ProtoValidationException("Empty services", config_);
  }

  if (service_map_.size() < static_cast<size_t>(config_.services_size())) {
    throw Envoy::ProtoValidationException("Duplicated service names", config_);
  }

  for (const auto& requirement : config_.requirements()) {
    const auto service_it = service_map_.find(requirement.service_name());
    if (service_it == service_map_.end()) {
      throw Envoy::ProtoValidationException("Invalid service name",
                                            requirement);
    }
    requirements_map_.emplace(requirement.operation_name(),
                              RequirementContextPtr(new RequirementContext(
                                  requirement, *service_it->second)));
  }

  if (requirements_map_.size() <
      static_cast<size_t>(config_.requirements_size())) {
    throw Envoy::ProtoValidationException("Duplicated operation names",
                                          config_);
  }

  // Construct a requirement for non matched requests
  non_match_rqm_cfg_.set_service_name(first_srv_ctx->config().service_name());
  non_match_rqm_cfg_.set_operation_name(kUnrecognizedOperation);
  non_match_rqm_ctx_.reset(
      new RequirementContext(non_match_rqm_cfg_, *first_srv_ctx));

  // The default places to extract api-key
  default_api_keys_.add_locations()->set_query("key");
  default_api_keys_.add_locations()->set_query("api_key");
  default_api_keys_.add_locations()->set_header("x-api-key");
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
