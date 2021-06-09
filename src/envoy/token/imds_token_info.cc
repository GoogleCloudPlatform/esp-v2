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

#include "src/envoy/token/imds_token_info.h"

#include "absl/strings/str_cat.h"
#include "source/common/http/headers.h"
#include "source/common/http/message_impl.h"
#include "source/common/http/utility.h"
#include "src/envoy/utils/json_struct.h"

namespace espv2 {
namespace envoy {
namespace token {

using utils::JsonStruct;

// Required header when fetching from IMDS.
const Envoy::Http::LowerCaseString kMetadataFlavorKey("Metadata-Flavor");
constexpr char kMetadataFlavor[]{"Google"};

// Default token expiry time for ID tokens.
constexpr std::chrono::seconds kDefaultTokenExpiry(3599);

ImdsTokenInfo::ImdsTokenInfo() {}

Envoy::Http::RequestMessagePtr ImdsTokenInfo::prepareRequest(
    absl::string_view token_url) const {
  absl::string_view host, path;
  Envoy::Http::Utility::extractHostPathFromUri(token_url, host, path);

  auto headers =
      Envoy::Http::createHeaderMap<Envoy::Http::RequestHeaderMapImpl>(
          {{Envoy::Http::Headers::get().Method, "GET"},
           {Envoy::Http::Headers::get().Host, std::string(host)},
           {Envoy::Http::Headers::get().Path, std::string(path)},
           {kMetadataFlavorKey, kMetadataFlavor}});

  Envoy::Http::RequestMessagePtr message(
      new Envoy::Http::RequestMessageImpl(std::move(headers)));

  return message;
}

// Access token response is a JSON payload in the format:
// {
//   "access_token": "string",
//   "expires_in": uint
// }
bool ImdsTokenInfo::parseAccessToken(absl::string_view response,
                                     TokenResult* ret) const {
  // Parse the JSON into a proto.
  ::google::protobuf::Struct response_pb;
  ::google::protobuf::util::Status parse_status =
      ::google::protobuf::util::JsonStringToMessage(std::string(response),
                                                    &response_pb);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed: {}", parse_status.ToString());
    return false;
  }
  JsonStruct json_struct(response_pb);

  // Parse the token.
  std::string token;
  parse_status = json_struct.getString("access_token", &token);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `access_token`: {}",
              parse_status.ToString());
    return false;
  }

  // Parse the expiry duration.
  int expires_seconds;
  parse_status = json_struct.getInteger("expires_in", &expires_seconds);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `expires_in`: {}",
              parse_status.ToString());
    return false;
  }

  const std::chrono::seconds expires_in = std::chrono::seconds(expires_seconds);
  ret->token = token;
  ret->expiry_duration = expires_in;
  return true;
}

// Identity token response is just the raw string, no JSON to parse.
bool ImdsTokenInfo::parseIdentityToken(absl::string_view response,
                                       TokenResult* ret) const {
  ret->token = std::string(response);
  ret->expiry_duration = kDefaultTokenExpiry;
  return true;
}

}  // namespace token
}  // namespace envoy
}  // namespace espv2
