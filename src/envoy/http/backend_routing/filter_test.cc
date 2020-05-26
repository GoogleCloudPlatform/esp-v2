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

#include "src/envoy/http/backend_routing/filter.h"

#include "common/common/empty_string.h"
#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {
namespace {

const char kFilterConfig[] = R"(
rules {
  operation: "append-operation"
  path_translation: APPEND_PATH_TO_ADDRESS
  path_prefix: ""
}
rules {
  operation: "append-with-prefix-operation"
  path_translation: APPEND_PATH_TO_ADDRESS
  path_prefix: "/test-prefix"
}
rules {
  operation: "const-operation"
  path_translation: CONSTANT_ADDRESS
  path_prefix: "/"
}
rules {
  operation: "const-with-prefix-operation"
  path_translation: CONSTANT_ADDRESS
  path_prefix: "/test-prefix"
}
rules {
  operation: "const-with-bad-prefix-operation"
  path_translation: CONSTANT_ADDRESS
  path_prefix: ""
}
)";

/**
 * Base class for testing the Backend Routing filter. Makes a simple request
 * with no query parameters in the request URL.
 */
class BackendRoutingFilterTest : public ::testing::Test {
 protected:
  BackendRoutingFilterTest() = default;

  void SetUp() override { setUp(kFilterConfig); }

  void setUp(absl::string_view filter_config) {
    google::api::envoy::http::backend_routing::FilterConfig proto_config;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(
        std::string(filter_config), &proto_config));
    ASSERT_GT(proto_config.rules_size(), 0);

    FilterConfigSharedPtr config = std::make_shared<FilterConfig>(
        proto_config, Envoy::EMPTY_STRING, mock_factory_context_);
    filter_ = std::make_unique<Filter>(config);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);
  }

  std::unique_ptr<Filter> filter_;
  testing::NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks_;
  testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context_;
};

TEST_F(BackendRoutingFilterTest, NoOperationName) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP and reject the request.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/books/1");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                      "backend_routing.denied");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterTest, UnknownOperationName) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "unknown-operation-name");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP and reject the request.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/books/1");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                      "backend_routing.denied");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterTest, NoPathHeader) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP and reject the request.
  EXPECT_EQ(headers.Path(), nullptr);
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::StopIteration);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(mock_factory_context_.scope_,
                                      "backend_routing.denied");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterTest, ConstantAddress) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterTest, ConstantAddressWithBadPrefix) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-with-bad-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified. Note that it is expected to be an empty
  // URL here. This is problematic in practice, since it doesn't have a '/'.
  // Config manager will ensure that this configuration is never passed to
  // Envoy.
  EXPECT_EQ(headers.Path()->value().getStringView(), Envoy::EMPTY_STRING);
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterTest, ConstantAddressWithPathMatcherQueryParams) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "GET"},
                                                {":path", "/books/1"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-operation");
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kQueryParams,
      "id=1");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(), "/?id=1");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

/**
 * The request URL contains multiple query parameters.
 */
class BackendRoutingFilterWithQueryParamsTest
    : public BackendRoutingFilterTest {};

TEST_F(BackendRoutingFilterWithQueryParamsTest, AppendPathToAddress) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "append-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(),
            "/books/1?view=summary&filter=deleted");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.append_path_to_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, AppendPathToAddressWithPrefix) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "append-with-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(),
            "/test-prefix/books/1?view=summary&filter=deleted");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.append_path_to_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, ConstantAddress) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(),
            "/?view=summary&filter=deleted");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest,
       ConstantAddressWithPathMatcherQueryParams) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-operation");
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kQueryParams,
      "id=1");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(),
            "/?view=summary&filter=deleted&id=1");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, ConstantAddressWithPrefix) {
  Envoy::Http::TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  utils::setStringFilterState(
      *mock_decoder_callbacks_.stream_info_.filter_state_, utils::kOperation,
      "const-with-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  EXPECT_EQ(headers.Path()->value().getStringView(),
            "/test-prefix?view=summary&filter=deleted");
  EXPECT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);

  // Stats.
  const Envoy::Stats::CounterSharedPtr counter =
      Envoy::TestUtility::findCounter(
          mock_factory_context_.scope_,
          "backend_routing.constant_address_request");
  ASSERT_NE(counter, nullptr);
  EXPECT_EQ(counter->value(), 1);
}

}  // namespace

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
