// Copyright 2023 Google LLC
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

#include "src/envoy/http/header_sanitizer/filter.h"

#include <string>

#include "envoy/http/header_map.h"
#include "source/common/http/headers.h"
#include "source/common/http/utility.h"
#include "src/envoy/utils/http_header_utils.h"
#include "src/envoy/utils/rc_detail_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace header_sanitizer {

using Envoy::Http::FilterDataStatus;
using Envoy::Http::FilterHeadersStatus;
using Envoy::Http::FilterTrailersStatus;
using Envoy::Http::RequestHeaderMap;

namespace {

const Envoy::Http::LowerCaseString kXEndpointApiUserInfo{
    "x-endpoint-api-userinfo"};
const Envoy::Http::LowerCaseString kXForwardedAuthorization{
    "x-forwarded-authorization"};
const Envoy::Http::LowerCaseString kXEndpointApiConsumerType{
    "x-endpoint-api-consumer-type"};
const Envoy::Http::LowerCaseString kXEndpointApiConsumerNumber{
    "x-endpoint-api-consumer-number"};

}  // namespace

FilterHeadersStatus Filter::decodeHeaders(RequestHeaderMap& headers, bool) {
  headers.remove(kXEndpointApiUserInfo);
  headers.remove(kXForwardedAuthorization);
  headers.remove(kXEndpointApiConsumerType);
  headers.remove(kXEndpointApiConsumerNumber);

  if (utils::handleHttpMethodOverride(headers)) {
    // Update later filters that the HTTP method has changed by clearing the
    // route cache.
    ENVOY_LOG(debug, "HTTP method override occurred, recalculating route");
    decoder_callbacks_->downstreamCallbacks()->clearRouteCache();
  }

  return FilterHeadersStatus::Continue;
}

}  // namespace header_sanitizer
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
