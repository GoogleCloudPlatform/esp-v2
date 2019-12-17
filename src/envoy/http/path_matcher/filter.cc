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

#include "common/common/logger.h"
#include "common/http/utility.h"
#include "common/protobuf/utility.h"
#include "envoy/server/filter_config.h"
#include "src/api_proxy/path_matcher/variable_binding_utils.h"
#include "src/envoy/http/path_matcher/filter_config.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

using ::google::api_proxy::path_matcher::VariableBinding;
using ::google::api_proxy::path_matcher::VariableBindingsToQueryParameters;
using ::google::protobuf::util::Status;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace PathMatcher {
namespace {

struct RcDetailsValues {
  // The path is not defined in the service config.
  const std::string PathNotDefined = "path_not_defined";
};
typedef ConstSingleton<RcDetailsValues> RcDetails;

}  // namespace

Http::FilterHeadersStatus Filter::decodeHeaders(Http::HeaderMap& headers,
                                                bool) {
  std::string method(Utils::getRequestHTTPMethodWithOverride(
      headers.Method()->value().getStringView(), headers));
  std::string path(headers.Path()->value().getStringView());
  const std::string* operation = config_->FindOperation(method, path);
  if (operation == nullptr) {
    rejectRequest(Http::Code(404),
                  "Path does not match any requirement URI template.");
    return Http::FilterHeadersStatus::StopIteration;
  }

  ENVOY_LOG(debug, "matched operation: {}", *operation);
  StreamInfo::FilterState& filter_state =
      decoder_callbacks_->streamInfo().filterState();
  Utils::setStringFilterState(filter_state, Utils::kOperation, *operation);

  if (config_->NeedPathParametersExtraction(*operation)) {
    std::vector<VariableBinding> variable_bindings;
    operation = config_->FindOperation(method, path, &variable_bindings);
    if (!variable_bindings.empty()) {
      const std::string query_params = VariableBindingsToQueryParameters(
          variable_bindings, config_->snake_to_json());
      Utils::setStringFilterState(filter_state, Utils::kQueryParams,
                                  query_params);
    }
  }

  config_->stats().allowed_.inc();
  return Http::FilterHeadersStatus::Continue;
}

void Filter::rejectRequest(Http::Code code, absl::string_view error_msg) {
  config_->stats().denied_.inc();

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     RcDetails::get().PathNotDefined);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

}  // namespace PathMatcher
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
