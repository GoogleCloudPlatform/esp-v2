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

#include "src/envoy/http/path_matcher/filter.h"
#include "common/common/empty_string.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {
namespace {

using Envoy::Http::MockStreamDecoderFilterCallbacks;
using Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::TextFormat;

const char kFilterConfig[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  pattern {
    http_method: "GET"
    uri_template: "/bar"
  }
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Foo"
  pattern {
    http_method: "GET"
    uri_template: "/foo/{foo_bar}"
  }
  path_parameter_extraction {
    snake_to_json_segments {
      key: "foo_bar"
      value: "fooBar"
    }
  }
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Long"
  pattern {
    http_method: "GET"
    uri_template: "/**/long"
  }
})";

class PathMatcherFilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
    ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfig, &config_pb));
    config_ = std::make_shared<FilterConfig>(config_pb, Envoy::EMPTY_STRING,
                                             mock_factory_context_);

    filter_ = std::make_unique<Filter>(config_);
    filter_->setDecoderFilterCallbacks(mock_cb_);
  }

  std::unique_ptr<Filter> filter_;
  FilterConfigSharedPtr config_;
  testing::NiceMock<MockFactoryContext> mock_factory_context_;
  testing::NiceMock<MockStreamDecoderFilterCallbacks> mock_cb_;
};

TEST_F(PathMatcherFilterTest, DecodeHeadersWithOperation) {
  // Test: a request matches a operation
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/bar"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, false));

  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateOperation),
            "1.cloudesf_testing_cloud_goog.Bar");
  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateQueryParams),
            Envoy::EMPTY_STRING);

  EXPECT_EQ(1L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.allowed")
                    ->value());
  EXPECT_EQ(0L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.denied")
                    ->value());

  Envoy::Buffer::OwnedImpl data(Envoy::EMPTY_STRING);
  EXPECT_EQ(Envoy::Http::FilterDataStatus::Continue,
            filter_->decodeData(data, false));

  Envoy::Http::TestRequestTrailerMapImpl trailers;
  EXPECT_EQ(Envoy::Http::FilterTrailersStatus::Continue,
            filter_->decodeTrailers(trailers));
}

TEST_F(PathMatcherFilterTest, DecodeHeadersWithMethodOverride) {
  // Test: a request with a method override matches a operation
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "POST"},
      {":path", "/bar"},
      {"x-http-method-override", "GET"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, true));

  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateOperation),
            "1.cloudesf_testing_cloud_goog.Bar");

  EXPECT_EQ(1L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.allowed")
                    ->value());
  EXPECT_EQ(0L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.denied")
                    ->value());
}

TEST_F(PathMatcherFilterTest, DecodeHeadersExtractParameters) {
  // Test: a request needs to extract parameters
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/foo/123"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, true));

  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateOperation),
            "1.cloudesf_testing_cloud_goog.Foo");
  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateQueryParams),
            "fooBar=123");

  EXPECT_EQ(1L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.allowed")
                    ->value());
  EXPECT_EQ(0L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.denied")
                    ->value());
}

TEST_F(PathMatcherFilterTest, DecodeHeadersNoMatch) {
  // Test: a request no match
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "POST"},
                                                {":path", "/bar"}};

  // Filter should reject this request
  EXPECT_CALL(
      mock_cb_.stream_info_,
      setResponseFlag(
          Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService));

  EXPECT_CALL(
      mock_cb_,
      sendLocalReply(Envoy::Http::Code::NotFound,
                     "Request `POST /bar` is not defined in the service spec.",
                     _, _, "path_not_defined"));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers, true));

  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateOperation),
            Envoy::EMPTY_STRING);

  EXPECT_EQ(0L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.allowed")
                    ->value());
  EXPECT_EQ(1L, Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                                "path_matcher.denied")
                    ->value());
}

TEST_F(PathMatcherFilterTest, DecodeHeadersMissingHeaders) {
  Envoy::Http::TestRequestHeaderMapImpl missingPath{{":method", "POST"}};
  Envoy::Http::TestRequestHeaderMapImpl missingMethod{{":path", "/bar"}};

  // Filter should reject this request
  EXPECT_CALL(mock_cb_.stream_info_,
              setResponseFlag(
                  Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService))
      .Times(2);

  EXPECT_CALL(mock_cb_, sendLocalReply(Envoy::Http::Code::BadRequest,
                                       "No path in request headers.", _, _,
                                       "path_not_defined"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(missingPath, true));
  EXPECT_CALL(mock_cb_, sendLocalReply(Envoy::Http::Code::BadRequest,
                                       "No method in request headers.", _, _,
                                       "path_not_defined"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(missingMethod, true));
}

TEST_F(PathMatcherFilterTest, DecodeHeadersShortWildcard) {
  // Test: a request matches a operation
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/foo/foo/long"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers, true));

  EXPECT_EQ(utils::getStringFilterState(*mock_cb_.stream_info_.filter_state_,
                                        utils::kFilterStateOperation),
            "1.cloudesf_testing_cloud_goog.Long");
}

TEST_F(PathMatcherFilterTest, DecodeHeadersOverflowWildcard) {
  // Construct a request with a long path: "/aaa...aaa/long"
  std::string a_chars(9000, 'a');
  std::string path = absl::StrCat("/", a_chars, "/long");

  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", path}};

  // Filter should reject the request.
  EXPECT_CALL(
      mock_cb_.stream_info_,
      setResponseFlag(
          Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService));

  EXPECT_CALL(mock_cb_,
              sendLocalReply(Envoy::Http::Code::BadRequest,
                             "Path is too long, max allowed size is 8192.", _,
                             _, "path_not_defined"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers, true));
}

}  // namespace

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
