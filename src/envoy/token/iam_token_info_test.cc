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
#include "common/common/empty_string.h"
#include "common/http/message_impl.h"
#include "gtest/gtest.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace token {
namespace test {

// Default token expiry time for ID tokens.
// Should match the value in `iam_token_info.cc`
constexpr std::chrono::seconds kDefaultTokenExpiry(3599);

class IamTokenInfoTest : public testing::Test {
 protected:
  void SetUp() override {}

  TokenInfoPtr info_;
};

TEST_F(IamTokenInfoTest, FailPreconditions) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  token::GetTokenFunc access_token_fn = []() { return Envoy::EMPTY_STRING; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, false, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg = info_->prepareRequest("iam-url");

  // Assert preconditions failed.
  EXPECT_EQ(got_msg, nullptr);
}

TEST_F(IamTokenInfoTest, SimpleSuccess) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  token::GetTokenFunc access_token_fn = []() { return "valid-access-token"; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, false, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg =
      info_->prepareRequest("https://iam-url.com/path1");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_EQ(got_msg->bodyAsString(), R"()");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Method)
                ->value()
                .getStringView(),
            "POST");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Host)
                ->value()
                .getStringView(),
            "iam-url.com");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Path)
                ->value()
                .getStringView(),
            "/path1");
  EXPECT_EQ(got_msg->headers()
                .get(Envoy::Http::Headers::get().Authorization)
                ->value()
                .getStringView(),
            "Bearer valid-access-token");
}

TEST_F(IamTokenInfoTest, SetDelegatesAndScopes) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  delegates.Add("delegate_foo");
  delegates.Add("delegate_bar");
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  scopes.Add("scope_foo");
  scopes.Add("scope_bar");
  token::GetTokenFunc access_token_fn = []() { return "valid-access-token"; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, false, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg = info_->prepareRequest("iam-url");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_TRUE(Envoy::TestUtility::jsonStringEqual(
      got_msg->bodyAsString(),
      R"({"scope":["scope_foo","scope_bar"],"delegates":["projects/-/serviceAccounts/delegate_foo","projects/-/serviceAccounts/delegate_bar"]})"));
}

TEST_F(IamTokenInfoTest, OnlySetDelegates) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  delegates.Add("delegate_foo");
  delegates.Add("delegate_bar");
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  token::GetTokenFunc access_token_fn = []() { return "valid-access-token"; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, false, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg = info_->prepareRequest("iam-url");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_TRUE(Envoy::TestUtility::jsonStringEqual(
      got_msg->bodyAsString(),
      R"({"delegates":["projects/-/serviceAccounts/delegate_foo","projects/-/serviceAccounts/delegate_bar"]})"));
}

TEST_F(IamTokenInfoTest, OnlySetScopes) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  scopes.Add("scope_foo");
  scopes.Add("scope_bar");
  token::GetTokenFunc access_token_fn = []() { return "valid-access-token"; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, false, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg = info_->prepareRequest("iam-url");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_TRUE(Envoy::TestUtility::jsonStringEqual(
      got_msg->bodyAsString(), R"({"scope":["scope_foo","scope_bar"]})"));
}

class IamParseTokenTest : public IamTokenInfoTest {
 protected:
  void SetUp() override {
    // None of these fields matter for parsing the token.
    ::google::protobuf::RepeatedPtrField<std::string> delegates;
    ::google::protobuf::RepeatedPtrField<std::string> scopes;
    token::GetTokenFunc access_token_fn = []() { return "fake-access-token"; };
    info_ = std::make_unique<IamTokenInfo>(delegates, scopes, false,
                                           access_token_fn);
  }
};

TEST_F(IamParseTokenTest, NonJsonResponse) {
  // Input.
  std::string response = R"({ non-json-response })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_FALSE(success);

  // Test identity token.
  success = info_->parseIdentityToken(response, &result);
  EXPECT_FALSE(success);
}

TEST_F(IamParseTokenTest, InvalidJsonResponse) {
  // Input.
  std::string response = R"({ "key": "value" })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_FALSE(success);

  // Test identity token.
  success = info_->parseIdentityToken(response, &result);
  EXPECT_FALSE(success);
}

TEST_F(IamParseTokenTest, IdentityTokenSuccess) {
  // Input.
  std::string response = R"({ "token": "fake-identity-token" })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseIdentityToken(response, &result);
  EXPECT_TRUE(success);
  EXPECT_EQ(result.token, "fake-identity-token");
  EXPECT_EQ(result.expiry_duration, kDefaultTokenExpiry);

  // Test identity token.
  success = info_->parseAccessToken(response, &result);
  EXPECT_FALSE(success);
}

TEST_F(IamParseTokenTest, AccessTokenSuccess) {
  // Input.
  std::string response =
      R"({ "accessToken": "fake-access-token", "expireTime": "2020-02-20T23:15:34-08:00" })";
  TokenResult result{};

  // Test access token.
  bool success = info_->parseAccessToken(response, &result);
  EXPECT_TRUE(success);
  EXPECT_EQ(result.token, "fake-access-token");
  EXPECT_NE(result.expiry_duration, kDefaultTokenExpiry);

  // Test identity token.
  success = info_->parseIdentityToken(response, &result);
  EXPECT_FALSE(success);
}

TEST_F(IamParseTokenTest, SetIncludeEmail) {
  // Create info that fails preconditions.
  ::google::protobuf::RepeatedPtrField<std::string> delegates;
  ::google::protobuf::RepeatedPtrField<std::string> scopes;
  scopes.Add("scope_foo");
  scopes.Add("scope_bar");
  token::GetTokenFunc access_token_fn = []() { return "valid-access-token"; };
  info_ =
      std::make_unique<IamTokenInfo>(delegates, scopes, true, access_token_fn);

  // Call function under test.
  Envoy::Http::RequestMessagePtr got_msg = info_->prepareRequest("iam-url");

  // Assert success.
  EXPECT_NE(got_msg, nullptr);
  EXPECT_TRUE(Envoy::TestUtility::jsonStringEqual(
      got_msg->bodyAsString(),
      R"({"includeEmail":true,"scope":["scope_foo","scope_bar"]})"));
}

}  // namespace test
}  // namespace token
}  // namespace envoy
}  // namespace espv2
