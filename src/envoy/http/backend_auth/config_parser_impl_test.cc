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
#include "src/envoy/http/backend_auth/config_parser_impl.h"

#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "source/common/common/empty_string.h"
#include "src/envoy/token/mocks.h"
#include "test/mocks/server/mocks.h"

using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;
namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

using ::espv2::api::envoy::v11::http::common::DependencyErrorBehavior;

class ConfigParserImplTest : public ::testing::Test {
 protected:
  void setUp(absl::string_view filter_config) {
    google::protobuf::TextFormat::ParseFromString(std::string(filter_config),
                                                  &proto_config_);
    config_parser_ = std::make_unique<FilterConfigParserImpl>(
        proto_config_, mock_factory_context_, mock_token_subscriber_factory_);
  }
  ::espv2::api::envoy::v11::http::backend_auth::FilterConfig proto_config_;
  testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context_;
  testing::NiceMock<token::test::MockTokenSubscriberFactory>
      mock_token_subscriber_factory_;

  std::unique_ptr<FilterConfigParser> config_parser_;
};

TEST_F(ConfigParserImplTest, GetIdTokenByImds) {
  const char filter_config[] = R"(
jwt_audience_list: ["audience-foo","audience-bar"]
imds_token {
  uri: "this-is-uri"
  cluster: "this-is-cluster"
  timeout: {
    seconds: 20
  }
}
)";
  const std::string token_foo("token-foo");
  const std::string token_bar("token-bar");

  EXPECT_CALL(mock_token_subscriber_factory_,
              createImdsTokenSubscriber(
                  token::TokenType::IdentityToken, "this-is-cluster",
                  "this-is-uri?format=standard&audience=audience-foo",
                  std::chrono::seconds(20), _, _))
      .WillOnce(Invoke([&token_foo](const token::TokenType&, const std::string&,
                                    const std::string&, std::chrono::seconds,
                                    DependencyErrorBehavior,
                                    token::UpdateTokenCallback callback)
                           -> token::TokenSubscriberPtr {
        callback(token_foo);
        return nullptr;
      }));
  EXPECT_CALL(mock_token_subscriber_factory_,
              createImdsTokenSubscriber(
                  token::TokenType::IdentityToken, "this-is-cluster",
                  "this-is-uri?format=standard&audience=audience-bar",
                  std::chrono::seconds(20), _, _))
      .WillOnce(Invoke([&token_bar](const token::TokenType&, const std::string&,
                                    const std::string&, std::chrono::seconds,
                                    DependencyErrorBehavior,
                                    token::UpdateTokenCallback callback)
                           -> token::TokenSubscriberPtr {
        callback(token_bar);
        return nullptr;
      }));

  setUp(filter_config);

  EXPECT_EQ(*config_parser_->getJwtToken("audience-foo"), "token-foo");
  EXPECT_EQ(*config_parser_->getJwtToken("audience-bar"), "token-bar");

  EXPECT_EQ(config_parser_->getJwtToken("audience-non-existent"), nullptr);
}

TEST_F(ConfigParserImplTest, GetIdTokenByIam) {
  const char filter_config[] = R"(
jwt_audience_list: ["audience-foo","audience-bar"]
iam_token {
  access_token {
    remote_token {
      uri: "this-is-imds-uri"
      cluster: "this-is-imds-cluster"
      timeout: {
        seconds: 20
      }
    }
  }
 iam_uri {
    uri: "this-is-iam-uri"
    cluster: "this-is-iam-cluster"
    timeout: {
      seconds: 4
    }
  }
}
)";
  const std::string access_token("access_token");
  const std::string id_token_foo("id-token-foo");
  const std::string id_token_bar("id-token-bar");

  EXPECT_CALL(mock_token_subscriber_factory_,
              createImdsTokenSubscriber(
                  token::TokenType::AccessToken, "this-is-imds-cluster",
                  "this-is-imds-uri", std::chrono::seconds(20), _, _))
      .WillOnce(
          Invoke([&access_token](const token::TokenType&, const std::string&,
                                 const std::string&, std::chrono::seconds,
                                 DependencyErrorBehavior,
                                 token::UpdateTokenCallback callback)
                     -> token::TokenSubscriberPtr {
            callback(access_token);
            return nullptr;
          }));

  EXPECT_CALL(mock_token_subscriber_factory_,
              createIamTokenSubscriber(_, "this-is-iam-cluster",
                                       "this-is-iam-uri?audience=audience-foo",
                                       std::chrono::seconds(4), _, _, _, _, _))
      .WillOnce(
          Invoke([&id_token_foo](
                     token::TokenType, const std::string&, const std::string&,
                     std::chrono::seconds, DependencyErrorBehavior,
                     token::UpdateTokenCallback callback,
                     const ::google::protobuf::RepeatedPtrField<std::string>&,
                     const ::google::protobuf::RepeatedPtrField<std::string>&,
                     token::GetTokenFunc access_token_fn)
                     -> token::TokenSubscriberPtr {
            EXPECT_EQ(access_token_fn(), "access_token");
            callback(id_token_foo);
            return nullptr;
          }));
  EXPECT_CALL(mock_token_subscriber_factory_,
              createIamTokenSubscriber(_, "this-is-iam-cluster",
                                       "this-is-iam-uri?audience=audience-bar",
                                       std::chrono::seconds(4), _, _, _, _, _))
      .WillOnce(
          Invoke([&id_token_bar](
                     token::TokenType, const std::string&, const std::string&,
                     std::chrono::seconds, DependencyErrorBehavior,
                     token::UpdateTokenCallback callback,
                     const ::google::protobuf::RepeatedPtrField<std::string>&,
                     const ::google::protobuf::RepeatedPtrField<std::string>&,
                     token::GetTokenFunc access_token_fn)
                     -> token::TokenSubscriberPtr {
            EXPECT_EQ(access_token_fn(), "access_token");
            callback(id_token_bar);
            return nullptr;
          }));

  setUp(filter_config);

  EXPECT_EQ(*config_parser_->getJwtToken("audience-foo"), "id-token-foo");
  EXPECT_EQ(*config_parser_->getJwtToken("audience-bar"), "id-token-bar");
}

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
