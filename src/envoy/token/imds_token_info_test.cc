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

#include "common/http/message_impl.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace Token {
namespace Test {

// Default token expiry time for ID tokens.
// Should match the value in `imds_token_info.cc`.
constexpr std::chrono::seconds kDefaultTokenExpiry(3599);

class ImdsTokenInfoTest : public testing::Test {
 protected:
  void SetUp() override { info_ = std::make_unique<ImdsTokenInfo>(); }

  TokenInfoPtr info_;
};

TEST_F(ImdsTokenInfoTest, SimpleSuccess) {
  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg =
      info_->prepareRequest("https://imds-url.com/path2");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_EQ(got_msg->bodyAsString(), R"()");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Method)
                ->value()
                .getStringView(),
            "GET");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Host)
                ->value()
                .getStringView(),
            "imds-url.com");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Path)
                ->value()
                .getStringView(),
            "/path2");
  Envoy::Http::LowerCaseString metadata_key("Metadata-Flavor");
  EXPECT_EQ(got_msg->headers().get(metadata_key)->value().getStringView(),
            "Google");
}

TEST_F(ImdsTokenInfoTest, IdentityTokenSuccess) {
  // Input.
  std::string response = R"(non-json-response)";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_FALSE(success);

  // Test identity token.
  success = info_->parseIdentityToken(response, &result);
  EXPECT_TRUE(success);
  EXPECT_EQ(result.token, "non-json-response");
  EXPECT_EQ(result.expiry_duration, kDefaultTokenExpiry);
}

TEST_F(ImdsTokenInfoTest, InvalidJsonResponse) {
  // Input.
  std::string response = R"({ "key": "value" })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_FALSE(success);
}

TEST_F(ImdsTokenInfoTest, AccessTokenSuccess) {
  // Input.
  std::string response =
      R"({ "access_token": "fake-access-token", "expires_in": 434432 })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_TRUE(success);
  EXPECT_EQ(result.token, "fake-access-token");
  EXPECT_NE(result.expiry_duration, kDefaultTokenExpiry);
}

}  // namespace Test
}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy
