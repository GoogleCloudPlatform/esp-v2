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
#include "common/common/empty_string.h"
#include "envoy/thread_local/thread_local.h"
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/token/token_subscriber_factory_impl.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_auth {

class TokenCache : public Envoy::ThreadLocal::ThreadLocalObject {
 public:
  TokenSharedPtr token_;
};

class AudienceContext {
 public:
  AudienceContext(
      const ::google::api::envoy::http::backend_auth::BackendAuthRule&
          proto_config,
      Envoy::Server::Configuration::FactoryContext& context,
      const ::google::api::envoy::http::backend_auth::FilterConfig& config,
      const token::TokenSubscriberFactory& token_subscriber_factory,
      token::GetTokenFunc access_token_fn);
  TokenSharedPtr token() const {
    if (tls_->getTyped<TokenCache>().token_) {
      return tls_->getTyped<TokenCache>().token_;
    }
    return nullptr;
  }

 private:
  Envoy::ThreadLocal::SlotPtr tls_;
  token::TokenSubscriberPtr iam_token_sub_ptr_;
  token::TokenSubscriberPtr imds_token_sub_ptr_;
};

using AudienceContextPtr = std::unique_ptr<AudienceContext>;

class FilterConfigParserImpl
    : public FilterConfigParser,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  FilterConfigParserImpl(
      const ::google::api::envoy::http::backend_auth::FilterConfig& config,
      Envoy::Server::Configuration::FactoryContext& context,
      const token::TokenSubscriberFactory& token_subscriber_factory);

  absl::string_view getAudience(absl::string_view operation) const override {
    auto operation_it = operation_map_.find(operation);
    if (operation_it == operation_map_.end()) {
      return Envoy::EMPTY_STRING;
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
  token::TokenSubscriberPtr access_token_sub_ptr_;
  absl::flat_hash_map<std::string, std::string> operation_map_;
  absl::flat_hash_map<std::string, AudienceContextPtr> audience_map_;
};

}  // namespace backend_auth
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
