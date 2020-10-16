// Copyright 2020 Google LLC
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

#include "api/envoy/v9/http/path_rewrite/config.pb.h"
#include "api/envoy/v9/http/path_rewrite/config.pb.validate.h"
#include "common/common/logger.h"
#include "src/api_proxy/path_matcher/path_matcher.h"
#include "src/envoy/http/path_rewrite/config_parser.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

class ConfigParserImpl
    : public ConfigParser,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  ConfigParserImpl(
      const ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig&
          config);

  bool rewrite(absl::string_view origin_path,
               std::string& new_path) const override;

  absl::string_view url_template() const override;

 private:
  // rewrite const path.
  bool constPath(const std::string& origin_path, std::string& new_path) const;
  // extract query parameters from variable bindings
  bool getVariableBindings(const std::string& origin_path,
                           std::string& query) const;

  // the per-route config
  ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig config_;
  // path matcher for extracting variable binding.
  ::espv2::api_proxy::path_matcher::PathMatcherPtr<
      const ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig*>
      path_matcher_;
};

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
