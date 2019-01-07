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

#ifndef ENVOY_SERVICE_CONTROL_RULE_PARSER_H
#define ENVOY_SERVICE_CONTROL_RULE_PARSER_H

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/requirement.pb.h"
#include "common/common/logger.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/path_matcher.h"

#include <unordered_map>

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ServiceContext {
 public:
  ServiceContext(
      const ::google::api::envoy::http::service_control::Service& proto_config)
      : proto_config_(proto_config),
        request_builder_({"endpoints_log"}, proto_config_.service_name(),
                         proto_config_.service_config_id()) {}

  const ::google::api::envoy::http::service_control::Service& config() const {
    return proto_config_;
  }

  const ::google::api_proxy::service_control::RequestBuilder& builder() const {
    return request_builder_;
  }

 private:
  const ::google::api::envoy::http::service_control::Service& proto_config_;
  ::google::api_proxy::service_control::RequestBuilder request_builder_;
};
typedef std::unique_ptr<ServiceContext> ServiceContextPtr;

class FilterConfigParser : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfigParser(
      const ::google::api::envoy::http::service_control::FilterConfig& config);

  const ::google::api::envoy::http::service_control::Requirement*
  FindRequirement(const std::string& http_method,
                  const std::string& path) const {
    return path_matcher_->Lookup(http_method, path);
  }

  const ServiceContext* FindService(const std::string& name) const {
    const auto it = service_map_.find(name);
    return (it != service_map_.end()) ? it->second.get() : nullptr;
  }

 private:
  // Build PatchMatcher for extracting api attributes.
  void BuildPathMatcher();

  // The path matcher for all url templates
  PathMatcherPtr<
      const ::google::api::envoy::http::service_control::Requirement*>
      path_matcher_;

  // The service map
  std::unordered_map<std::string, ServiceContextPtr> service_map_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

#endif  // ENVOY_SERVICE_CONTROL_RULE_PARSER_H
