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
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {
namespace {

using ::espv2::api::envoy::v9::http::path_matcher::PathMatcherRule;
using ::espv2::api_proxy::path_matcher::VariableBinding;
using ::google::protobuf::TextFormat;
using VariableBindings = std::vector<VariableBinding>;
using FieldPath = std::vector<std::string>;

TEST(FilterConfigTest, EmptyConfig) {
  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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
  extract_path_parameters: true
  pattern {
    http_method: "GET"
    uri_template: "/foo/{id}"
  }
})";

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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
  extract_path_parameters: true
  pattern {
    http_method: "GET"
    uri_template: "/foo/{id}"
  }
})";

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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

TEST(FilterConfigTest, DuplicatedPatterns) {
  const char kFilterConfig[] = R"(
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar"
  extract_path_parameters: true
  pattern {
    http_method: "GET"
    uri_template: "/bar/{id}"
  }
}
rules {
  operation: "1.cloudesf_testing_cloud_goog.Bar1"
  extract_path_parameters: true
  pattern {
    http_method: "GET"
    uri_template: "/bar/{x}"
  }
})";

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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
  extract_path_parameters: true
  pattern {
    http_method: "GET"
    uri_template: "/bar/{id{x}}"
  }
})";

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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

  ::espv2::api::envoy::v9::http::path_matcher::FilterConfig config_pb;
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
