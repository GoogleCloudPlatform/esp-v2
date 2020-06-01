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

#include "common/common/empty_string.h"
#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/http/backend_auth/mocks.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

using ::testing::_;
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
    filter_ = std::make_unique<Filter>(mock_filter_config_);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);
  }

  std::shared_ptr<MockFilterConfigParser> mock_filter_config_parser_;
  std::shared_ptr<MockFilterConfig> mock_filter_config_;
  testing::NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks_;
  std::unique_ptr<Filter> filter_;
};

TEST_F(BackendAuthFilterTest, NoOperationName) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};

  EXPECT_CALL(*mock_filter_config_, cfg_parser).Times(0);

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendAuthFilterTest, NotHaveAudience) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "operation-without-audience");

  EXPECT_CALL(*mock_filter_config_, cfg_parser)
      .Times(1)
      .WillRepeatedly(testing::ReturnRef(*mock_filter_config_parser_));
  EXPECT_CALL(*mock_filter_config_parser_, getAudience)
      .Times(1)
      .WillRepeatedly(testing::Return(nullptr));
  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken).Times(0);

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendAuthFilterTest, HasAudienceButGetsEmptyToken) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "operation-with-audience");

  EXPECT_CALL(*mock_filter_config_, cfg_parser)
      .WillRepeatedly(testing::ReturnRef(*mock_filter_config_parser_));
  EXPECT_CALL(*mock_filter_config_parser_, getAudience)
      .Times(1)
      .WillRepeatedly(testing::Return("this-is-audience"));
  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken)
      .Times(1)
      .WillRepeatedly(testing::Return(nullptr));
  EXPECT_CALL(
      mock_decoder_callbacks_.stream_info_,
      setResponseFlag(
          Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);
}

TEST_F(BackendAuthFilterTest, SucceedAppendToken) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "operation-with-audience");
  testing::NiceMock<Envoy::Stats::MockIsolatedStatsStore> scope;
  const std::string prefix = Envoy::EMPTY_STRING;
  FilterStats filter_stats{
      ALL_BACKEND_AUTH_FILTER_STATS(POOL_COUNTER_PREFIX(scope, prefix))};

  EXPECT_CALL(*mock_filter_config_, cfg_parser)
      .WillRepeatedly(testing::ReturnRef(*mock_filter_config_parser_));
  EXPECT_CALL(*mock_filter_config_, stats)
      .WillRepeatedly(testing::ReturnRef(filter_stats));

  EXPECT_CALL(*mock_filter_config_parser_, getAudience)
      .Times(1)
      .WillRepeatedly(testing::Return("this-is-audience"));
  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken)
      .Times(1)
      .WillRepeatedly(
          testing::Return(std::make_shared<std::string>("this-is-token")));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(headers.Authorization()->value().getStringView(),
            "Bearer this-is-token");
  EXPECT_EQ(headers.get(kXForwardedAuthorization), nullptr);
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendAuthFilterTest, SucceedTokenCopied) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"},
      {":path", "/books/1"},
      {"authorization", "Bearer origin-token"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "operation-with-audience");
  testing::NiceMock<Envoy::Stats::MockIsolatedStatsStore> scope;
  const std::string prefix = Envoy::EMPTY_STRING;
  FilterStats filter_stats{
      ALL_BACKEND_AUTH_FILTER_STATS(POOL_COUNTER_PREFIX(scope, prefix))};

  EXPECT_CALL(*mock_filter_config_, cfg_parser)
      .WillRepeatedly(testing::ReturnRef(*mock_filter_config_parser_));
  EXPECT_CALL(*mock_filter_config_, stats)
      .WillRepeatedly(testing::ReturnRef(filter_stats));

  EXPECT_CALL(*mock_filter_config_parser_, getAudience)
      .Times(1)
      .WillRepeatedly(testing::Return("this-is-audience"));
  EXPECT_CALL(*mock_filter_config_parser_, getJwtToken)
      .Times(1)
      .WillRepeatedly(
          testing::Return(std::make_shared<std::string>("new-id-token")));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(headers.Authorization()->value().getStringView(),
            "Bearer new-id-token");
  EXPECT_EQ(headers.get(kXForwardedAuthorization)->value().getStringView(),
            "Bearer origin-token");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
