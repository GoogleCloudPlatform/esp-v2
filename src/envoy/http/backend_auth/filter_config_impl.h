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

#include "api/envoy/http/backend_auth/config.pb.h"
#include "common/common/logger.h"
#include "src/envoy/http/backend_auth/config_parser.h"
#include "src/envoy/http/backend_auth/filter_config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {
using ConfigParserCreateFunc = std::function<FilterConfigParserPtr(
    const ::google::api::envoy::http::backend_auth::FilterConfig& proto_config,
    Server::Configuration::FactoryContext& context)>;
// The Envoy filter config for ESPv2 backend auth filter.
class FilterConfigImpl : public FilterConfig,
                         public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfigImpl(const ::google::api::envoy::http::backend_auth::FilterConfig&
                       proto_config,
                   const std::string& stats_prefix,
                   Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())),
        token_subscriber_factory_(context),
        config_parser_(std::make_unique<FilterConfigParserImpl>(
            proto_config_, context, token_subscriber_factory_)) {}

  const ::google::api::envoy::http::backend_auth::FilterConfig& config() const {
    return proto_config_;
  }

  FilterStats& stats() override { return stats_; }
  const FilterConfigParser& cfg_parser() const override {
    return *config_parser_;
  }

 private:
  FilterStats generateStats(const std::string& prefix, Stats::Scope& scope) {
    const std::string final_prefix = prefix + "backend_auth.";
    return {ALL_BACKEND_AUTH_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  ::google::api::envoy::http::backend_auth::FilterConfig proto_config_;
  FilterStats stats_;
  const Token::TokenSubscriberFactoryImpl token_subscriber_factory_;
  FilterConfigParserPtr config_parser_;
};

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
