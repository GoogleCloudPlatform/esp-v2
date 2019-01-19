// Copyright 2019 Google Cloud Platform Proxy Authors
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
#ifndef ENVOY_BACKEND_AUTH_RULE_PARSER_H
#define ENVOY_BACKEND_AUTH_RULE_PARSER_H

#include "api/envoy/http/backend_auth/config.pb.h"
#include "envoy/thread_local/thread_local.h"
#include "src/envoy/utils/token_subscriber.h"

#include <list>
#include <unordered_map>

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

// Use shared_ptr to do atomic token update.
typedef std::shared_ptr<std::string> TokenSharedPtr;
class TokenCache : public ThreadLocal::ThreadLocalObject {
 public:
  TokenSharedPtr token_;
};

class AudienceContext : public Utils::TokenSubscriber::Callback {
 public:
  AudienceContext(
      const ::google::api::envoy::http::backend_auth::BackendAuthRule& proto_config,
      Server::Configuration::FactoryContext& context)
      : tls_(context.threadLocal().allocateSlot()),
        token_subscriber_(
            context, Utils::makeClientFactory(context,
            proto_config.token_cluster()), *this, &proto_config.jwt_audience()) {
    tls_->set(
        [](Event::Dispatcher&) -> ThreadLocal::ThreadLocalObjectSharedPtr {
          return std::make_shared<TokenCache>();
        });
  }

  // TokenSubscriber::Callback function
  void onTokenUpdate(const std::string& token) override {
    TokenSharedPtr new_token = std::make_shared<std::string>(token);
    tls_->runOnAllThreads([this, new_token]() {
      tls_->getTyped<TokenCache>().token_ = new_token;
    });
  }

  TokenSharedPtr token() const {
    if (tls_->getTyped<TokenCache>().token_) {
        return tls_->getTyped<TokenCache>().token_;
    }
    return nullptr;
  }

 private:
  ThreadLocal::SlotPtr tls_;
  Utils::TokenSubscriber token_subscriber_;
};

typedef std::unique_ptr<AudienceContext> AudienceContextPtr;

class FilterConfigParser {
 public:
  FilterConfigParser(
      const ::google::api::envoy::http::backend_auth::FilterConfig& config,
      Server::Configuration::FactoryContext& context);

  const TokenSharedPtr getJwtToken(const std::string& operation) const {
    auto operation_it = operation_map_.find(operation);
    if (operation_it == operation_map_.end()) {
      return nullptr;
    }
    auto audience_it = audience_map_.find(operation_it->second);
    if (audience_it == audience_map_.end()) {
      return nullptr;
    }
    return audience_it->second->token();
  }

 private:
  std::unordered_map<std::string, std::string> operation_map_;
  std::unordered_map<std::string, AudienceContextPtr> audience_map_;
};

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

#endif  // ENVOY_BACKEND_AUTH_RULE_PARSER_H
