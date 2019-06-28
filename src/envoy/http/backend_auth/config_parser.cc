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

#include "src/envoy/http/backend_auth/config_parser.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

using ::google::api::envoy::http::backend_auth::FilterConfig;

// TODO(kyuc): add unit tests for all possible backend rule configs.

namespace {

// TODO(toddbeckman): Figure out if this should be abstracted to the config
const char kDefaultIdentityUrl[]{
    "http://metadata.google.internal/computeMetadata/v1/instance/"
    "service-accounts/default/identity"};

}  // namespace

AudienceContext::AudienceContext(
    const ::google::api::envoy::http::backend_auth::BackendAuthRule&
        proto_config,
    Server::Configuration::FactoryContext& context,
    const FilterConfig& filter_config)
    : tls_(context.threadLocal().allocateSlot()),
      token_subscriber_(
          context, *this, filter_config.access_token().remote_token().cluster(),
          absl::StrCat(
              filter_config.access_token().remote_token().uri().empty()
                  ? kDefaultIdentityUrl
                  : filter_config.access_token().remote_token().uri(),
              "?format=standard&audience=", proto_config.jwt_audience()),
          /*json_response=*/false) {
  tls_->set([](Event::Dispatcher&) -> ThreadLocal::ThreadLocalObjectSharedPtr {
    return std::make_shared<TokenCache>();
  });
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
