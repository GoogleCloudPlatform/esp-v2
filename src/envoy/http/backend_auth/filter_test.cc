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
#include "src/envoy/http/backend_auth/filter.h"

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "source/common/common/empty_string.h"
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/http/backend_auth/mocks.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/router/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

using ::testing::_;
using ::testing::NiceMock;
using ::testing::Return;
using ::testing::ReturnRef;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

const Envoy::Http::LowerCaseString kXForwardedAuthorization{
    "x-forwarded-authorization"};

/**
 * Base class for testing the Backend Auth filter. Makes a simple request
 * with no query parameters in the request URL.
 */
class BackendAuthFilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    mock_filter_config_parser_ =
        std::make_shared<NiceMock<MockFilterConfigParser>>();
    mock_filter_config_ = std::make_shared<NiceMock<MockFilterConfig>>();

    EXPECT_CALL(*mock_filter_config_, stats).WillRepeatedly(ReturnRef(stats_));
    EXPECT_CALL(*mock_filter_config_, cfg_parser)
        .WillRepeatedly(ReturnRef(*mock_filter_config_parser_));

    mock_route_ = std::make_shared<NiceMock<Envoy::Router::MockRoute>>();

    filter_ = std::make_unique<Filter>(mock_filter_config_);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);
  }

  void setPerRouteJwtAudience(const std::string& jwt_audience) {
    ::espv2::api::envoy::v10::http::backend_auth::PerRouteFilterConfig
        per_route_cfg;
    per_route_cfg.set_jwt_audience(jwt_audience);
    auto per_route = std::make_shared<PerRouteFilterConfig>(per_route_cfg);
    EXPECT_CALL(mock_decoder_callbacks_, route())
        .WillRepeatedly(Return(mock_route_));
    EXPECT_CALL(*mock_route_, mostSpecificPerFilterConfig(kFilterName))
        .WillRepeatedly(
            Invoke([per_route](const std::string&)
                       -> const Envoy::Router::RouteSpecificFilterConfig* {
              return per_route.get();
            }));
  }

  testing::NiceMock<Envoy::Stats::MockIsolatedStatsStore> scope_;
  FilterStats stats_{ALL_BACKEND_AUTH_FILTER_STATS(
      POOL_COUNTER_PREFIX(scope_, "backend_auth."))};

  std::shared_ptr<MockFilterConfigParser> mock_filter_config_parser_;
  std::shared_ptr<MockFilterConfig> mock_filter_config_;
  testing::NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks_;
  std::shared_ptr<NiceMock<Envoy::Router::MockRoute>> mock_route_;
  std::unique_ptr<Filter> filter_;
};

TEST_F(BackendAuthFilterTest, NoRouteRejectAllow) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};

  EXPECT_CALL(mock_decoder_callbacks_, route()).WillOnce(Return(nullptr));

  ASSERT_EQ(filter_->decodeHeaders(headers, false),
            Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendAuthFilterTest, NotPerRouteConfigAllowed) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  EXPECT_CALL(mock_decoder_callbacks_, route())
      .WillRepeatedly(Return(mock_route_));
  EXPECT_CALL(*mock_route_, mostSpecificPerFilterConfig(kFilterName))
      .WillRepeatedly(Return(nullptr));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          scope_, "backend_auth.allowed_by_auth_not_required");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendAuthFilterTest, EmptyTokenRejected) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  setPerRouteJwtAudience("this-is-audience");

  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken("this-is-audience"))
      .WillOnce(Return(nullptr));
  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(Envoy::Http::Code::InternalServerError,
                             "Token not found for audience: this-is-audience",
                             _, _, "backend_auth_missing_backend_token"));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_,
                                      "backend_auth.denied_by_no_token");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendAuthFilterTest, SucceedAppendToken) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  setPerRouteJwtAudience("this-is-audience");

  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken("this-is-audience"))
      .Times(1)
      .WillRepeatedly(Return(std::make_shared<std::string>("this-is-token")));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(headers.get(Envoy::Http::CustomHeaders::get().Authorization)[0]
                ->value()
                .getStringView(),
            "Bearer this-is-token");
  EXPECT_TRUE(headers.get(kXForwardedAuthorization).empty());
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "backend_auth.token_added");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendAuthFilterTest, SucceedTokenCopied) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"},
      {":path", "/books/1"},
      {"authorization", "Bearer origin-token"}};

  setPerRouteJwtAudience("this-is-audience");

  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken("this-is-audience"))
      .Times(1)
      .WillRepeatedly(Return(std::make_shared<std::string>("new-id-token")));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(headers.get(Envoy::Http::CustomHeaders::get().Authorization).size(),
            1);
  EXPECT_EQ(headers.get(Envoy::Http::CustomHeaders::get().Authorization)[0]
                ->value()
                .getStringView(),
            "Bearer new-id-token");
  ASSERT_EQ(headers.get(kXForwardedAuthorization).size(), 1);
  EXPECT_EQ(headers.get(kXForwardedAuthorization)[0]->value().getStringView(),
            "Bearer origin-token");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "backend_auth.token_added");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendAuthFilterTest, SucceedTokenOverridden) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"},
      {":path", "/books/1"},
      {"authorization", "Bearer origin-token"},
      {"x-forwarded-authorization", "Bearer untrusted-token"}};

  setPerRouteJwtAudience("this-is-audience");

  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken("this-is-audience"))
      .Times(1)
      .WillRepeatedly(Return(std::make_shared<std::string>("new-id-token")));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(headers.get(Envoy::Http::CustomHeaders::get().Authorization).size(),
            1);
  EXPECT_EQ(headers.get(Envoy::Http::CustomHeaders::get().Authorization)[0]
                ->value()
                .getStringView(),
            "Bearer new-id-token");
  ASSERT_EQ(headers.get(kXForwardedAuthorization).size(), 1);
  EXPECT_EQ(headers.get(kXForwardedAuthorization)[0]->value().getStringView(),
            "Bearer origin-token");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "backend_auth.token_added");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
