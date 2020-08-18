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

#include "src/envoy/http/path_matcher/filter_config.h"
#include "common/common/empty_string.h"
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

using ::espv2::api::envoy::v8::http::path_matcher::PathMatcherRule;
using ::espv2::api::envoy::v8::http::path_matcher::PathParameterExtractionRule;
using ::espv2::api_proxy::path_matcher::VariableBinding;
using ::google::protobuf::TextFormat;
using VariableBindings = std::vector<VariableBinding>;
using FieldPath = std::vector<std::string>;

TEST(FilterConfigTest, EmptyConfig) {
  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;
  FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory);

  EXPECT_EQ(cfg.findRule("GET", "/foo"), nullptr);
}

TEST(FilterConfigTest, BasicConfig) {
  const char kFilterConfigBasic[] = R"(
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
    uri_template: "/foo/{id}"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;
  FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory);

  EXPECT_EQ(cfg.findRule("GET", "/bar")->operation(),
            "1.cloudesf_testing_cloud_goog.Bar");
  EXPECT_EQ(cfg.findRule("GET", "/foo/xyz")->operation(),
            "1.cloudesf_testing_cloud_goog.Foo");

  EXPECT_EQ(cfg.findRule("POST", "/bar"), nullptr);
  EXPECT_EQ(cfg.findRule("POST", "/foo/xyz"), nullptr);
}

TEST(FilterConfigTest, VariableBinding) {
  const char kFilterConfigBasic[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Foo"
  pattern {
    http_method: "GET"
    uri_template: "/foo/{id}"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;
  FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory);

  VariableBindings bindings;
  EXPECT_EQ(cfg.findRule("GET", "/foo/xyz", &bindings)->operation(),
            "1.cloudesf_testing_cloud_goog.Foo");
  EXPECT_EQ(bindings, VariableBindings({
                          VariableBinding{FieldPath{"id"}, "xyz"},
                      }));

  // With query parameters
  EXPECT_EQ(cfg.findRule("GET", "/foo/xyz?zone=east", &bindings)->operation(),
            "1.cloudesf_testing_cloud_goog.Foo");
  EXPECT_EQ(bindings, VariableBindings({
                          VariableBinding{FieldPath{"id"}, "xyz"},
                      }));
}

TEST(FilterConfigTest, PathParameterExtraction) {
  const char kFilterConfigBasic[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  pattern {
    http_method: "GET"
    uri_template: "/bar/{shelf_id}"
  }
  path_parameter_extraction {
    snake_to_json_segments {
      key: "shelf_id"
      value: "shelfId"
    }
    snake_to_json_segments {
      key: "foo_bar"
      value: "fooBar"
    }
  }
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Foo"
  pattern {
    http_method: "GET"
    uri_template: "/foo/{id}"
  }
  path_parameter_extraction {}
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Baz"
  pattern {
    http_method: "GET"
    uri_template: "/baz/{id}"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;
  FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory);

  const PathMatcherRule* matcher_rule = cfg.findRule("GET", "/baz/1");
  EXPECT_FALSE(matcher_rule->has_path_parameter_extraction());

  matcher_rule = cfg.findRule("GET", "/foo/2");
  ASSERT_TRUE(matcher_rule->has_path_parameter_extraction());
  PathParameterExtractionRule param_rule =
      matcher_rule->path_parameter_extraction();
  EXPECT_TRUE(param_rule.snake_to_json_segments().empty());

  matcher_rule = cfg.findRule("GET", "/bar/3");
  ASSERT_TRUE(matcher_rule->has_path_parameter_extraction());
  param_rule = matcher_rule->path_parameter_extraction();
  EXPECT_FALSE(param_rule.snake_to_json_segments().empty());
  EXPECT_EQ(param_rule.snake_to_json_segments().at("shelf_id"), "shelfId");
  EXPECT_EQ(param_rule.snake_to_json_segments().at("foo_bar"), "fooBar");
}

TEST(FilterConfigTest, DuplicatedPatterns) {
  const char kFilterConfig[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  pattern {
    http_method: "GET"
    uri_template: "/bar/{id}"
  }
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar1"
  pattern {
    http_method: "GET"
    uri_template: "/bar/{x}"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfig, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;

  EXPECT_THROW_WITH_REGEX(
      FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory),
      Envoy::ProtoValidationException, "Duplicated pattern");
}

TEST(FilterConfigTest, InvalidPattern) {
  const char kFilterConfig[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  pattern {
    http_method: "GET"
    uri_template: "/bar/{id{x}}"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfig, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;

  EXPECT_THROW_WITH_REGEX(
      FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory),
      Envoy::ProtoValidationException, "invalid pattern");
}

TEST(FilterConfigTest, NonStandardHttpMethod) {
  const char kFilterConfigBasic[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  pattern {
    http_method: "NonStandardMethod"
    uri_template: "/bar"
  }
})";

  ::espv2::api::envoy::v8::http::path_matcher::FilterConfig config_pb;
  ASSERT_TRUE(TextFormat::ParseFromString(kFilterConfigBasic, &config_pb));
  ::testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory;
  FilterConfig cfg(config_pb, Envoy::EMPTY_STRING, mock_factory);

  EXPECT_EQ(cfg.findRule("NonStandardMethod", "/bar")->operation(),
            "1.cloudesf_testing_cloud_goog.Bar");

  EXPECT_EQ(cfg.findRule("GET", "/bar"), nullptr);
  EXPECT_EQ(cfg.findRule("POST", "/bar"), nullptr);
}

}  // namespace
}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
