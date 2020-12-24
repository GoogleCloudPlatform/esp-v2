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
#include "envoy/http/header_map.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

using Envoy::Http::CustomHeaders;
using Envoy::Http::CustomInlineHeaderRegistry;
using Envoy::Http::FilterDataStatus;
using Envoy::Http::FilterHeadersStatus;
using Envoy::Http::FilterTrailersStatus;
using Envoy::Http::RegisterCustomInlineHeader;
using Envoy::Http::RequestHeaderMap;

namespace {
constexpr char kBearer[] = "Bearer ";

RegisterCustomInlineHeader<CustomInlineHeaderRegistry::Type::RequestHeaders>
    authorization_handle(CustomHeaders::get().Authorization);

// The Http header to copy the original Authorization before it is overwritten.
const Envoy::Http::LowerCaseString kXForwardedAuthorization{
    "x-forwarded-authorization"};

}  // namespace

FilterHeadersStatus Filter::decodeHeaders(RequestHeaderMap& headers, bool) {
  // Make sure route is calculated
  auto route = decoder_callbacks_->route();

  // `route` shouldn't be nullptr as the catch-all route match should catch all
  // the undefined requests.
  if (route == nullptr || route->routeEntry() == nullptr) {
    return Envoy::Http::FilterHeadersStatus::Continue;
  }

  const auto* per_route =
      route->routeEntry()->perFilterConfigTyped<PerRouteFilterConfig>(
          kFilterName);
  if (per_route == nullptr) {
    ENVOY_LOG(debug, "no per-route config");
    config_->stats().allowed_by_auth_not_required_.inc();
    return FilterHeadersStatus::Continue;
  }

  const auto& audience = per_route->jwt_audience();
  ENVOY_LOG(debug, "Found jwt_audience: {}", audience);
  const TokenSharedPtr jwt_token = config_->cfg_parser().getJwtToken(audience);
  if (!jwt_token) {
    config_->stats().denied_by_no_token_.inc();
    rejectRequest(
        Envoy::Http::Code::InternalServerError,
        absl::StrCat("Token not found for audience: ", audience),
        utils::generateRcDetails(utils::kRcDetailFilterBackendAuth,
                                 utils::kRcDetailErrorTypeMissingBackendToken));
    return FilterHeadersStatus::StopIteration;
  }

  // Copy the existing `Authorization` header to `x-forwarded-authorization`
  // header.
  const Envoy::Http::HeaderEntry* existAuthToken =
      headers.getInline(authorization_handle.handle());
  if (existAuthToken != nullptr) {
    headers.addCopy(kXForwardedAuthorization,
                    existAuthToken->value().getStringView());
  }

  headers.setInline(authorization_handle.handle(), kBearer + *jwt_token);
  config_->stats().token_added_.inc();
  return FilterHeadersStatus::Continue;
}

void Filter::rejectRequest(Envoy::Http::Code code, absl::string_view error_msg,
                           absl::string_view details) {
  ENVOY_LOG(debug, "{}", error_msg);
  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt,
                                     details);
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
