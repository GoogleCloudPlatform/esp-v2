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

#include "src/envoy/token/token_info.h"

namespace Envoy {
namespace Extensions {
namespace Token {

// `ImdsTokenInfo` is a bridge `TokenInfo` for parsing
// identity and access tokens from the Instance Metadata Server.
class ImdsTokenInfo : public TokenInfo {
 public:
  ImdsTokenInfo();

  Envoy::Http::RequestMessagePtr prepareRequest(
      absl::string_view token_url) const override;
  bool parseAccessToken(absl::string_view response,
                        TokenResult* ret) const override;
  bool parseIdentityToken(absl::string_view response,
                          TokenResult* ret) const override;
};

}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy
