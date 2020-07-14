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

#include "src/envoy/http/path_matcher/filter_config.h"
#include "common/common/empty_string.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {

using ::espv2::api::envoy::v7::http::path_matcher::PathMatcherRule;

FilterConfig::FilterConfig(
    const ::espv2::api::envoy::v7::http::path_matcher::FilterConfig&
        proto_config,
    const std::string& stats_prefix,
    Envoy::Server::Configuration::FactoryContext& context)
    : proto_config_(proto_config),
      stats_(generateStats(stats_prefix, context.scope())) {
  ::espv2::api_proxy::path_matcher::PathMatcherBuilder<const PathMatcherRule*>
      pmb;
  for (const auto& rule : proto_config_.rules()) {
    if (!pmb.Register(rule.pattern().http_method(),
                      rule.pattern().uri_template(),
                      /*body_field_path=*/Envoy::EMPTY_STRING, &rule)) {
      throw Envoy::ProtoValidationException(
          "Duplicated pattern or invalid pattern", rule.pattern());
    }
  }
  path_matcher_ = pmb.Build();
}

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
