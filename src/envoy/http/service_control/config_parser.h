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

#pragma once

#include "absl/container/flat_hash_map.h"
#include "absl/strings/string_view.h"

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/requirement.pb.h"
#include "src/envoy/http/service_control/service_control_call.h"

// Default minimum interval (milliseconds) for streaming reports.
#define kDefaultMinStreamReportIntervalMs 10000

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ServiceContext {
 public:
  ServiceContext(
      const ::google::api::envoy::http::service_control::Service& config,
      const ::google::api::envoy::http::service_control::FilterConfig&
          filter_config,
      ServiceControlCallFactory& factory)
      : config_(config),
        service_control_call_(factory.create(config_, filter_config)) {
    min_stream_report_interval_ms_ = config_.min_stream_report_interval_ms();
    if (!min_stream_report_interval_ms_) {
      min_stream_report_interval_ms_ = kDefaultMinStreamReportIntervalMs;
    }
  }

  const ::google::api::envoy::http::service_control::Service& config() const {
    return config_;
  }

  int64_t get_min_stream_report_interval_ms() const {
    return min_stream_report_interval_ms_;
  }

  ServiceControlCall& call() const { return *service_control_call_; }

 private:
  const ::google::api::envoy::http::service_control::Service& config_;
  ServiceControlCallPtr service_control_call_;
  int64_t min_stream_report_interval_ms_;
};
typedef std::unique_ptr<ServiceContext> ServiceContextPtr;

class RequirementContext {
 public:
  RequirementContext(
      const ::google::api::envoy::http::service_control::Requirement& config,
      const ServiceContext& service_ctx)
      : config_(config), service_ctx_(service_ctx) {
    for (const auto& metric_cost : config.metric_costs()) {
      metric_costs_.push_back(
          std::make_pair(metric_cost.name(), metric_cost.cost()));
    }
  }

  const ::google::api::envoy::http::service_control::Requirement& config()
      const {
    return config_;
  }

  const ServiceContext& service_ctx() const { return service_ctx_; }

  const std::vector<std::pair<std::string, int>>* metric_costs() const {
    return &metric_costs_;
  }

 private:
  const ::google::api::envoy::http::service_control::Requirement& config_;
  const ServiceContext& service_ctx_;
  std::vector<std::pair<std::string, int>> metric_costs_;
};
typedef std::unique_ptr<RequirementContext> RequirementContextPtr;

class FilterConfigParser {
 public:
  FilterConfigParser(
      const ::google::api::envoy::http::service_control::FilterConfig& config,
      ServiceControlCallFactory& factory);

  const ::google::api::envoy::http::service_control::FilterConfig& config()
      const {
    return config_;
  }
  const RequirementContext* FindRequirement(absl::string_view operation) const {
    const auto requirement_it = requirements_map_.find(operation);
    if (requirement_it == requirements_map_.end()) {
      return nullptr;
    }
    return requirement_it->second.get();
  }

  const ::google::api::envoy::http::service_control::APIKeyRequirement&
  default_api_keys() const {
    return default_api_keys_;
  }

  const RequirementContext* non_match_rqm_ctx() const {
    return non_match_rqm_ctx_.get();
  }

 private:
  // The proto config.
  const ::google::api::envoy::http::service_control::FilterConfig config_;
  // Operation name to RequirementContext map.
  absl::flat_hash_map<std::string, RequirementContextPtr> requirements_map_;
  // The requirement for non matched requests for sending their reports.
  ::google::api::envoy::http::service_control::Requirement non_match_rqm_cfg_;
  RequirementContextPtr non_match_rqm_ctx_;
  // Service name to ServiceContext map.
  absl::flat_hash_map<std::string, ServiceContextPtr> service_map_;
  // The default locations to extract api-key.
  ::google::api::envoy::http::service_control::APIKeyRequirement
      default_api_keys_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
