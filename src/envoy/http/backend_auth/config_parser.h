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
#pragma once
#include <list>
#include <unordered_map>

#include "absl/container/flat_hash_map.h"
#include "absl/strings/str_cat.h"
#include "api/envoy/v10/http/backend_auth/config.pb.h"
#include "envoy/thread_local/thread_local.h"
#include "src/envoy/token/token_subscriber_factory.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

// The filter name.
constexpr const char kFilterName[] =
    "com.google.espv2.filters.http.backend_auth";

// Use shared_ptr to do atomic token update.
using TokenSharedPtr = std::shared_ptr<std::string>;

class FilterConfigParser {
 public:
  virtual ~FilterConfigParser() = default;

  virtual const TokenSharedPtr getJwtToken(
      absl::string_view audience) const PURE;
};

using FilterConfigParserPtr = std::unique_ptr<FilterConfigParser>;

class PerRouteFilterConfig : public Envoy::Router::RouteSpecificFilterConfig {
 public:
  PerRouteFilterConfig(
      const ::espv2::api::envoy::v10::http::backend_auth::PerRouteFilterConfig&
          per_route)
      : jwt_audience_(per_route.jwt_audience()) {}

  absl::string_view jwt_audience() const { return jwt_audience_; }

 private:
  std::string jwt_audience_;
};

using PerRouteFilterConfigSharedPtr = std::shared_ptr<PerRouteFilterConfig>;

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
