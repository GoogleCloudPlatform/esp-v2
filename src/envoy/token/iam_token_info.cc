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

#include "src/envoy/token/iam_token_info.h"
#include "absl/strings/str_cat.h"
#include "common/http/headers.h"
#include "common/http/message_impl.h"
#include "src/envoy/utils/json_struct.h"

namespace Envoy {
namespace Extensions {
namespace Token {

using Utils::JsonStruct;

// Body field for the sequence of service accounts in a delegation chain.
constexpr char kDelegatesField[]("delegates");

// The prefix for delegates body field. They must have the following format:
// projects/-/serviceAccounts/{ACCOUNT_EMAIL_OR_UNIQUEID}, by
// https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/generateIdToken.
constexpr char kDelegatePrefix[]("projects/-/serviceAccounts/");

constexpr char kIncludeEmail[]("includeEmail");

// Body field to identify the scopes to be included in the OAuth 2.0 access
// token
constexpr char kScopesField[]("scope");

// Required header when fetching from the IAM server.
const Envoy::Http::LowerCaseString kAuthorizationKey("Authorization");

// Default token expiry time for ID tokens.
constexpr std::chrono::seconds kDefaultTokenExpiry(3599);

IamTokenInfo::IamTokenInfo(
    const ::google::protobuf::RepeatedPtrField<std::string>& delegates,
    const ::google::protobuf::RepeatedPtrField<std::string>& scopes,
    const GetTokenFunc access_token_fn)
    : delegates_(delegates),
      scopes_(scopes),
      access_token_fn_(access_token_fn) {}

Envoy::Http::MessagePtr IamTokenInfo::prepareRequest(
    absl::string_view token_url) const {
  const std::string access_token = access_token_fn_();
  // Wait for the access token to be set.
  if (access_token.empty()) {
    // This codes depends on access_token. This periodical pulling is not ideal.
    // But when both imds_token_subscriber and iam_token_subscriber register to
    // init_manager,  it will trigger both at the same time. For
    // easy implementation,  just using periodical pulling for now
    return nullptr;
  }

  absl::string_view host, path;
  Http::Utility::extractHostPathFromUri(token_url, host, path);
  Envoy::Http::HeaderMapImplPtr headers{new Envoy::Http::HeaderMapImpl{
      {Envoy::Http::Headers::get().Method, "POST"},
      {Envoy::Http::Headers::get().Host, std::string(host)},
      {Envoy::Http::Headers::get().Path, std::string(path)},
      {kAuthorizationKey, "Bearer " + access_token}}};

  Envoy::Http::MessagePtr message(
      new Envoy::Http::RequestMessageImpl(std::move(headers)));

  Envoy::ProtobufWkt::Value body;
  if (!delegates_.empty()) {
    insertStrListToProto(body, kDelegatesField, delegates_, kDelegatePrefix);
  }

  if (!scopes_.empty()) {
    insertStrListToProto(body, kScopesField, scopes_, "");
  }


  //  Include the service account email in the token.
  Envoy::ProtobufWkt::Value val;
  vals.set_bool_value(true);
  (*body.mutable_struct_value()->mutable_fields())[kIncludeEmail].Swap(&val);

  std::string bodyStr =
      MessageUtil::getJsonStringFromMessage(body, false, false);
  message->body() =
      std::make_unique<Buffer::OwnedImpl>(bodyStr.data(), bodyStr.size());

  return message;
}

// Access token response is a JSON payload in the format:
// {
//   "accessToken": "string",
//   "expireTime": "Timestamp"
// }
bool IamTokenInfo::parseAccessToken(absl::string_view response,
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
  parse_status = json_struct.getString("accessToken", &token);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `accessToken`: {}",
              parse_status.ToString());
    return false;
  }

  // Parse the expiry timestamp.
  ::google::protobuf::Timestamp expireTime;
  parse_status = json_struct.getTimestamp("expireTime", &expireTime);

  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `expireTime`: {}",
              parse_status.ToString());
    return false;
  }

  const std::chrono::seconds& expires_in = std::chrono::seconds(
      (expireTime - ::google::protobuf::util::TimeUtil::GetCurrentTime())
          .seconds());
  ret->token = token;
  ret->expiry_duration = expires_in;
  return true;
}

// Identity token response is a JSON payload in the format:
// {
//   "token": "string",
// }
bool IamTokenInfo::parseIdentityToken(absl::string_view response,
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
  parse_status = json_struct.getString("token", &token);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `token`: {}",
              parse_status.ToString());
    return false;
  }

  ret->token = token;
  ret->expiry_duration = kDefaultTokenExpiry;
  return true;
}

void IamTokenInfo::insertStrListToProto(
    Envoy::ProtobufWkt::Value& body, const std::string& key,
    const ::google::protobuf::RepeatedPtrField<std::string>& val_list,
    const absl::string_view& val_prefix) const {
  Envoy::ProtobufWkt::Value vals;
  for (const auto& val : val_list) {
    vals.mutable_list_value()->add_values()->set_string_value(
        absl::StrCat(val_prefix, val));
  }
  (*body.mutable_struct_value()->mutable_fields())[key].Swap(&vals);
}

}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy
