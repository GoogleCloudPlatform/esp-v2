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

#include "src/envoy/http/grpc_metadata_scrubber/filter.h"

#include "source/common/common/empty_string.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace grpc_metadata_scrubber {
namespace {

using Envoy::Http::MockStreamEncoderFilterCallbacks;
using Envoy::Server::Configuration::MockFactoryContext;

class GrpcMetadataScrubberFilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    config_ = std::make_shared<FilterConfig>(Envoy::EMPTY_STRING,
                                             mock_factory_context_);
    filter_ = std::make_unique<Filter>(config_);
    filter_->setEncoderFilterCallbacks(mock_cb_);
  }

  std::unique_ptr<Filter> filter_;
  FilterConfigSharedPtr config_;
  testing::NiceMock<MockFactoryContext> mock_factory_context_;
  testing::NiceMock<MockStreamEncoderFilterCallbacks> mock_cb_;
};

TEST_F(GrpcMetadataScrubberFilterTest, ContentLengthRemoved) {
  Envoy::Http::TestResponseHeaderMapImpl headers{
      {"content-type", "application/grpc"}, {"content-length", "100"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->encodeHeaders(headers, false));

  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.all")
                ->value(),
            1L);
  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.removed")
                ->value(),
            1L);
  // Content-Length is removed
  EXPECT_TRUE(headers.ContentLength() == nullptr);
}

TEST_F(GrpcMetadataScrubberFilterTest, WrongContentType) {
  Envoy::Http::TestResponseHeaderMapImpl headers{{"content-type", "text"},
                                                 {"content-length", "100"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->encodeHeaders(headers, false));

  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.all")
                ->value(),
            1L);
  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.removed")
                ->value(),
            0L);
  // Content-Length is not removed
  EXPECT_TRUE(headers.ContentLength() != nullptr);
}

TEST_F(GrpcMetadataScrubberFilterTest, NotContentLength) {
  Envoy::Http::TestResponseHeaderMapImpl headers{
      {"content-type", "application/grpc"}};
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->encodeHeaders(headers, false));

  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.all")
                ->value(),
            1L);
  EXPECT_EQ(Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                            "grpc_metadata_scrubber.removed")
                ->value(),
            0L);
}

}  // namespace

}  // namespace grpc_metadata_scrubber
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
