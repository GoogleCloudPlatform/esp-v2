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

#include "gmock/gmock.h"
#include "google/protobuf/util/message_differencer.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

using ::google::api_proxy::envoy::http::service_control::FilterConfig;
using ::google::api::envoy::http::service_control::Requirement;
using ::google::protobuf::TextFormat;
using ::google::protobuf::util::MessageDifferencer;
using ::testing::ReturnRef;

const char kFilterConfig[] = R"(
service_name: "echo"
rules {
  patterns {
    uri_template: "/get/{foo}"
    http_method: "GET"
  }
  patterns {
    uri_template: "/post/{bar}"
    http_method: "POST"
  }
  requires {
    operation_name: "Check"
  }
})";

const char kFilterConfigMultiRule[] = R"(
service_name: "echo"
rules {
  patterns {
    uri_template: "/get/{foo}"
    http_method: "GET"
  }
  patterns {
    uri_template: "/post/{bar}"
    http_method: "POST"
  }
  requires {
    operation_name: "Check"
  }
}
rules {
  patterns {
    uri_template: "/get2/{foo2}"
    http_method: "GET"
  }
  patterns {
    uri_template: "/post2/{bar2}"
    http_method: "POST"
  }
  requires {
    operation_name: "Report"
  }
}
rules {
  patterns {
    uri_template: "/{foo2}"
    http_method: "GET"
  }
  patterns {
    uri_template: "/{bar2}"
    http_method: "POST"
  }
  requires {
    operation_name: "Echo"
  }
})";

const char kFilterConfigSameUri[] = R"(
service_name: "echo"
rules {
  patterns {
    uri_template: "/same"
    http_method: "GET"
  }
  patterns {
    uri_template: "/same"
    http_method: "POST"
  }
  requires {
    operation_name: "Check"
  }
})";

const char kFilterConfigDuplicatePattern[] = R"(
service_name: "echo"
rules {
  patterns {
    uri_template: "/same"
    http_method: "GET"
  }
  patterns {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    operation_name: "Report"
  }
})";

const char kFilterConfigNoPattern[] = R"(
service_name: "echo"
rules {
  requires {
    operation_name: "Check"
  }
})";

TEST(ConfigParserTest, TestConfigEmpty) {
  FilterConfig config;
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/get", &requirement);

  EXPECT_TRUE(requirement.operation_name() == "");
}

TEST(ConfigParserTest, TestConfig) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfig, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/get/key", &requirement);

  Requirement expected;
  const char kResult[] = R"(operation_name: "Check")";
  ASSERT_TRUE(TextFormat::ParseFromString(kResult, &expected));
  EXPECT_TRUE(MessageDifferencer::Equals(requirement, expected));
}

TEST(ConfigParserTest, TestConfigMultiRule) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigMultiRule, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("POST", "/echo", &requirement);

  Requirement expected;
  const char kResult[] = R"(operation_name: "Echo")";
  ASSERT_TRUE(TextFormat::ParseFromString(kResult, &expected));
  EXPECT_TRUE(MessageDifferencer::Equals(requirement, expected));
}

TEST(ConfigParserTest, TestConfigSamePathGet) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigSameUri, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/same", &requirement);

  Requirement expected;
  const char kResult[] = R"(operation_name: "Check")";
  ASSERT_TRUE(TextFormat::ParseFromString(kResult, &expected));
  EXPECT_TRUE(MessageDifferencer::Equals(requirement, expected));
}

TEST(ConfigParserTest, TestConfigSamePathPost) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigSameUri, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("POST", "/same", &requirement);

  Requirement expected;
  const char kResult[] = R"(operation_name: "Check")";
  ASSERT_TRUE(TextFormat::ParseFromString(kResult, &expected));
  EXPECT_TRUE(MessageDifferencer::Equals(requirement, expected));
}

TEST(ConfigParserTest, TestConfigDuplicatePattern) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigDuplicatePattern, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/same", &requirement);

  Requirement expected;
  const char kResult[] = R"(operation_name: "Report")";
  ASSERT_TRUE(TextFormat::ParseFromString(kResult, &expected));
  EXPECT_TRUE(MessageDifferencer::Equals(requirement, expected));
}

TEST(ConfigParserTest, TestConfigEmptyPattern) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigNoPattern, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/test", &requirement);
  EXPECT_TRUE(requirement.operation_name() == "");
}

TEST(ConfigParserTest, TestConfigUnmatchedPattern) {
  FilterConfig config;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfig, &config));
  auto parser = std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config));
  Requirement requirement;
  parser->FindRequirement("GET", "/test", &requirement);
  EXPECT_TRUE(requirement.operation_name() == "");
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy