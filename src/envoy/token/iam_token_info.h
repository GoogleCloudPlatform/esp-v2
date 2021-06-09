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

#pragma once

#include "source/common/http/utility.h"
#include "google/protobuf/repeated_field.h"
#include "src/envoy/token/token_info.h"

namespace espv2 {
namespace envoy {
namespace token {

using GetTokenFunc = std::function<std::string()>;

// `IamTokenInfo` is a bridge `TokenInfo` for parsing
// identity and access tokens from the IAM server.
class IamTokenInfo : public TokenInfo {
 public:
  IamTokenInfo(
      const ::google::protobuf::RepeatedPtrField<std::string>& delegates,
      const ::google::protobuf::RepeatedPtrField<std::string>& scopes,
      const bool include_email, const GetTokenFunc access_token_fn);

  Envoy::Http::RequestMessagePtr prepareRequest(
      absl::string_view token_url) const override;
  bool parseAccessToken(absl::string_view response,
                        TokenResult* ret) const override;
  bool parseIdentityToken(absl::string_view response,
                          TokenResult* ret) const override;

 private:
  void insertStrListToProto(
      Envoy::ProtobufWkt::Value& body, const std::string& key,
      const ::google::protobuf::RepeatedPtrField<std::string>& val_list,
      const absl::string_view& val_prefix) const;

  const ::google::protobuf::RepeatedPtrField<std::string>& delegates_;
  const ::google::protobuf::RepeatedPtrField<std::string> scopes_;
  const bool include_email_;
  const GetTokenFunc access_token_fn_;
};

}  // namespace token
}  // namespace envoy
}  // namespace espv2
