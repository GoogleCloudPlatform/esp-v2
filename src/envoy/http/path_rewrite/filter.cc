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

#include "src/envoy/http/path_rewrite/filter.h"

#include <string>

#include "common/http/headers.h"
#include "common/http/utility.h"
#include "envoy/http/header_map.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

using Envoy::Http::FilterDataStatus;
using Envoy::Http::FilterHeadersStatus;
using Envoy::Http::FilterTrailersStatus;
using Envoy::Http::RequestHeaderMap;

FilterHeadersStatus Filter::decodeHeaders(RequestHeaderMap& headers, bool) {
  if (headers.Path() == nullptr) {
    // NOTE: this shouldn't happen in practice because ServiceControl filter
    // would have already rejected the request.
    config_->stats().denied_by_no_path_.inc();
    rejectRequest(Envoy::Http::Code::BadRequest, "No path in request headers",
                  utils::generateRcDetails(utils::kRcDetailFilterPathRewrite,
                                           utils::kRcDetailErrorTypeBadRequest,
                                           utils::kRcDetailErrorMissingPath));
    return FilterHeadersStatus::StopIteration;
  }

  absl::string_view original_path = headers.Path()->value().getStringView();
  // Reject requests with fragment identifiers. They should never be sent to
  // servers, and it breaks how we handle path translation (query params
  // appended incorrectly).
  if (absl::StrContains(original_path, "#")) {
    config_->stats().denied_by_invalid_path_.inc();
    rejectRequest(
        Envoy::Http::Code::BadRequest,
        "Path cannot contain fragment identifier (#)",
        utils::generateRcDetails(utils::kRcDetailFilterPathRewrite,
                                 utils::kRcDetailErrorTypeBadRequest,
                                 utils::kRcDetailErrorFragmentIdentifier));
    return FilterHeadersStatus::StopIteration;
  }

  // Make sure route is calculated
  auto route = decoder_callbacks_->route();
  if (route == nullptr || route->routeEntry() == nullptr) {
    config_->stats().denied_by_no_route_.inc();

    rejectRequest(
        Envoy::Http::Code::NotFound,
        absl::StrCat("Request `", utils::readHeaderEntry(headers.Method()), " ",
                     utils::readHeaderEntry(headers.Path()),
                     "` is not defined by this API."),
        utils::generateRcDetails(utils::kRcDetailFilterPathRewrite,
                                 utils::kRcDetailErrorTypeUndefinedRequest));
    return FilterHeadersStatus::StopIteration;
  }

  const auto* per_route =
      route->routeEntry()->perFilterConfigTyped<PerRouteFilterConfig>(
          kFilterName);
  if (per_route == nullptr) {
    ENVOY_LOG(debug, "no per-route path_rewrite config");
    config_->stats().path_not_changed_.inc();
    return FilterHeadersStatus::Continue;
  }

  std::string new_path;
  if (!per_route->config_parser().rewrite(original_path, new_path)) {
    config_->stats().denied_by_url_template_mismatch_.inc();
    rejectRequest(
        Envoy::Http::Code::InternalServerError,
        absl::StrCat("Request `", utils::readHeaderEntry(headers.Method()), " ",
                     utils::readHeaderEntry(headers.Path()),
                     "` is mismatched with url_template: ",
                     per_route->config_parser().url_template()),
        utils::generateRcDetails(utils::kRcDetailFilterPathRewrite,
                                 utils::kRcDetailErrorTypeUndefinedRequest));
    return FilterHeadersStatus::StopIteration;
  }

  config_->stats().path_changed_.inc();
  if (!headers.EnvoyOriginalPath()) {
    headers.setEnvoyOriginalPath(headers.getPathValue());
  }
  headers.setPath(new_path);
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

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
