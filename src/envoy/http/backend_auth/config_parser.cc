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

#include <memory>

#include "src/envoy/http/backend_auth/config_parser.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

using ::google::api::envoy::http::backend_auth::FilterConfig;
using Utils::TokenSubscriber;

// TODO(kyuc): add unit tests for all possible backend rule configs.

namespace {

// TODO(toddbeckman): Figure out if this url should be abstracted to the config.

// Suspected Envoy has listener initialization bug: if a http filter needs to
// use a cluster with DSN lookup for initialization, e.g. fetching a remote
// access token, the cluster is not ready so the whole listener is destroyed.
// ADS will repeatedly send the same listener again until the cluster is ready.
// Then the listener is marked as ready but the whole Envoy server is not marked
// as ready (worker did not start) somehow. To work around this problem, use IP
// for metadata server to fetch access token.
constexpr char kDefaultIdentityUrl[]{
    "http://169.254.169.254/computeMetadata/v1/instance/"
    "service-accounts/default/identity"};

}  // namespace

AudienceContext::AudienceContext(
    const ::google::api::envoy::http::backend_auth::BackendAuthRule&
        proto_config,
    Server::Configuration::FactoryContext& context,
    const FilterConfig& filter_config)
    : tls_(context.threadLocal().allocateSlot()) {
  tls_->set([](Event::Dispatcher&) -> ThreadLocal::ThreadLocalObjectSharedPtr {
    return std::make_shared<TokenCache>();
  });

  TokenSubscriber::TokenUpdateFunc callback = [this](const std::string& token) {
    TokenSharedPtr new_token = std::make_shared<std::string>(token);
    tls_->runOnAllThreads([this, new_token]() {
      tls_->getTyped<TokenCache>().token_ = new_token;
    });
  };

  switch (filter_config.id_token_info_case()) {
    case FilterConfig::kImdsToken: {
      const std::string& uri =
          filter_config.imds_token().imds_server_uri().uri();
      const std::string& cluster =
          filter_config.imds_token().imds_server_uri().cluster();
      const std::string real_uri = absl::StrCat(
          uri.empty() ? kDefaultIdentityUrl : uri,
          "?format=standard&audience=", proto_config.jwt_audience());

      imds_token_sub_ptr_ =
          std::make_unique<TokenSubscriber>(context, cluster, real_uri,
                                            /*json_response=*/
                                            false, callback);
    }
      return;
    default:
      return;
  }
}

FilterConfigParser::FilterConfigParser(
    const FilterConfig& config,
    Server::Configuration::FactoryContext& context) {
  for (const auto& rule : config.rules()) {
    operation_map_[rule.operation()] = rule.jwt_audience();
    auto it = audience_map_.find(rule.jwt_audience());
    if (it == audience_map_.end()) {
      audience_map_[rule.jwt_audience()] =
          AudienceContextPtr(new AudienceContext(rule, context, config));
    }
  }
}

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
