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

#include "absl/strings/match.h"
#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/requirement.pb.h"
#include "common/config/metadata.h"
#include "common/http/utility.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/filter_stats.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// Searches the headers at the given locations and sets the `api_key` if one is
// found.
//
// Returns whether an `api_key` was found.
bool extractAPIKey(
    const Envoy::Http::RequestHeaderMap& headers,
    const ::google::protobuf::RepeatedPtrField<
        ::espv2::api::envoy::http::service_control::ApiKeyLocation>& locations,
    std::string& api_key);

// Adds information from the `FilterConfig`'s gcp_attributes to the given info.
void fillGCPInfo(
    const ::espv2::api::envoy::http::service_control::FilterConfig&
        filter_config,
    ::espv2::api_proxy::service_control::ReportRequestInfo& info);

// Searches the `headers` for the given `log_headers` and appends all matches
// to the string provided.
void fillLoggedHeader(
    const Envoy::Http::HeaderMap* headers,
    const ::google::protobuf::RepeatedPtrField<::std::string>& log_headers,
    std::string& info_header_field);

// Fills the `request_time_ms`, `backend_time_ms`, and `overhead_time_ms` of the
// info provided.
void fillLatency(const Envoy::StreamInfo::StreamInfo& stream_info,
                 ::espv2::api_proxy::service_control::LatencyInfo& latency,
                 ServiceControlFilterStats& filter_stats);

// Fills the jwt payload of the info provided
void fillJwtPayloads(const ::envoy::config::core::v3::Metadata& metadata,
                     const std::string& jwt_payload_metadata_name,
                     const ::google::protobuf::RepeatedPtrField<::std::string>&
                         jwt_payload_paths,
                     std::string& info_jwt_payloads);

void fillJwtPayload(const ::envoy::config::core::v3::Metadata& metadata,
                    const std::string& jwt_payload_metadata_name,
                    const std::string& jwt_payload_path,
                    std::string& info_iss_or_aud);

// Returns the protocol of the frontend request or UNKNOWN if not found
::espv2::api_proxy::service_control::protocol::Protocol getFrontendProtocol(
    const Envoy::Http::HeaderMap* response_headers,
    const Envoy::StreamInfo::StreamInfo& stream_info);

// Returns the protocol of the backend service or UNKNOWN if not found
::espv2::api_proxy::service_control::protocol::Protocol getBackendProtocol(
    const ::espv2::api::envoy::http::service_control::Service& service);

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
