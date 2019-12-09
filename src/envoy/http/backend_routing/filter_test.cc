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

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "src/envoy/http/backend_routing/filter.h"
#include "src/envoy/utils/filter_state_utils.h"

using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendRouting {
namespace {

const char kFilterConfig[] = R"(
rules {
  operation: "append-operation"
  is_const_address: false
  path_prefix: ""
}
rules {
  operation: "append-with-prefix-operation"
  is_const_address: false
  path_prefix: "/test-prefix"
}
rules {
  operation: "const-operation"
  is_const_address: true
  path_prefix: "/"
}
rules {
  operation: "const-with-prefix-operation"
  is_const_address: true
  path_prefix: "/test-prefix"
}
rules {
  operation: "const-with-bad-prefix-operation"
  is_const_address: true
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
        proto_config, "test-stats", mock_factory_context_);
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
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/books/1"}};

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP
  ASSERT_EQ(headers.Path()->value().getStringView(), "/books/1");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterTest, UnknownOperationName) {
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/books/1"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "unknown-operation-name");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the filter to be a NOOP
  ASSERT_EQ(headers.Path()->value().getStringView(), "/books/1");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterTest, ConstantAddress) {
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/books/1"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(), "/");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterTest, ConstantAddressWithBadPrefix) {
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/books/1"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-with-bad-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified. Note that it is expected to be an empty
  // URL here. This is problematic in practice, since it doesn't have a '/'.
  // Config manager will ensure that this configuration is never passed to
  // Envoy.
  ASSERT_EQ(headers.Path()->value().getStringView(), "");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterTest, ConstantAddressWithPathMatcherQueryParams) {
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/books/1"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-operation");
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kQueryParams,
      "id=1");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(), "/?id=1");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

/**
 * The request URL contains multiple query parameters.
 */
class BackendRoutingFilterWithQueryParamsTest
    : public BackendRoutingFilterTest {};

TEST_F(BackendRoutingFilterWithQueryParamsTest, AppendPathToAddress) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "append-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(),
            "/books/1?view=summary&filter=deleted");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, AppendPathToAddressWithPrefix) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "append-with-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(),
            "/test-prefix/books/1?view=summary&filter=deleted");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, ConstantAddress) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(),
            "/?view=summary&filter=deleted");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest,
       ConstantAddressWithPathMatcherQueryParams) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-operation");
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kQueryParams,
      "id=1");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(),
            "/?view=summary&filter=deleted&id=1");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

TEST_F(BackendRoutingFilterWithQueryParamsTest, ConstantAddressWithPrefix) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/books/1?view=summary&filter=deleted"}};
  Utils::setStringFilterState(
      mock_decoder_callbacks_.stream_info_.filter_state_, Utils::kOperation,
      "const-with-prefix-operation");

  // Call function under test
  Envoy::Http::FilterHeadersStatus status =
      filter_->decodeHeaders(headers, false);

  // Expect the path to be modified.
  ASSERT_EQ(headers.Path()->value().getStringView(),
            "/test-prefix?view=summary&filter=deleted");
  ASSERT_EQ(status, Envoy::Http::FilterHeadersStatus::Continue);
}

}  // namespace

}  // namespace BackendRouting
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
