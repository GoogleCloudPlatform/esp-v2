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

#pragma once

#include "src/envoy/token/iam_token_info.h"
#include "src/envoy/token/imds_token_info.h"
#include "src/envoy/token/token_subscriber.h"
#include "src/envoy/token/token_subscriber_factory.h"

namespace Envoy {
namespace Extensions {
namespace Token {

class TokenSubscriberFactoryImpl : public TokenSubscriberFactory {
 public:
  TokenSubscriberFactoryImpl(Server::Configuration::FactoryContext& context)
      : context_(context) {}

  TokenSubscriberPtr createImdsTokenSubscriber(
      const TokenType& token_type, const std::string& token_cluster,
      const std::string& token_url,
      UpdateTokenCallback callback) const override {
    TokenInfoPtr info = std::make_unique<ImdsTokenInfo>();
    TokenSubscriberPtr subscriber =
        std::make_unique<TokenSubscriber>(context_, token_type, token_cluster,
                                          token_url, callback, std::move(info));
    subscriber->init();
    return subscriber;
  }

  TokenSubscriberPtr createIamTokenSubscriber(
      const TokenType& token_type, const std::string& token_cluster,
      const std::string& token_url, UpdateTokenCallback callback,
      const ::google::protobuf::RepeatedPtrField<std::string>& delegates,
      const ::google::protobuf::RepeatedPtrField<std::string>& scopes,
      GetTokenFunc access_token_fn) const override {
    TokenInfoPtr info = std::make_unique<IamTokenInfo>(
        delegates, scopes, token_type == IdentityToken ? true : false,
        access_token_fn);
    TokenSubscriberPtr subscriber =
        std::make_unique<TokenSubscriber>(context_, token_type, token_cluster,
                                          token_url, callback, std::move(info));
    subscriber->init();
    return subscriber;
  }

  ServiceAccountTokenPtr createServiceAccountTokenGenerator(
      const std::string& service_account_key, const std::string& audience,
      ServiceAccountTokenGenerator::TokenUpdateFunc callback) const override {
    return std::make_unique<ServiceAccountTokenGenerator>(
        context_, service_account_key, audience, callback);
  };

 private:
  Envoy::Server::Configuration::FactoryContext& context_;
};

}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy
