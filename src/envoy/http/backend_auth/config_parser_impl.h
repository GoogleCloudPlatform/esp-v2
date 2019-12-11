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
#include <list>
#include <unordered_map>

#include "absl/container/flat_hash_map.h"
#include "absl/strings/str_cat.h"
#include "api/envoy/http/backend_auth/config.pb.h"
#include "envoy/thread_local/thread_local.h"
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/utils/iam_token_subscriber.h"
#include "src/envoy/utils/token_subscriber.h"
#include "src/envoy/utils/token_subscriber_factory_impl.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

class TokenCache : public ThreadLocal::ThreadLocalObject {
 public:
  TokenSharedPtr token_;
};

class AudienceContext {
 public:
  AudienceContext(
      const ::google::api::envoy::http::backend_auth::BackendAuthRule&
          proto_config,
      Server::Configuration::FactoryContext& context,
      const ::google::api::envoy::http::backend_auth::FilterConfig& config,
      const Utils::TokenSubscriberFactory& token_subscriber_factory,
      Utils::IamTokenSubscriber::TokenGetFunc access_token_fn);
  TokenSharedPtr token() const {
    if (tls_->getTyped<TokenCache>().token_) {
      return tls_->getTyped<TokenCache>().token_;
    }
    return nullptr;
  }

 private:
  ThreadLocal::SlotPtr tls_;
  Utils::IamTokenSubscriberPtr iam_token_sub_ptr_;
  Utils::TokenSubscriberPtr imds_token_sub_ptr_;
};

typedef std::unique_ptr<AudienceContext> AudienceContextPtr;

class FilterConfigParserImpl
    : public FilterConfigParser,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  FilterConfigParserImpl(
      const ::google::api::envoy::http::backend_auth::FilterConfig& config,
      Server::Configuration::FactoryContext& context,
      const Utils::TokenSubscriberFactory& token_subscriber_factory);

  absl::string_view getAudience(absl::string_view operation) const override {
    static const std::string empty = "";
    auto operation_it = operation_map_.find(operation);
    if (operation_it == operation_map_.end()) {
      return empty;
    }
    return operation_it->second;
  }

  const TokenSharedPtr getJwtToken(absl::string_view audience) const override {
    auto audience_it = audience_map_.find(audience);
    if (audience_it == audience_map_.end()) {
      return nullptr;
    }
    return audience_it->second->token();
  }

 private:
  //  access_token_ is required for authentication during fetching id_token from
  //  IAM server.
  std::string access_token_;
  Utils::TokenSubscriberPtr access_token_sub_ptr_;
  absl::flat_hash_map<std::string, std::string> operation_map_;
  absl::flat_hash_map<std::string, AudienceContextPtr> audience_map_;
};

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
