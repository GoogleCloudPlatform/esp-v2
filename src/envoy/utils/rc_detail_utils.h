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
#include <string>
#include "absl/strings/str_cat.h"

namespace espv2 {
namespace envoy {
namespace utils {

// The filter prefixes.
const char kRcDetailFilterServiceControl[] = "service_control";
const char kRcDetailFilterPathMatcher[] = "path_matcher";
const char kRcDetailFilterBackendAuth[] = "backend_auth";
const char kRcDetailFilterBackendRouting[] = "backend_routing";

// The error types
//
// The common ones
const char kRcDetailErrorTypeBadRequest[] = "bad_request";
const char kRcDetailErrorTypeUndefinedRequest[] = "undefined_request";
// The ones specific to the service control filter.
const char kRcDetailErrorTypeScCheck[] = "check_error";
const char kRcDetailErrorTypeScQuota[] = "quota_error";
const char kRcDetailErrorTypeScCheckNetwork[] = "check_network_failure";
const char kRcDetailErrorTypeScQuotaNetwork[] = "quota_network_failure";
// The ones specific to the backend auth filter
const char kRcDetailErrorTypeMissingBackendToken[] = "missing_backend_token";

// The detailed errors.
const char kRcDetailErrorMissingApiKey[] = "MISSING_API_KEY";
const char kRcDetailErrorMissingMethod[] = "MISSING_METHOD";
const char kRcDetailErrorMissingPath[] = "MISSING_PATH";
const char kRcDetailErrorOversizePath[] = "OVERSIZE_PATH";
const char kRcDetailErrorFragmentIdentifier[] = "PATH_WITH_FRAGMENT_IDENTIFIER";

std::string generateRcDetails(absl::string_view filter_name,
                              absl::string_view error_type,
                              const std::string& error_detail = "");

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
