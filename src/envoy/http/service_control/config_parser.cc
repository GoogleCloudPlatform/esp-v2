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

using ::google::api::envoy::http::service_control::Requirement;
using ::google::api_proxy::envoy::http::service_control::FilterConfig;

FilterConfigParser::FilterConfigParser(const FilterConfig& config) {
  PathMatcherBuilder<const Requirement*> pmb;
  for (const auto& rule : config.rules()) {
    if (!rule.has_pattern()) {
      ENVOY_LOG(error, "Empty rule pattern");
      continue;
    }
    const auto& pattern = rule.pattern();
    if (!pmb.Register(pattern.http_method(), pattern.uri_template(),
                      std::string(), &rule.requires())) {
      ENVOY_LOG(error, "Invalid rule pattern: http_method: {}, uri_template {}",
                pattern.http_method(), pattern.uri_template());
    }
  }
  path_matcher_ = pmb.Build();
}

const Requirement* FilterConfigParser::FindRequirement(
    const std::string& http_method, const std::string& path) const {
  return path_matcher_->Lookup(http_method, path);
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
