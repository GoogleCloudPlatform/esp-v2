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
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

using ::google::api_proxy::envoy::http::service_control::FilterConfig;
using ::google::protobuf::TextFormat;

TEST(ConfigParserTest, TestConfigEmpty) {
  FilterConfig config;
  FilterConfigParser parser(config);

  EXPECT_FALSE(parser.FindRequirement("GET", "/get"));
}

TEST(ConfigParserTest, TestConfig) {
  FilterConfig config;
  const char kFilterConfigBasic[] = R"(
service_name: "echo"
rules {
  pattern {
    uri_template: "/get/{foo}"
    http_method: "GET"
  }
  requires {
    operation_name: "get_foo"
  }
}
rules {
  pattern {
    uri_template: "/post/{bar}"
    http_method: "POST"
  }
  requires {
    operation_name: "post_bar"
  }
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config));
  FilterConfigParser parser(config);

  EXPECT_EQ(parser.FindRequirement("GET", "/get/key")->operation_name(),
            "get_foo");
  EXPECT_EQ(parser.FindRequirement("POST", "/post/key")->operation_name(),
            "post_bar");
  EXPECT_FALSE(parser.FindRequirement("GET", "/test"));
}

TEST(ConfigParserTest, TestConfigDuplicatePattern) {
  FilterConfig config;
  const char kFilterConfigDuplicateRule[] = R"(
service_name: "echo"
rules {
  pattern {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    operation_name: "Report1"
  }
}
rules {
  pattern {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    operation_name: "Report2"
  }
})";

  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigDuplicateRule, &config));
  FilterConfigParser parser(config);

  // Not allowed to have duplicated pattern, the second one is not added
  EXPECT_EQ(parser.FindRequirement("GET", "/same")->operation_name(),
            "Report1");
}

TEST(ConfigParserTest, TestConfigEmptyPattern) {
  FilterConfig config;
  const char kFilterConfigNoPattern[] = R"(
service_name: "echo"
rules {
  requires {
    operation_name: "Check"
  }
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigNoPattern, &config));
  FilterConfigParser parser(config);

  // Empty pattern rule is skipped.
  EXPECT_FALSE(parser.FindRequirement("GET", "/same"));
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
