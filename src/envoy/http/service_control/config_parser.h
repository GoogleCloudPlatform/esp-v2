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
#include "src/envoy/http/service_control/path_matcher.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class FilterConfigParser : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfigParser(
      const ::google::api_proxy::envoy::http::service_control::FilterConfig&
          config);

  const ::google::api::envoy::http::service_control::Requirement*
  FindRequirement(const std::string& http_method, const std::string& path) const;

 private:
  // Build PatchMatcher for extracting api attributes.
  void BuildPathMatcher();

  // The path matcher for all url templates
  PathMatcherPtr<
      const ::google::api::envoy::http::service_control::Requirement*>
      path_matcher_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

#endif  // ENVOY_SERVICE_CONTROL_RULE_PARSER_H
