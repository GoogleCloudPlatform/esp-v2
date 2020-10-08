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
  absl::string_view operation = utils::getStringFilterState(
      *decoder_callbacks_->streamInfo().filterState(),
      utils::kFilterStateOperation);
  // NOTE: this shouldn't happen in practice because Path Matcher filter would
  // have already rejected the request.
  if (operation.empty()) {
    config_->stats().denied_by_no_operation_.inc();

    rejectRequest(
        Envoy::Http::Code::InternalServerError,
        absl::StrCat("Request `", utils::readHeaderEntry(headers.Method()), " ",
                     utils::readHeaderEntry(headers.Path()),
                     "` is not defined by this API."),
        utils::generateRcDetails(utils::kRcDetailFilterBackendAuth,
                                 utils::kRcDetailErrorTypeUndefinedRequest));
    return FilterHeadersStatus::StopIteration;
  }

  ENVOY_LOG(debug, "Found operation: {}", operation);
  absl::string_view audience = config_->cfg_parser().getAudience(operation);
  if (audience.empty()) {
    // By design, we only want to apply the filter to operations that are in the
    // configuration. Otherwise, let it pass through (no need to add a JWT for
    // this request). If the request already has an Authorization header, it
    // will be preserved.
    config_->stats().allowed_by_no_configured_rules_.inc();
    return FilterHeadersStatus::Continue;
  }

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
  decoder_callbacks_->streamInfo().setResponseFlag(
      Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
