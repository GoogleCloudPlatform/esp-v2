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

#include <string>

#include "src/envoy/http/path_matcher/filter.h"

#include "common/http/utility.h"
#include "src/api_proxy/path_matcher/variable_binding_utils.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {

using ::Envoy::Http::RequestHeaderMap;
using ::espv2::api::envoy::v7::http::path_matcher::PathMatcherRule;
using ::espv2::api::envoy::v7::http::path_matcher::PathParameterExtractionRule;
using ::espv2::api_proxy::path_matcher::VariableBinding;
using ::espv2::api_proxy::path_matcher::VariableBindingsToQueryParameters;
using ::google::protobuf::util::Status;

namespace {

// Half of the max header value size Envoy allows.
// 4x the standard browser request size.
constexpr uint32_t PathMaxSize = 8192;

struct RcDetailsValues {
  // The path is not defined in the service config.
  const std::string PathNotDefined = "path_not_defined";
};
using RcDetails = Envoy::ConstSingleton<RcDetailsValues>;

}  // namespace

Envoy::Http::FilterHeadersStatus Filter::decodeHeaders(
    RequestHeaderMap& headers, bool) {
  if (!headers.Method()) {
    rejectRequest(Envoy::Http::Code(400), "No method in request headers.");
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (!headers.Path()) {
    rejectRequest(Envoy::Http::Code(400), "No path in request headers.");
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (headers.Path()->value().size() > PathMaxSize) {
    rejectRequest(Envoy::Http::Code(400),
                  absl::StrCat("Path is too long, max allowed size is ",
                               PathMaxSize, "."));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  if (utils::handleHttpMethodOverride(headers)) {
    // Update later filters that the HTTP method has changed by clearing the
    // route cache.
    ENVOY_LOG(debug, "HTTP method override occurred, recalculating route");
    decoder_callbacks_->clearRouteCache();
  }

  std::string method(headers.Method()->value().getStringView());
  std::string path(headers.Path()->value().getStringView());

  const PathMatcherRule* rule = config_->findRule(method, path);
  if (rule == nullptr) {
    rejectRequest(Envoy::Http::Code(404),
                  "Path does not match any requirement URI template.");
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  Envoy::StreamInfo::FilterState& filter_state =
      *decoder_callbacks_->streamInfo().filterState();
  const absl::string_view operation = rule->operation();
  ENVOY_LOG(debug, "matched operation: {}", operation);
  utils::setStringFilterState(filter_state, utils::kFilterStateOperation,
                              operation);

  if (rule->has_path_parameter_extraction()) {
    const PathParameterExtractionRule& param_rule =
        rule->path_parameter_extraction();

    std::vector<VariableBinding> variable_bindings;
    config_->findRule(method, path, &variable_bindings);

    if (!variable_bindings.empty()) {
      const std::string query_params = VariableBindingsToQueryParameters(
          variable_bindings, param_rule.snake_to_json_segments());
      utils::setStringFilterState(filter_state, utils::kFilterStateQueryParams,
                                  query_params);
    }
  }

  config_->stats().allowed_.inc();
  return Envoy::Http::FilterHeadersStatus::Continue;
}

void Filter::rejectRequest(Envoy::Http::Code code,
                           absl::string_view error_msg) {
  config_->stats().denied_.inc();

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     RcDetails::get().PathNotDefined);
  decoder_callbacks_->streamInfo().setResponseFlag(
      Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
