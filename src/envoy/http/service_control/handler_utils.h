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
#include "src/envoy/utils/filter_state_utils.h"
#include "src/envoy/utils/http_header_utils.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// Searches the headers at the given locations and sets the `api_key` if one is
// found.
//
// Returns whether an `api_key` was found.
bool extractAPIKey(
    const Http::HeaderMap& headers,
    const ::google::protobuf::RepeatedPtrField<
        ::google::api::envoy::http::service_control::ApiKeyLocation>& locations,
    std::string& api_key);

// Adds information from the `FilterConfig`'s gcp_attributes to the given info.
void fillGCPInfo(
    const ::google::api::envoy::http::service_control::FilterConfig&
        filter_config,
    ::google::api_proxy::service_control::ReportRequestInfo& info);

// Searches the `headers` for the given `log_headers` and appends all matches
// to the string provided.
void fillLoggedHeader(
    const Http::HeaderMap* headers,
    const ::google::protobuf::RepeatedPtrField<::std::string>& log_headers,
    std::string& info_header_field);

// Fills the `request_time_ms`, `backend_time_ms`, and `overhead_time_ms` of the
// info provided.
void fillLatency(const StreamInfo::StreamInfo& stream_info,
                 ::google::api_proxy::service_control::LatencyInfo& latency);

// Fills the jwt payload of the info provided
void fillJwtPayloads(const envoy::config::core::v3alpha::Metadata& metadata,
                     const std::string& jwt_payload_metadata_name,
                     const ::google::protobuf::RepeatedPtrField<::std::string>&
                         jwt_payload_paths,
                     std::string& info_jwt_payloads);

void fillJwtPayload(const envoy::config::core::v3alpha::Metadata& metadata,
                    const std::string& jwt_payload_metadata_name,
                    const std::string& jwt_payload_path,
                    std::string& info_iss_or_aud);

// Returns the protocol of the frontend request or UNKNOWN if not found
::google::api_proxy::service_control::protocol::Protocol getFrontendProtocol(
    const Http::HeaderMap* response_headers,
    const StreamInfo::StreamInfo& stream_info);

// Returns the protocol of the backend service or UNKNOWN if not found
::google::api_proxy::service_control::protocol::Protocol getBackendProtocol(
    const ::google::api::envoy::http::service_control::Service& service);

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
