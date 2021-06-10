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

#include "src/envoy/http/service_control/handler_utils.h"

#include <sstream>
#include <vector>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_split.h"
#include "absl/types/optional.h"
#include "api/envoy/v9/http/service_control/config.pb.h"
#include "envoy/grpc/status.h"
#include "envoy/http/header_map.h"
#include "envoy/server/filter_config.h"
#include "source/common/common/logger.h"
#include "source/common/grpc/common.h"
#include "source/common/http/header_utility.h"
#include "source/common/http/utility.h"
#include "source/extensions/filters/http/well_known_names.h"
#include "src/api_proxy/service_control/request_builder.h"

using ::espv2::api::envoy::v9::http::service_control::ApiKeyLocation;
using ::espv2::api::envoy::v9::http::service_control::Service;
using ::espv2::api_proxy::service_control::LatencyInfo;
using ::espv2::api_proxy::service_control::protocol::Protocol;
using ::google::protobuf::util::StatusCode;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

namespace {

// Delimeter used in jwt payload key path
constexpr char kJwtPayLoadsDelimeter = '.';

constexpr char kContentTypeApplicationGrpcPrefix[] = "application/grpc";
const Envoy::Http::LowerCaseString kContentTypeHeader{"content-type"};

inline int64_t convertNsToMs(std::chrono::nanoseconds ns) {
  return std::chrono::duration_cast<std::chrono::milliseconds>(ns).count();
}

bool extractAPIKeyFromQuery(const Envoy::Http::RequestHeaderMap& headers,
                            const std::string& query, bool& were_params_parsed,
                            Envoy::Http::Utility::QueryParams& parsed_params,
                            std::string& api_key) {
  if (!were_params_parsed) {
    if (headers.Path() == nullptr) {
      return false;
    }
    parsed_params = Envoy::Http::Utility::parseQueryString(
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

bool extractAPIKeyFromHeader(const Envoy::Http::RequestHeaderMap& headers,
                             const std::string& header, std::string& api_key) {
  // TODO(qiwzhang): optimize this by using LowerCaseString at init.
  auto entry = headers.get(Envoy::Http::LowerCaseString(header));
  if (!entry.empty()) {
    api_key = std::string(entry[0]->value().getStringView());
    return true;
  }
  return false;
}

bool extractAPIKeyFromCookie(const Envoy::Http::RequestHeaderMap& headers,
                             const std::string& cookie, std::string& api_key) {
  std::string parsed_api_key =
      Envoy::Http::Utility::parseCookieValue(headers, cookie);
  if (!parsed_api_key.empty()) {
    api_key = parsed_api_key;
    return true;
  }
  return false;
}

void extractJwtPayload(const Envoy::ProtobufWkt::Value& value,
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
    const ::espv2::api::envoy::v9::http::service_control::FilterConfig&
        filter_config,
    ::espv2::api_proxy::service_control::ReportRequestInfo& info) {
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

  if (!gcp_attributes.project_id().empty()) {
    info.project_id = gcp_attributes.project_id();
  }
}

void fillLoggedHeader(
    const Envoy::Http::HeaderMap* headers,
    const ::google::protobuf::RepeatedPtrField<::std::string>& log_headers,
    std::string& info_header_field) {
  if (headers == nullptr) {
    return;
  }
  for (const auto& log_header : log_headers) {
    const auto entry = Envoy::Http::HeaderUtility::getAllOfHeaderAsString(
        *headers, Envoy::Http::LowerCaseString(log_header));
    if (entry.result().has_value()) {
      absl::StrAppend(&info_header_field, log_header, "=",
                      entry.result().value(), ";");
    }
  }
}

void fillLatency(const Envoy::StreamInfo::StreamInfo& stream_info,
                 LatencyInfo& latency,
                 ServiceControlFilterStats& filter_stats) {
  if (stream_info.requestComplete()) {
    latency.request_time_ms =
        convertNsToMs(stream_info.requestComplete().value());
  }

  auto start = stream_info.firstUpstreamTxByteSent();
  auto end = stream_info.lastUpstreamRxByteReceived();
  if (start && end) {
    ENVOY_BUG(end.value() >= start.value(), "End time < start time");
    latency.backend_time_ms = convertNsToMs(end.value() - start.value());
  } else if (!start) {
    // The request is rejected at a filter (does not
    // reach backend).
    latency.backend_time_ms = 0;
  }

  if (latency.request_time_ms >= 0) {
    if (latency.backend_time_ms >= 0) {
      // Calculation based on backend response timing.
      latency.overhead_time_ms =
          latency.request_time_ms - latency.backend_time_ms;
    } else if (start) {
      // Less accurate calculation based on overhead timing.
      latency.overhead_time_ms = convertNsToMs(start.value());
      latency.backend_time_ms =
          latency.request_time_ms - latency.overhead_time_ms;
    }
  }

  // FIXME(https://github.com/envoyproxy/envoy/issues/16684): When stats are
  // reported correctly, remove this hack. It results in under-reporting
  // overhead latency and over-reporting backend latency.
  if (!start && stream_info.responseCodeDetails() &&
      (stream_info.responseCodeDetails().value() ==
           Envoy::StreamInfo::ResponseCodeDetails::get().UpstreamTimeout ||
       stream_info.responseCodeDetails().value() ==
           Envoy::StreamInfo::ResponseCodeDetails::get()
               .UpstreamPerTryTimeout)) {
    latency.backend_time_ms = latency.overhead_time_ms;
    latency.overhead_time_ms = 0;
  }

  filter_stats.filter_.request_time_.recordValue(latency.request_time_ms);
  filter_stats.filter_.backend_time_.recordValue(latency.backend_time_ms);
  filter_stats.filter_.overhead_time_.recordValue(latency.overhead_time_ms);
}

Protocol getFrontendProtocol(
    const Envoy::Http::ResponseHeaderMap* response_headers,
    const Envoy::StreamInfo::StreamInfo& stream_info) {
  if (response_headers != nullptr) {
    auto content_type = response_headers->getContentTypeValue();

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
void fillJwtPayloads(const ::envoy::config::core::v3::Metadata& metadata,
                     const std::string& jwt_payload_metadata_name,
                     const ::google::protobuf::RepeatedPtrField<::std::string>&
                         jwt_payload_paths,
                     std::string& info_jwt_payloads) {
  for (const std::string& jwt_payload_path : jwt_payload_paths) {
    std::vector<std::string> steps =
        absl::StrSplit(jwt_payload_path, kJwtPayLoadsDelimeter);
    steps.insert(steps.begin(), jwt_payload_metadata_name);
    const Envoy::ProtobufWkt::Value& value =
        Envoy::Config::Metadata::metadataValue(
            &metadata,
            Envoy::Extensions::HttpFilters::HttpFilterNames::get().JwtAuthn,
            steps);
    if (&value != &Envoy::ProtobufWkt::Value::default_instance()) {
      extractJwtPayload(value, jwt_payload_path, info_jwt_payloads);
    }
  }
}

void fillJwtPayload(const ::envoy::config::core::v3::Metadata& metadata,
                    const std::string& jwt_payload_metadata_name,
                    const std::string& jwt_payload_path,
                    std::string& info_iss_or_aud) {
  std::vector<std::string> steps = {jwt_payload_metadata_name,
                                    jwt_payload_path};
  const Envoy::ProtobufWkt::Value& value =
      Envoy::Config::Metadata::metadataValue(
          &metadata,
          Envoy::Extensions::HttpFilters::HttpFilterNames::get().JwtAuthn,
          steps);
  if (&value != &Envoy::ProtobufWkt::Value::default_instance()) {
    absl::StrAppend(&info_iss_or_aud, value.string_value());
  }
}

bool extractAPIKey(
    const Envoy::Http::RequestHeaderMap& headers,
    const ::google::protobuf::RepeatedPtrField<
        ::espv2::api::envoy::v9::http::service_control::ApiKeyLocation>&
        locations,
    std::string& api_key) {
  // If checking multiple headers, cache the parameters so they are only parsed
  // once
  bool were_params_parsed{false};
  Envoy::Http::Utility::QueryParams parsed_params;

  for (const auto& location : locations) {
    switch (location.key_case()) {
      case ApiKeyLocation::kQuery:
        if (extractAPIKeyFromQuery(headers, location.query(),
                                   were_params_parsed, parsed_params, api_key))
          return true;
        break;
      case ApiKeyLocation::kHeader:
        if (extractAPIKeyFromHeader(headers, location.header(), api_key))
          return true;
        break;
      case ApiKeyLocation::kCookie:
        if (extractAPIKeyFromCookie(headers, location.cookie(), api_key))
          return true;
        break;
      case ApiKeyLocation::KEY_NOT_SET:
        break;
    }
  }
  return false;
}

void fillStatus(const Envoy::Http::ResponseHeaderMap* response_headers,
                const Envoy::Http::ResponseTrailerMap* response_trailers,
                const Envoy::StreamInfo::StreamInfo& stream_info,
                ::espv2::api_proxy::service_control::ReportRequestInfo& info) {
  // When response code is missing, we want to set to a value that makes it
  // obvious, so 0.
  info.http_response_code = stream_info.responseCode().value_or(0);

  if (info.http_response_code != 200 ||
      info.frontend_protocol != Protocol::GRPC) {
    return;
  }

  absl::optional<Envoy::Grpc::Status::GrpcStatus> status;
  if (response_trailers) {
    // grpc-status should be in the trailers.
    status = Envoy::Grpc::Common::getGrpcStatus(*response_trailers, false);
  }
  if (!status.has_value() && response_headers) {
    // Envoy http2 codec treats trailers-only response as headers-only response.
    status = Envoy::Grpc::Common::getGrpcStatus(*response_headers, false);
  }
  if (!status.has_value()) {
    return;
  }

  if (status.value() == Envoy::Grpc::Status::WellKnownGrpcStatus::InvalidCode) {
    return;
  }

  info.grpc_response_code = static_cast<StatusCode>(status.value());
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
