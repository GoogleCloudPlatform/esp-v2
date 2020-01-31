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

#pragma once
#include "gmock/gmock.h"
#include "src/envoy/utils/token_subscriber_factory.h"
namespace Envoy {
namespace Extensions {
namespace Utils {

class MockTokenSubscriberFactory : public TokenSubscriberFactory {
 public:
  MOCK_METHOD(TokenSubscriberPtr, createTokenSubscriber,
              (const std::string& token_cluster, const std::string& token_url,
               const bool json_response,
               TokenSubscriber::TokenUpdateFunc callback),
              (const));

  MOCK_METHOD(
      IamTokenSubscriberPtr, createIamTokenSubscriber,
      (IamTokenSubscriber::TokenGetFunc access_token_fn,
       const std::string& iam_service_cluster,
       const std::string& iam_service_uri,
       IamTokenSubscriber::TokenType token_type,
       const ::google::protobuf::RepeatedPtrField<std::string>& delegates,
       const ::google::protobuf::RepeatedPtrField<std::string>& scopes,
       IamTokenSubscriber::TokenUpdateFunc callback),
      (const));

  MOCK_METHOD(ServiceAccountTokenPtr, createServiceAccountTokenPtr,
              (const std::string& service_account_key,
               const std::string& audience,
               ServiceAccountToken::TokenUpdateFunc callback),
              (const));
};

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
