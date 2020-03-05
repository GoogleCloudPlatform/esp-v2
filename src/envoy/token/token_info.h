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

#include "common/common/logger.h"
#include "envoy/common/pure.h"
#include "envoy/http/message.h"

namespace Envoy {
namespace Extensions {
namespace Token {

struct TokenResult {
  std::string token;
  std::chrono::seconds expiry_duration;
};

// `TokenInfo` is an adapter that knows how to create requests and parse
// responses for identity and access tokens from various external APIs.
class TokenInfo : public Envoy::Logger::Loggable<Envoy::Logger::Id::init> {
 public:
  virtual ~TokenInfo() = default;

  virtual Envoy::Http::RequestMessagePtr prepareRequest(
      absl::string_view token_url) const PURE;
  virtual bool parseAccessToken(absl::string_view response,
                                TokenResult* ret) const PURE;
  virtual bool parseIdentityToken(absl::string_view response,
                                  TokenResult* ret) const PURE;
};

typedef std::unique_ptr<TokenInfo> TokenInfoPtr;

}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy
