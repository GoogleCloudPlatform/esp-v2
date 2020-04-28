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

using ::espv2::api_proxy::path_matcher::VariableBinding;
using ::espv2::api_proxy::path_matcher::VariableBindingsToQueryParameters;
using ::google::protobuf::util::Status;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {
namespace {

struct RcDetailsValues {
  // The path is not defined in the service config.
  const std::string PathNotDefined = "path_not_defined";
};
using RcDetails = Envoy::ConstSingleton<RcDetailsValues>;

}  // namespace

Envoy::Http::FilterHeadersStatus Filter::decodeHeaders(
    Envoy::Http::RequestHeaderMap& headers, bool) {
  if (!headers.Method()) {
    rejectRequest(Envoy::Http::Code(400), "No method in request headers.");
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (!headers.Path()) {
    rejectRequest(Envoy::Http::Code(400), "No path in request headers.");
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
  const std::string* operation = config_->findOperation(method, path);
  if (operation == nullptr) {
    rejectRequest(Envoy::Http::Code(404),
                  "Path does not match any requirement URI template.");
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  ENVOY_LOG(debug, "matched operation: {}", *operation);
  Envoy::StreamInfo::FilterState& filter_state =
      *decoder_callbacks_->streamInfo().filterState();
  utils::setStringFilterState(filter_state, utils::kOperation, *operation);

  if (config_->needParameterExtraction(*operation)) {
    std::vector<VariableBinding> variable_bindings;
    operation = config_->findOperation(method, path, &variable_bindings);
    if (!variable_bindings.empty()) {
      const std::string query_params = VariableBindingsToQueryParameters(
          variable_bindings, config_->getSnakeToJsonMap());
      utils::setStringFilterState(filter_state, utils::kQueryParams,
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
