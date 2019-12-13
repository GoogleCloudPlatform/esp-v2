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
#include "src/envoy/utils/mocks.h"
#include "test/mocks/server/mocks.h"

using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;
namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

class ConfigParserImplTest : public ::testing::Test {
 protected:
  void setUp(absl::string_view filter_config) {
    google::protobuf::TextFormat::ParseFromString(std::string(filter_config),
                                                  &proto_config_);
    config_parser_ = std::make_unique<FilterConfigParserImpl>(
        proto_config_, mock_factory_context_, mock_token_subscriber_factory_);
  }
  google::api::envoy::http::backend_auth::FilterConfig proto_config_;
  testing::NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context_;
  testing::NiceMock<Utils::MockTokenSubscriberFactory>
      mock_token_subscriber_factory_;

  std::unique_ptr<FilterConfigParser> config_parser_;
};

TEST_F(ConfigParserImplTest, IamIdTokenWithServiceAccountAsAccessToken) {
  const char filter_config[] = R"(
iam_token {
  access_token {
    service_account_secret{}
  }
}
rules {
  operation: "append-with-audience"
  jwt_audience: "this-is-audience"
}
)";

  EXPECT_CALL(mock_token_subscriber_factory_, createTokenSubscriber).Times(0);
  EXPECT_CALL(mock_token_subscriber_factory_, createIamTokenSubscriber)
      .Times(0);
  setUp(filter_config);
}

TEST_F(ConfigParserImplTest, GetIdTokenByImds) {
  const char filter_config[] = R"(
imds_token {
  imds_server_uri {
      uri: "this-is-uri"
      cluster: "this-is-cluster"
  }
}
rules {
  operation: "operation-foo"
  jwt_audience: "audience-foo"
}
rules {
  operation: "operation-bar"
  jwt_audience: "audience-bar"
}
)";
  const std::string token_foo("token-foo");

  const std::string token_bar("token-bar");

  EXPECT_CALL(
      mock_token_subscriber_factory_,
      createTokenSubscriber("this-is-cluster",
                            "this-is-uri?format=standard&audience=audience-foo",
                            false, _))
      .WillOnce(Invoke(
          [&token_foo](const std::string&, const std::string&, const bool,
                       Utils::TokenSubscriber::TokenUpdateFunc callback)
              -> Utils::TokenSubscriberPtr {
            callback(token_foo);
            return nullptr;
          }));
  EXPECT_CALL(
      mock_token_subscriber_factory_,
      createTokenSubscriber("this-is-cluster",
                            "this-is-uri?format=standard&audience=audience-bar",
                            false, _))
      .WillOnce(Invoke(
          [&token_bar](const std::string&, const std::string&, const bool,
                       Utils::TokenSubscriber::TokenUpdateFunc callback)
              -> Utils::TokenSubscriberPtr {
            callback(token_bar);
            return nullptr;
          }));

  setUp(filter_config);

  EXPECT_EQ(config_parser_->getAudience("operation-foo"), "audience-foo");
  EXPECT_EQ(config_parser_->getAudience("operation-bar"), "audience-bar");

  EXPECT_EQ(*config_parser_->getJwtToken("audience-foo"), "token-foo");
  EXPECT_EQ(*config_parser_->getJwtToken("audience-bar"), "token-bar");
}

TEST_F(ConfigParserImplTest, GetIdTokenByIam) {
  const char filter_config[] = R"(
iam_token {
  access_token {
    remote_token {
      uri: "this-is-imds-uri"
      cluster: "this-is-imds-cluster"
    }
  }
 iam_uri {
      uri: "this-is-iam-uri"
      cluster: "this-is-iam-cluster"
  }
}
rules {
  operation: "operation-foo"
  jwt_audience: "audience-foo"
}
rules {
  operation: "operation-bar"
  jwt_audience: "audience-bar"
}
)";
  const std::string access_token("access_token");
  const std::string id_token_foo("id-token-foo");
  const std::string id_token_bar("id-token-bar");

  EXPECT_CALL(mock_token_subscriber_factory_,
              createTokenSubscriber("this-is-imds-cluster", "this-is-imds-uri",
                                    true, _))
      .WillOnce(Invoke(
          [&access_token](const std::string&, const std::string&, const bool,
                          Utils::TokenSubscriber::TokenUpdateFunc callback)
              -> Utils::TokenSubscriberPtr {
            callback(access_token);
            return nullptr;
          }));

  EXPECT_CALL(
      mock_token_subscriber_factory_,
      createIamTokenSubscriber(_, "this-is-iam-cluster",
                               "this-is-iam-uri?audience=audience-foo", _))
      .WillOnce(
          Invoke([&id_token_foo](
                     Utils::IamTokenSubscriber::TokenGetFunc access_token_fn,
                     const std::string&, const std::string&,
                     Utils::IamTokenSubscriber::TokenUpdateFunc callback)
                     -> Utils::IamTokenSubscriberPtr {
            EXPECT_EQ(access_token_fn(), "access_token");
            callback(id_token_foo);
            return nullptr;
          }));
  EXPECT_CALL(
      mock_token_subscriber_factory_,
      createIamTokenSubscriber(_, "this-is-iam-cluster",
                               "this-is-iam-uri?audience=audience-bar", _))
      .WillOnce(
          Invoke([&id_token_bar](
                     Utils::IamTokenSubscriber::TokenGetFunc access_token_fn,
                     const std::string&, const std::string&,
                     Utils::IamTokenSubscriber::TokenUpdateFunc callback)
                     -> Utils::IamTokenSubscriberPtr {
            EXPECT_EQ(access_token_fn(), "access_token");
            callback(id_token_bar);
            return nullptr;
          }));

  setUp(filter_config);

  EXPECT_EQ(config_parser_->getAudience("operation-foo"), "audience-foo");
  EXPECT_EQ(config_parser_->getAudience("operation-bar"), "audience-bar");

  EXPECT_EQ(*config_parser_->getJwtToken("audience-foo"), "id-token-foo");
  EXPECT_EQ(*config_parser_->getJwtToken("audience-bar"), "id-token-bar");
}

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
