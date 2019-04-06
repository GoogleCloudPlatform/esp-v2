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

#pragma once

#include "absl/container/flat_hash_map.h"
#include "absl/strings/string_view.h"

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/requirement.pb.h"
#include "src/envoy/http/service_control/service_control_call.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ServiceContext {
 public:
  ServiceContext(
      const ::google::api::envoy::http::service_control::Service& config,
      ServiceControlCallFactory& factory)
      : config_(config), service_control_call_(factory.create(config)) {}

  const ::google::api::envoy::http::service_control::Service& config() const {
    return config_;
  }

  ServiceControlCall& call() const { return *service_control_call_; }

 private:
  const ::google::api::envoy::http::service_control::Service& config_;
  ServiceControlCallPtr service_control_call_;
};
typedef std::unique_ptr<ServiceContext> ServiceContextPtr;

class RequirementContext {
 public:
  RequirementContext(
      const ::google::api::envoy::http::service_control::Requirement& config,
      const ServiceContext& service_ctx)
      : config_(config), service_ctx_(service_ctx) {}

  const ::google::api::envoy::http::service_control::Requirement& config()
      const {
    return config_;
  }

  const ServiceContext& service_ctx() const { return service_ctx_; }

 private:
  const ::google::api::envoy::http::service_control::Requirement& config_;
  const ServiceContext& service_ctx_;
};
typedef std::unique_ptr<RequirementContext> RequirementContextPtr;

class FilterConfigParser {
 public:
  FilterConfigParser(
      const ::google::api::envoy::http::service_control::FilterConfig& config,
      ServiceControlCallFactory& factory);

  const RequirementContext* FindRequirement(absl::string_view operation) const {
    const auto requirement_it = requirements_map_.find(operation);
    if (requirement_it == requirements_map_.end()) {
      return nullptr;
    }
    return requirement_it->second.get();
  }

 private:
  // Operation name to RequirementContext map.
  absl::flat_hash_map<std::string, RequirementContextPtr> requirements_map_;
  // Service name to ServiceContext map.
  absl::flat_hash_map<std::string, ServiceContextPtr> service_map_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
