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
#include "src/envoy/http/path_rewrite/filter.h"

#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/http/path_rewrite/mocks.h"
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
namespace path_rewrite {

class FilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    filter_config_ = std::make_shared<FilterConfig>("", scope_);
    mock_route_ = std::make_shared<NiceMock<Envoy::Router::MockRoute>>();

    filter_ = std::make_unique<Filter>(filter_config_);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);

    auto mock_parser = std::make_unique<NiceMock<MockConfigParser>>();
    raw_mock_parser_ = mock_parser.get();
    per_route_config_ =
        std::make_shared<PerRouteFilterConfig>(std::move(mock_parser));
  }

  NiceMock<Envoy::Stats::MockIsolatedStatsStore> scope_;

  std::shared_ptr<NiceMock<MockConfigParser>> mock_config_parser_;
  std::shared_ptr<FilterConfig> filter_config_;
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks_;
  std::shared_ptr<NiceMock<Envoy::Router::MockRoute>> mock_route_;
  std::unique_ptr<Filter> filter_;
  std::shared_ptr<PerRouteFilterConfig> per_route_config_;
  MockConfigParser* raw_mock_parser_;
};

TEST_F(FilterTest, NoPathHeaderBlocked) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"}};

  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(Envoy::Http::Code::BadRequest,
                             "No path in request headers", _, _,
                             "path_rewrite_bad_request{MISSING_PATH}"));
  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP and reject the request.
  EXPECT_EQ(headers.Path(), nullptr);
  EXPECT_EQ(headers.EnvoyOriginalPath(), nullptr);
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "path_rewrite.denied_by_no_path");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, DecodeHeadersOverflowWildcard) {
  // Construct a request with a long path: "/aaa...aaa/long"
  std::string a_chars(9000, 'a');
  std::string path = absl::StrCat("/", a_chars, "/long");

  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", path}};

  // Filter should reject the request.
  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(Envoy::Http::Code::BadRequest,
                             "Path is too long, max allowed size is 8192.", _,
                             _, "path_rewrite_bad_request{OVERSIZE_PATH}"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers, true));

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_,
                                      "path_rewrite.denied_by_oversize_path");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, FragmentPathHeaderBlocked) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1#abc"}};

  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(
                  Envoy::Http::Code::BadRequest,
                  "Path cannot contain fragment identifier (#)", _, _,
                  "path_rewrite_bad_request{PATH_WITH_FRAGMENT_IDENTIFIER}"));
  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP and reject the request.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/books/1#abc");
  EXPECT_EQ(headers.EnvoyOriginalPath(), nullptr);
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_,
                                      "path_rewrite.denied_by_invalid_path");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, NoRouteRejected) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};

  EXPECT_CALL(mock_decoder_callbacks_, route()).WillOnce(Return(nullptr));

  EXPECT_CALL(mock_decoder_callbacks_.stream_info_,
              setResponseFlag(Envoy::StreamInfo::ResponseFlag::NoRouteFound));
  EXPECT_CALL(
      mock_decoder_callbacks_,
      sendLocalReply(Envoy::Http::Code::NotFound,
                     "Request `GET /books/1` is not defined by this API.", _, _,
                     "path_rewrite_undefined_request"));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_,
                                      "path_rewrite.denied_by_no_route");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, NotPerRouteConfigNotChanged) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  EXPECT_CALL(mock_decoder_callbacks_, route())
      .WillRepeatedly(Return(mock_route_));
  EXPECT_CALL(mock_route_->route_entry_, perFilterConfig(kFilterName))
      .WillRepeatedly(Return(nullptr));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // path not changed.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/books/1");
  EXPECT_EQ(headers.EnvoyOriginalPath(), nullptr);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "path_rewrite.path_not_changed");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, RejectedByMismatchUrlTemplate) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  EXPECT_CALL(mock_decoder_callbacks_, route())
      .WillRepeatedly(Return(mock_route_));
  EXPECT_CALL(mock_route_->route_entry_, perFilterConfig(kFilterName))
      .WillRepeatedly(Return(per_route_config_.get()));

  // Mismatch
  EXPECT_CALL(*raw_mock_parser_, rewrite("/books/1", _))
      .WillOnce(Invoke(
          [](absl::string_view, std::string&) -> bool { return false; }));
  EXPECT_CALL(*raw_mock_parser_, url_template()).WillOnce(Return("/bar/{xyz}"));

  // The request is rejected
  EXPECT_CALL(
      mock_decoder_callbacks_,
      sendLocalReply(Envoy::Http::Code::InternalServerError,
                     "Request `GET /books/1` is getting wrong route config", _,
                     _, "path_rewrite_wrong_route_config"));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // path not changed.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/books/1");
  EXPECT_EQ(headers.EnvoyOriginalPath(), nullptr);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          scope_, "path_rewrite.denied_by_url_template_mismatch");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(FilterTest, PathUpdated) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  EXPECT_CALL(mock_decoder_callbacks_, route())
      .WillRepeatedly(Return(mock_route_));
  EXPECT_CALL(mock_route_->route_entry_, perFilterConfig(kFilterName))
      .WillRepeatedly(Return(per_route_config_.get()));

  // path rewrite ok
  EXPECT_CALL(*raw_mock_parser_, rewrite("/books/1", _))
      .WillOnce(Invoke([](absl::string_view, std::string& new_path) -> bool {
        new_path = "/tree/2";
        return true;
      }));

  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // path changed.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/tree/2");
  EXPECT_EQ(headers.EnvoyOriginalPath()->value().getStringView(), "/books/1");

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(scope_, "path_rewrite.path_changed");
  EXPECT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
