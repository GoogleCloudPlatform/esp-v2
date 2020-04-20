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

#include "src/envoy/http/error_translator/filter.h"

#include "absl/strings/string_view.h"
#include "common/common/empty_string.h"
#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {
namespace {

using ::Envoy::Http::MockStreamEncoderFilterCallbacks;
using ::Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::TextFormat;
using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;

// TODO(nareddyt)
const char kMessageOnlyFilterConfig[] = R"()";
const char kFullDetailsFilterConfig[] = R"()";

class ErrorTranslatorFilterTest : public ::testing::Test {
 protected:
  void setUp(absl::string_view filterConfig) {
    ::google::api::envoy::http::error_translator::FilterConfig config_pb;
    ASSERT_TRUE(
        TextFormat::ParseFromString(std::string(filterConfig), &config_pb));
    config_ = std::make_shared<FilterConfig>(config_pb, Envoy::EMPTY_STRING,
                                             mock_factory_context_);

    filter_ = std::make_unique<Filter>(config_);
    filter_->setEncoderFilterCallbacks(mock_cb_);
  }

  std::unique_ptr<Filter> filter_;
  FilterConfigSharedPtr config_;
  testing::NiceMock<MockFactoryContext> mock_factory_context_;
  testing::NiceMock<MockStreamEncoderFilterCallbacks> mock_cb_;
};

// TODO(nareddyt)
TEST_F(ErrorTranslatorFilterTest, TODO) { ASSERT_EQ(1, 1); }

}  // namespace

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
