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

// TODO(kyuc): Add unit tests

#include "src/envoy/http/backend_auth/filter.h"

#include <string>

#include "common/http/headers.h"
#include "common/http/utility.h"
#include "src/envoy/utils/filter_state_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

using Envoy::Http::FilterDataStatus;
using Envoy::Http::FilterHeadersStatus;
using Envoy::Http::FilterTrailersStatus;
using Envoy::Http::RequestHeaderMap;

namespace {
constexpr char kBearer[] = "Bearer ";

struct RcDetailsValues {
  // The request is rejected due to missing backend auth token.
  const std::string MissingBackendToken = "missing_backend_token";
};
typedef Envoy::ConstSingleton<RcDetailsValues> RcDetails;

}  // namespace

FilterHeadersStatus Filter::decodeHeaders(RequestHeaderMap& headers, bool) {
  absl::string_view operation = utils::getStringFilterState(
      *decoder_callbacks_->streamInfo().filterState(), utils::kOperation);
  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    ENVOY_LOG(debug, "No operation found from DynamicMetadata");
    return FilterHeadersStatus::Continue;
  }

  ENVOY_LOG(debug, "Found operation: {}", operation);
  absl::string_view audience = config_->cfg_parser().getAudience(operation);
  if (audience.empty()) {
    // This filter does not need to set a JWT Token for this operation.
    // If the request already has an Authorization header, it will be preserved.
    return FilterHeadersStatus::Continue;
  }

  const TokenSharedPtr jwt_token = config_->cfg_parser().getJwtToken(audience);
  if (!jwt_token) {
    ENVOY_LOG(debug, "Token not found for audience: {}", audience);
    decoder_callbacks_->sendLocalReply(Envoy::Http::Code::InternalServerError,
                                       "missing tokens", nullptr, absl::nullopt,
                                       RcDetails::get().MissingBackendToken);
    decoder_callbacks_->streamInfo().setResponseFlag(
        Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService);
    return FilterHeadersStatus::StopIteration;
  }

  const auto& authorization = Envoy::Http::Headers::get().Authorization;
  headers.remove(authorization);
  headers.addCopy(authorization, kBearer + *jwt_token);
  config_->stats().token_added_.inc();
  return FilterHeadersStatus::Continue;
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
