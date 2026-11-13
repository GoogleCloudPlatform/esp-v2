// Copyright 2026 Google LLC
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

#include "src/envoy/http/header_sanitizer/filter.h"

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "test/mocks/http/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace header_sanitizer {
namespace {

using Envoy::Http::LowerCaseString;
using Envoy::Http::MockStreamDecoderFilterCallbacks;
using Envoy::Http::TestRequestHeaderMapImpl;
using ::testing::NiceMock;

const LowerCaseString kXEndpointApiUserInfo{"x-endpoint-api-userinfo"};
const LowerCaseString kXForwardedAuthorization{"x-forwarded-authorization"};
const LowerCaseString kXEndpointApiConsumerType{"x-endpoint-api-consumer-type"};
const LowerCaseString kXEndpointApiConsumerNumber{
    "x-endpoint-api-consumer-number"};
const LowerCaseString kHttpMethodOverrideHeader{"x-http-method-override"};
const LowerCaseString kCustomHeader{"x-custom-header"};

class HeaderSanitizerFilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    filter_ = std::make_unique<Filter>();
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);
  }

  NiceMock<MockStreamDecoderFilterCallbacks> mock_decoder_callbacks_;
  std::unique_ptr<Filter> filter_;
};

TEST_F(HeaderSanitizerFilterTest, StripsSpoofedProxyAssertedHeaders) {
  TestRequestHeaderMapImpl headers{
      {":method", "GET"},
      {":path", "/books/1"},
      {"x-endpoint-api-userinfo", "forged-userinfo"},
      {"x-forwarded-authorization", "Bearer forged-token"},
      {"x-endpoint-api-consumer-type", "PROJECT"},
      {"x-endpoint-api-consumer-number", "123456"},
      {"x-custom-header", "preserved-value"},
  };

  EXPECT_CALL(mock_decoder_callbacks_.downstream_callbacks_, clearRouteCache())
      .Times(0);

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, false));

  EXPECT_TRUE(headers.get(kXEndpointApiUserInfo).empty());
  EXPECT_TRUE(headers.get(kXForwardedAuthorization).empty());
  EXPECT_TRUE(headers.get(kXEndpointApiConsumerType).empty());
  EXPECT_TRUE(headers.get(kXEndpointApiConsumerNumber).empty());
  ASSERT_EQ(headers.get(kCustomHeader).size(), 1);
  EXPECT_EQ(headers.get(kCustomHeader)[0]->value().getStringView(),
            "preserved-value");
}

TEST_F(HeaderSanitizerFilterTest, HandlesHttpMethodOverrideAndStripsHeaders) {
  TestRequestHeaderMapImpl headers{
      {":method", "POST"},
      {":path", "/books/1"},
      {"x-http-method-override", "DELETE"},
      {"x-endpoint-api-userinfo", "forged-userinfo"},
  };

  EXPECT_CALL(mock_decoder_callbacks_.downstream_callbacks_, clearRouteCache())
      .Times(1);

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, false));

  EXPECT_EQ(headers.Method()->value().getStringView(), "DELETE");
  EXPECT_TRUE(headers.get(kHttpMethodOverrideHeader).empty());
  EXPECT_TRUE(headers.get(kXEndpointApiUserInfo).empty());
}

}  // namespace
}  // namespace header_sanitizer
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
