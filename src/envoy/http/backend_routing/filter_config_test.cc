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
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
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

class BackendRoutingConfigTest : public ::testing::Test {
 protected:
  void createConfig(absl::string_view filter_config) {
    google::api::envoy::http::backend_routing::FilterConfig proto_config;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(
        std::string(filter_config), &proto_config));
    ASSERT_GT(proto_config.rules_size(), 0);

    FilterConfig config(proto_config, Envoy::EMPTY_STRING,
                        mock_factory_context_);
  }

  testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context_;
};

TEST_F(BackendRoutingConfigTest, AppendAddressNoPrefixThrows) {
  ASSERT_THROW(createConfig(R"(
    rules {
      operation: "append-with-noop-prefix-operation"
      path_translation: APPEND_PATH_TO_ADDRESS
      path_prefix: ""
    }
  )"),
               Envoy::ProtoValidationException);
}

TEST_F(BackendRoutingConfigTest, ConstAddressNoPrefixThrows) {
  ASSERT_THROW(createConfig(R"(
    rules {
      operation: "const-with-invalid-prefix-operation"
      path_translation: CONSTANT_ADDRESS
      path_prefix: ""
    }
  )"),
               Envoy::ProtoValidationException);
}

TEST_F(BackendRoutingConfigTest, ConstAddressRootPrefixWorks) {
  ASSERT_NO_THROW(createConfig(R"(
    rules {
      operation: "const-with-root-prefix-operation"
      path_translation: CONSTANT_ADDRESS
      path_prefix: "/"
    }
  )"));
}

TEST_F(BackendRoutingConfigTest, PathTranslationUnspecifiedThrows) {
  ASSERT_THROW(createConfig(R"(
    rules {
      operation: "invalid-path-translation-operation"
      path_prefix: "/test"
    }
  )"),
               Envoy::ProtoValidationException);
}

TEST_F(BackendRoutingConfigTest, InvalidPathCharactersThrows) {
  ASSERT_THROW(createConfig(R"(
    rules {
      operation: "invalid-path-prefix-operation"
      path_translation: APPEND_PATH_TO_ADDRESS
      path_prefix: "/test\r\n/invalid"
    }
  )"),
               Envoy::ProtoValidationException);
}

}  // namespace

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
