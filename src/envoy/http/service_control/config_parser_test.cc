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

TEST(ConfigParserTest, TestConfigEmpty) {
  FilterConfig config;
  FilterConfigParser parser(config);

  EXPECT_FALSE(parser.FindRequirement("GET", "/get"));
}

TEST(ConfigParserTest, TestConfig) {
  FilterConfig config;
  const char kFilterConfigBasic[] = R"(
services {
  service_name: "echo"
}
services {
  service_name: "echo111"
}
rules {
  pattern {
    uri_template: "/get/{foo}"
    http_method: "GET"
  }
  requires {
    service_name: "echo"
    operation_name: "get_foo"
  }
}
rules {
  pattern {
    uri_template: "/post/{bar}"
    http_method: "POST"
  }
  requires {
    service_name: "echo111"
    operation_name: "post_bar"
  }
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config));
  FilterConfigParser parser(config);

  EXPECT_EQ(
      parser.FindRequirement("GET", "/get/key")->config().operation_name(),
      "get_foo");
  EXPECT_EQ(parser.FindRequirement("GET", "/get/key")
                ->service_ctx()
                .config()
                .service_name(),
            "echo");

  EXPECT_EQ(
      parser.FindRequirement("POST", "/post/key")->config().operation_name(),
      "post_bar");
  EXPECT_EQ(parser.FindRequirement("POST", "/post/key")
                ->service_ctx()
                .config()
                .service_name(),
            "echo111");

  EXPECT_FALSE(parser.FindRequirement("GET", "/test"));
}

TEST(ConfigParserTest, TestConfigDuplicatePattern) {
  FilterConfig config;
  const char kFilterConfigDuplicateRule[] = R"(
services {
  service_name: "echo"
}
rules {
  pattern {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    service_name: "echo"
    operation_name: "Report1"
  }
}
rules {
  pattern {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    service_name: "echo"
    operation_name: "Report2"
  }
})";

  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigDuplicateRule, &config));
  EXPECT_THROW_WITH_REGEX(FilterConfigParser parser(config),
                          ProtoValidationException, "Duplicated pattern");
}

TEST(ConfigParserTest, TestConfigEmptyPattern) {
  FilterConfig config;
  const char kFilterInvalidService[] = R"(
rules {
  pattern {
    uri_template: "/same"
    http_method: "GET"
  }
  requires {
    service_name: "echo"
    operation_name: "Check"
  }
})";
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterInvalidService, &config));
  EXPECT_THROW_WITH_REGEX(FilterConfigParser parser(config),
                          ProtoValidationException, "Invalid service name");
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
