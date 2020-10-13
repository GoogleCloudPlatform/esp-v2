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

#include "common/http/utility.h"

#include "src/api_proxy/path_matcher/variable_binding_utils.h"
#include "src/envoy/http/path_matcher/filter.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {

using ::Envoy::Http::RequestHeaderMap;
using ::espv2::api::envoy::v9::http::path_matcher::PathMatcherRule;
using ::espv2::api_proxy::path_matcher::VariableBinding;
using ::espv2::api_proxy::path_matcher::VariableBindingsToQueryParameters;
using ::google::protobuf::util::Status;

namespace {

// Half of the max header value size Envoy allows.
// 4x the standard browser request size.
constexpr uint32_t PathMaxSize = 8192;

}  // namespace

Envoy::Http::FilterHeadersStatus Filter::decodeHeaders(
    RequestHeaderMap& headers, bool) {
  if (!headers.Method()) {
    rejectRequest(Envoy::Http::Code::BadRequest,
                  "No method in request headers.",
                  utils::generateRcDetails(utils::kRcDetailFilterPathMatcher,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorMissingMethod));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (!headers.Path()) {
    rejectRequest(Envoy::Http::Code::BadRequest, "No path in request headers.",
                  utils::generateRcDetails(utils::kRcDetailFilterPathMatcher,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorMissingPath));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  } else if (headers.Path()->value().size() > PathMaxSize) {
    rejectRequest(Envoy::Http::Code::BadRequest,
                  absl::StrCat("Path is too long, max allowed size is ",
                               PathMaxSize, "."),
                  utils::generateRcDetails(utils::kRcDetailFilterPathMatcher,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorOversizePath));
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
    rejectRequest(
        Envoy::Http::Code::NotFound,
        absl::StrCat("Request `", method, " ", path,
                     "` is not defined by this API."),
        utils::generateRcDetails(utils::kRcDetailFilterPathMatcher,
                                 utils::kRcDetailErrorTypeUndefinedRequest));
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  const absl::string_view operation = rule->operation();
  ENVOY_LOG(debug, "matched operation: {}", operation);
  Envoy::StreamInfo::FilterState& filter_state =
      *decoder_callbacks_->streamInfo().filterState();
  utils::setStringFilterState(filter_state, utils::kFilterStateOperation,
                              operation);

  if (rule->extract_path_parameters()) {
    std::vector<VariableBinding> variable_bindings;
    config_->findRule(method, path, &variable_bindings);

    if (!variable_bindings.empty()) {
      utils::setStringFilterState(
          filter_state, utils::kFilterStateQueryParams,
          VariableBindingsToQueryParameters(variable_bindings));
    }
  }

  config_->stats().allowed_.inc();
  return Envoy::Http::FilterHeadersStatus::Continue;
}

void Filter::rejectRequest(Envoy::Http::Code code, absl::string_view error_msg,
                           absl::string_view rc_detail) {
  config_->stats().denied_.inc();

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     rc_detail);
  decoder_callbacks_->streamInfo().setResponseFlag(
      Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
