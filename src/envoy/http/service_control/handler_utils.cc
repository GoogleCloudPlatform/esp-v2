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

#include <sstream>
#include <vector>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_split.h"
#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "common/http/utility.h"
#include "envoy/http/header_map.h"
#include "envoy/server/filter_config.h"
#include "extensions/filters/http/well_known_names.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "src/envoy/http/service_control/handler_utils.h"

using ::google::api::envoy::http::service_control::APIKeyLocation;
using ::google::api::envoy::http::service_control::Service;
using ::google::api_proxy::service_control::LatencyInfo;
using ::google::api_proxy::service_control::protocol::Protocol;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

namespace {

// Delimeter used in jwt payload key path
constexpr char kJwtPayLoadsDelimeter = '.';

constexpr char kContentTypeApplicationGrpcPrefix[] = "application/grpc";
const Http::LowerCaseString kContentTypeHeader{"content-type"};

inline int64_t convertNsToMs(std::chrono::nanoseconds ns) {
  return std::chrono::duration_cast<std::chrono::milliseconds>(ns).count();
}

bool extractAPIKeyFromQuery(const Http::HeaderMap& headers,
                            const std::string& query, bool& were_params_parsed,
                            Http::Utility::QueryParams& parsed_params,
                            std::string& api_key) {
  if (!were_params_parsed) {
    parsed_params = Http::Utility::parseQueryString(
        headers.Path()->value().getStringView());
    were_params_parsed = true;
  }

  const auto& it = parsed_params.find(query);
  if (it != parsed_params.end()) {
    api_key = it->second;
    return true;
  }
  return false;
}

bool extractAPIKeyFromHeader(const Http::HeaderMap& headers,
                             const std::string& header, std::string& api_key) {
  // TODO(qiwzhang): optimize this by using LowerCaseString at init.
  auto* entry = headers.get(Http::LowerCaseString(header));
  if (entry) {
    api_key = std::string(entry->value().getStringView());
    return true;
  }
  return false;
}

bool extractAPIKeyFromCookie(const Http::HeaderMap& headers,
                             const std::string& cookie, std::string& api_key) {
  std::string parsed_api_key = Http::Utility::parseCookieValue(headers, cookie);
  if (!parsed_api_key.empty()) {
    api_key = parsed_api_key;
    return true;
  }
  return false;
}

void extractJwtPayload(const ProtobufWkt::Value& value,
                       const std::string& jwt_payload_path,
                       std::string& info_jwt_payloads) {
  switch (value.kind_case()) {
    case ::google::protobuf::Value::kNullValue:
      absl::StrAppend(&info_jwt_payloads, jwt_payload_path, "=;");
      return;
    case ::google::protobuf::Value::kNumberValue:
      absl::StrAppend(&info_jwt_payloads, jwt_payload_path, "=",
                      std::to_string(static_cast<long>(value.number_value())),
                      ";");
      return;
    case ::google::protobuf::Value::kBoolValue:
      absl::StrAppend(&info_jwt_payloads, jwt_payload_path, "=",
                      value.bool_value() ? "true" : "false", ";");
      return;
    case ::google::protobuf::Value::kStringValue:
      absl::StrAppend(&info_jwt_payloads, jwt_payload_path, "=",
                      value.string_value(), ";");
      return;
    default:
      return;
  }
}

bool isGrpcRequest(absl::string_view content_type) {
  // Formally defined as:
  // `application/grpc(-web(-text))[+proto/+json/+thrift/{custom}]`
  //
  // The worst case is `application/grpc{custom}`. Just check the beginning.
  return absl::StartsWith(content_type, kContentTypeApplicationGrpcPrefix);
}

}  // namespace

void fillGCPInfo(
    const ::google::api::envoy::http::service_control::FilterConfig&
        filter_config,
    ::google::api_proxy::service_control::ReportRequestInfo& info) {
  if (!filter_config.has_gcp_attributes()) {
    return;
  }

  const auto& gcp_attributes = filter_config.gcp_attributes();
  if (!gcp_attributes.zone().empty()) {
    info.location = gcp_attributes.zone();
  }

  if (!gcp_attributes.platform().empty()) {
    info.compute_platform = gcp_attributes.platform();
  }
}

void fillLoggedHeader(
    const Http::HeaderMap* headers,
    const ::google::protobuf::RepeatedPtrField<::std::string>& log_headers,
    std::string& info_header_field) {
  if (headers == nullptr) {
    return;
  }
  for (const auto& log_header : log_headers) {
    auto* entry = headers->get(Http::LowerCaseString(log_header));
    if (entry) {
      absl::StrAppend(&info_header_field, log_header, "=",
                      entry->value().getStringView(), ";");
    }
  }
}

