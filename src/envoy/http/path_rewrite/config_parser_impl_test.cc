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

#include "src/envoy/http/path_rewrite/config_parser_impl.h"

#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "source/common/protobuf/utility.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_rewrite {

class ConfigParserImplTest : public ::testing::Test {
 protected:
  void setUp(const std::string& config_str) {
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(config_str,
                                                              &proto_config_));
    obj_ = std::make_unique<ConfigParserImpl>(proto_config_);
  }

  void validateConfig(const std::string& config_str) {
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(config_str,
                                                              &proto_config_));
    Envoy::TestUtility::validate(proto_config_);
  }

  ::espv2::api::envoy::v12::http::path_rewrite::PerRouteFilterConfig
      proto_config_;
  std::unique_ptr<ConfigParserImpl> obj_;
  std::string new_path_;
};

TEST_F(ConfigParserImplTest, ValidatePathPrefixEmptyConfig) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidatePathPrefixWithRoot) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    path_prefix: "/"
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidatePathPrefixWithQuestionMark) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    path_prefix: "/foo?a=1"
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidatePathPrefixWithFragment) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    path_prefix: "/foo#a=1"
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidateConstPathEmpty) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    constant_path: {
      path: ""
    }
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidateConstPathWithQuestionMark) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    constant_path: {
      path: "/foo?a=1"
    }
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, ValidateConstPathWithFragment) {
  EXPECT_THROW_WITH_REGEX(validateConfig(R"(
    constant_path: {
      path: "/bar#abc"
    }
  )"),
                          Envoy::ProtoValidationException,
                          "Proto constraint validation failed");
}

TEST_F(ConfigParserImplTest, PathPrefixBasic) {
  setUp(R"(
  path_prefix: "/foo"
)");

  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/foo/bar");

  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/foo/bar?xyz=123");
}

TEST_F(ConfigParserImplTest, PathPrefixRemoveLastSlash) {
  setUp(R"(
  path_prefix: "/foo/"
)");

  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/foo/bar");

  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/foo/bar?xyz=123");
}

TEST_F(ConfigParserImplTest, PathPrefixBasicRoot) {
  // This is no-op, should not be used
  setUp(R"(
  path_prefix: "/"
)");

  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/bar");

  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/bar?xyz=123");
}

TEST_F(ConfigParserImplTest, ConstantPathNoUrlTemplate) {
  setUp(R"(
  constant_path: {
     path: "/foo"
  }
)");

  // /bar => /foo
  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/foo");

  // /bar?xyz=123 => /foo?xyz=123
  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/foo?xyz=123");
}

TEST_F(ConfigParserImplTest, ConstantPathUrlTemplate) {
  setUp(R"(
  constant_path: {
     path: "/foo"
     url_template: "/bar/{abc}"
  }
)");

  // A  mistmatched case
  EXPECT_FALSE(obj_->rewrite("/foo/bar", new_path_));

  // /bar/567 => /foo?abc=567
  EXPECT_TRUE(obj_->rewrite("/bar/567", new_path_));
  EXPECT_EQ(new_path_, "/foo?abc=567");

  // /bar/567?xyz=123 => /foo?xyz=123&abc=567
  EXPECT_TRUE(obj_->rewrite("/bar/567?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/foo?xyz=123&abc=567");
}

TEST_F(ConfigParserImplTest, ConstantPathNoUrlTemplateRemovedLastSlash) {
  setUp(R"(
  constant_path: {
     path: "/foo/"
  }
)");

  // /bar => /foo
  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/foo");

  // /bar?xyz=123 => /foo?xyz=123
  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/foo?xyz=123");
}

TEST_F(ConfigParserImplTest, ConstantPathNoUrlTemplateRoot) {
  setUp(R"(
  constant_path: {
     path: "/"
  }
)");

  // /bar => /
  EXPECT_TRUE(obj_->rewrite("/bar", new_path_));
  EXPECT_EQ(new_path_, "/");

  // /bar?xyz=123 => /foo?xyz=123
  EXPECT_TRUE(obj_->rewrite("/bar?xyz=123", new_path_));
  EXPECT_EQ(new_path_, "/?xyz=123");
}

}  // namespace path_rewrite
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
