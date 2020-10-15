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

#include "src/envoy/http/path_rewrite/config_parser_impl.h"

#include "absl/strings/str_cat.h"
#include "common/common/empty_string.h"
#include "src/api_proxy/path_matcher/variable_binding_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {
namespace {

// Use fixed HTTP method for path_matcher
constexpr const char kHttpMethod[] = "GET";

}  // namespace

ConfigParserImpl::ConfigParserImpl(
    const ::espv2::api::envoy::v9::http::path_rewrite::PerRouteFilterConfig&
        config)
    : config_(config) {
  if (config_.has_constant_path()) {
    const auto& path_cfg = config_.constant_path();
    if (!path_cfg.url_template().empty()) {
      ENVOY_LOG(debug, "Building path_matcher for url_template: {}",
                path_cfg.url_template());

      ::espv2::api_proxy::path_matcher::PathMatcherBuilder<
          const ::espv2::api::envoy::v9::http::path_rewrite::
              PerRouteFilterConfig*>
          pmb;
      pmb.Register(kHttpMethod, path_cfg.url_template(), Envoy::EMPTY_STRING,
                   &config_);
      path_matcher_ = pmb.Build();
    }

    // If the last char of the path is "/", remove it, unless it is just root
    // "/".
    const std::string& path = path_cfg.path();
    if (path.size() > 1 && path[path.size() - 1] == '/') {
      ENVOY_LOG(warn, "Remove last slash of constant_path.path: {}", path);
      config_.mutable_constant_path()->set_path(
          path.substr(0, path.size() - 1));
    }
  } else {
    // even "/" should be removed
    const std::string& path = config_.path_prefix();
    if (path.size() > 0 && path[path.size() - 1] == '/') {
      ENVOY_LOG(warn, "Remove last slash of path_prefix: {}", path);
      config_.set_path_prefix(path.substr(0, path.size() - 1));
    }
  }
}

bool ConfigParserImpl::rewrite(absl::string_view origin_path,
                               std::string& new_path) const {
  if (config_.has_constant_path()) {
    return constPath(std::string(origin_path), new_path);
  }

  new_path = absl::StrCat(config_.path_prefix(), origin_path);
  ENVOY_LOG(debug, "Use path prefix: new path: {}", new_path);
  return true;
}

bool ConfigParserImpl::getVariableBindings(const std::string& origin_path,
                                           std::string& query) const {
  query = Envoy::EMPTY_STRING;
  if (path_matcher_) {
    std::vector<espv2::api_proxy::path_matcher::VariableBinding>
        variable_bindings;
    if (path_matcher_->Lookup(kHttpMethod, origin_path, &variable_bindings) ==
        nullptr) {
      // mismatched case
      ENVOY_LOG(warn, "Request path: {} doesn't match url_template: {}",
                origin_path, config_.constant_path().url_template());
      return false;
    }

    if (!variable_bindings.empty()) {
      query = espv2::api_proxy::path_matcher::VariableBindingsToQueryParameters(
          variable_bindings);
      ENVOY_LOG(debug, "Extracted query parameters: {}", query);
    }
  }
  return true;
}

bool ConfigParserImpl::constPath(const std::string& origin_path,
                                 std::string& new_path) const {
  std::string extracted_query_params;
  if (!getVariableBindings(origin_path, extracted_query_params)) {
    return false;
  }

  const auto& path_cfg = config_.constant_path();
  new_path = path_cfg.path();

  std::size_t originalQueryParamPos = origin_path.find('?');
  if (originalQueryParamPos == std::string::npos) {
    // No query param in original request.
    if (!extracted_query_params.empty()) {
      // Add extracted variable bindings.
      absl::StrAppend(&new_path, "?", extracted_query_params);
    }
    ENVOY_LOG(debug, "Use constant path, new path: {}", new_path);
    return true;
  }

  // Has query parameters in original request.
  const std::string originalQueryParam =
      origin_path.substr(originalQueryParamPos);
  absl::StrAppend(&new_path, originalQueryParam);
  if (!extracted_query_params.empty()) {
    // Append extracted variable bindings.
    absl::StrAppend(&new_path, "&", extracted_query_params);
  }
  ENVOY_LOG(debug, "Use constant path, new path: {}", new_path);
  return true;
}

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