void fillLatency(const StreamInfo::StreamInfo& stream_info,
                 LatencyInfo& latency) {
  if (stream_info.requestComplete()) {
    latency.request_time_ms =
        convertNsToMs(stream_info.requestComplete().value());
  }

  auto start = stream_info.firstUpstreamTxByteSent();
  auto end = stream_info.lastUpstreamRxByteReceived();
  if (start && end && end.value() >= start.value()) {
    latency.backend_time_ms = convertNsToMs(end.value() - start.value());
  } else {
    // for cases like request is rejected at service control filter (does not
    // reach backend)
    latency.backend_time_ms = 0;
  }

  if (latency.request_time_ms >= latency.backend_time_ms) {
    latency.overhead_time_ms =
        latency.request_time_ms - latency.backend_time_ms;
  }
}

Protocol getFrontendProtocol(const Http::HeaderMap* response_headers,
                             const StreamInfo::StreamInfo& stream_info) {
  if (response_headers != nullptr) {
    auto content_type =
        Utils::extractHeader(*response_headers, kContentTypeHeader);
    if (isGrpcRequest(content_type)) {
      return Protocol::GRPC;
    }
  }

  if (!stream_info.protocol().has_value()) {
    return Protocol::UNKNOWN;
  }

  // TODO(toddbeckman) figure out HTTPS
  return Protocol::HTTP;
}

Protocol getBackendProtocol(const Service& service) {
  std::string protocol = service.backend_protocol();

  if (protocol == "http1" || protocol == "http2") {
    return Protocol::HTTP;
  }

  if (protocol == "grpc") {
    return Protocol::GRPC;
  }

  return Protocol::UNKNOWN;
}

// TODO(taoxuy): Add Unit Test
void fillJwtPayloads(const envoy::api::v2::core::Metadata& metadata,
                     const std::string& jwt_payload_metadata_name,
                     const ::google::protobuf::RepeatedPtrField<::std::string>&
                         jwt_payload_paths,
                     std::string& info_jwt_payloads) {
  for (const std::string& jwt_payload_path : jwt_payload_paths) {
    std::vector<std::string> steps =
        absl::StrSplit(jwt_payload_path, kJwtPayLoadsDelimeter);
    steps.insert(steps.begin(), jwt_payload_metadata_name);
    const ProtobufWkt::Value& value = Config::Metadata::metadataValue(
        metadata, HttpFilters::HttpFilterNames::get().JwtAuthn, steps);
    if (&value != &ProtobufWkt::Value::default_instance()) {
      extractJwtPayload(value, jwt_payload_path, info_jwt_payloads);
    }
  }
}

void fillJwtPayload(const envoy::api::v2::core::Metadata& metadata,
                    const std::string& jwt_payload_metadata_name,
                    const std::string& jwt_payload_path,
                    std::string& info_iss_or_aud) {
  std::vector<std::string> steps = {jwt_payload_metadata_name,
                                    jwt_payload_path};
  const ProtobufWkt::Value& value = Config::Metadata::metadataValue(
      metadata, HttpFilters::HttpFilterNames::get().JwtAuthn, steps);
  if (&value != &ProtobufWkt::Value::default_instance()) {
    absl::StrAppend(&info_iss_or_aud, value.string_value());
  }
}

bool extractAPIKey(
    const Http::HeaderMap& headers,
    const ::google::protobuf::RepeatedPtrField<
        ::google::api::envoy::http::service_control::APIKeyLocation>& locations,
    std::string& api_key) {
  // If checking multiple headers, cache the parameters so they are only parsed
  // once
  bool were_params_parsed{false};
  Http::Utility::QueryParams parsed_params;

  for (const auto& location : locations) {
    switch (location.key_case()) {
      case APIKeyLocation::kQuery:
        if (extractAPIKeyFromQuery(headers, location.query(),
                                   were_params_parsed, parsed_params, api_key))
          return true;
        break;
      case APIKeyLocation::kHeader:
        if (extractAPIKeyFromHeader(headers, location.header(), api_key))
          return true;
        break;
      case APIKeyLocation::kCookie:
        if (extractAPIKeyFromCookie(headers, location.cookie(), api_key))
          return true;
        break;
      case APIKeyLocation::KEY_NOT_SET:
        break;
    }
  }
  return false;
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
