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

#include "google/protobuf/stubs/logging.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

using ::google::api_proxy::envoy::http::service_control::FilterConfig;
using ::google::api::envoy::http::service_control::Requirement;
using std::string;

ServiceControlFilterConfigParser::ServiceControlFilterConfigParser(
    const FilterConfig& config) : config_(config) {
  BuildPathMatcher();
}

void ServiceControlFilterConfigParser::BuildPathMatcher() {
  PathMatcherBuilder<const Requirement*> pmb;
  for (const auto& rule : config_.rules()) {
    if (!pmb.Register(rule.pattern().http_method(), rule.pattern().uri_template(),
                      string(), &rule.requires())) {
      GOOGLE_LOG(WARNING)
          << "Invalid uri_template: " << rule.pattern().uri_template();
    }
  }
  path_matcher_ = pmb.Build();
}

void ServiceControlFilterConfigParser::FindRequirement(
    const string& http_method, const string& path, Requirement* requirement) {
  const Requirement* matched_requirement =
      path_matcher_->Lookup(http_method, path);
  if (matched_requirement) {
    requirement->MergeFrom(*matched_requirement);
  }
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy