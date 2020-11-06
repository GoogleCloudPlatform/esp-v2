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

#include "api/envoy/v9/http/common/base.pb.h"
#include "src/envoy/token/iam_token_info.h"
#include "src/envoy/token/imds_token_info.h"
#include "src/envoy/token/token_subscriber.h"
#include "src/envoy/token/token_subscriber_factory.h"

namespace espv2 {
namespace envoy {
namespace token {

class TokenSubscriberFactoryImpl : public TokenSubscriberFactory {
 public:
  TokenSubscriberFactoryImpl(
      Envoy::Server::Configuration::FactoryContext& context)
      : context_(context) {}

  TokenSubscriberPtr createImdsTokenSubscriber(
      const TokenType& token_type, const std::string& token_cluster,
      const std::string& token_url, std::chrono::seconds fetch_timeout,
      ::espv2::api::envoy::v9::http::common::DependencyErrorBehavior
          error_behavior,
      UpdateTokenCallback callback) const override {
    TokenInfoPtr info = std::make_unique<ImdsTokenInfo>();
    TokenSubscriberPtr subscriber = std::make_unique<TokenSubscriber>(
        context_, token_type, token_cluster, token_url, fetch_timeout,
        error_behavior, callback, std::move(info));
    subscriber->init();
    return subscriber;
  }

  TokenSubscriberPtr createIamTokenSubscriber(
      const TokenType& token_type, const std::string& token_cluster,
      const std::string& token_url, std::chrono::seconds fetch_timeout,
      ::espv2::api::envoy::v9::http::common::DependencyErrorBehavior
          error_behavior,
      UpdateTokenCallback callback,
      const ::google::protobuf::RepeatedPtrField<std::string>& delegates,
      const ::google::protobuf::RepeatedPtrField<std::string>& scopes,
      GetTokenFunc access_token_fn) const override {
    TokenInfoPtr info = std::make_unique<IamTokenInfo>(
        delegates, scopes, token_type == IdentityToken, access_token_fn);
    TokenSubscriberPtr subscriber = std::make_unique<TokenSubscriber>(
        context_, token_type, token_cluster, token_url, fetch_timeout,
        error_behavior, callback, std::move(info));
    subscriber->init();
    return subscriber;
  }

 private:
  Envoy::Server::Configuration::FactoryContext& context_;
};

}  // namespace token
}  // namespace envoy
}  // namespace espv2
