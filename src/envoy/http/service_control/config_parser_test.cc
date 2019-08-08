// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/mocks.h"
#include "test/test_common/utility.h"

#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::protobuf::TextFormat;

TEST(ConfigParserTest, EmptyConfig) {
  FilterConfig config;
  testing::NiceMock<MockServiceControlCallFactory> mock_factory;
  FilterConfigParser parser(config, mock_factory);

  EXPECT_FALSE(parser.FindRequirement("foo"));
}

TEST(ConfigParserTest, ValidConfig) {
  FilterConfig config;
  const char kFilterConfigBasic[] = R"(
services {
  service_name: "echo"
}
services {
  service_name: "echo111"
}
requirements {
  service_name: "echo"
  operation_name: "get_foo"
}
requirements {
  service_name: "echo111"
  operation_name: "post_bar"
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config));
  testing::NiceMock<MockServiceControlCallFactory> mock_factory;
  FilterConfigParser parser(config, mock_factory);

  EXPECT_EQ(parser.FindRequirement("get_foo")->config().operation_name(),
            "get_foo");
  EXPECT_EQ(
      parser.FindRequirement("get_foo")->service_ctx().config().service_name(),
      "echo");

  EXPECT_EQ(parser.FindRequirement("post_bar")->config().operation_name(),
            "post_bar");
  EXPECT_EQ(
      parser.FindRequirement("post_bar")->service_ctx().config().service_name(),
      "echo111");

  EXPECT_FALSE(parser.FindRequirement("non-existing-operation"));
}

TEST(ConfigParserTest, DuplicatedServiceNames) {
  FilterConfig config;
  const char kConfigWithDupliacedService[] = R"(
services {
  service_name: "dup"
}
services {
  service_name: "dup"
})";
  ASSERT_TRUE(
      TextFormat::ParseFromString(kConfigWithDupliacedService, &config));
  testing::NiceMock<MockServiceControlCallFactory> mock_factory;
  EXPECT_THROW_WITH_REGEX(FilterConfigParser parser(config, mock_factory),
                          ProtoValidationException, "Duplicated service names");
}

TEST(ConfigParserTest, DuplicatedOperationNames) {
  FilterConfig config;
  const char kConfigWithDupliacedService[] = R"(
services {
  service_name: "echo"
}
requirements {
  service_name: "echo"
  operation_name: "get_foo"
}
requirements {
  service_name: "echo"
  operation_name: "get_foo"
})";
  ASSERT_TRUE(
      TextFormat::ParseFromString(kConfigWithDupliacedService, &config));
  testing::NiceMock<MockServiceControlCallFactory> mock_factory;
  EXPECT_THROW_WITH_REGEX(FilterConfigParser parser(config, mock_factory),
                          ProtoValidationException,
                          "Duplicated operation names");
}

TEST(ConfigParserTest, InvalidServiceInRequirement) {
  FilterConfig config;
  const char kFilterInvalidService[] = R"(
requirements {
  service_name: "non-existing-service"
  operation_name: "Check"
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterInvalidService, &config));
  testing::NiceMock<MockServiceControlCallFactory> mock_factory;
  EXPECT_THROW_WITH_REGEX(FilterConfigParser parser(config, mock_factory),
                          ProtoValidationException, "Invalid service name");
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
