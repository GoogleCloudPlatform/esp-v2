// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include "src/envoy/http/service_control/config_parser.h"
#include "src/api_proxy/service_control/logs_metrics_loader.h"

#include "common/protobuf/utility.h"
#include "google/protobuf/stubs/logging.h"

using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::api::envoy::http::service_control::Service;
using ::google::api_proxy::service_control::LogsMetricsLoader;
using ::google::api_proxy::service_control::RequestBuilder;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

ServiceContext::ServiceContext(const Service& filter_service,
                               Server::Configuration::FactoryContext& context)
    : filter_service_(filter_service),
      tls_(context.threadLocal().allocateSlot()),
      token_subscriber_(
          context,
          Utils::makeClinetFactory(context, filter_service_.token_cluster()),
          *this) {
  if (filter_service.has_service_config()) {
    if (!filter_service.service_config().UnpackTo(&origin_service_)) {
      throw ProtoValidationException("Invalid service config", filter_service_);
    }

    std::set<std::string> logs, metrics, labels;
    LogsMetricsLoader::Load(origin_service_, &logs, &metrics, &labels);
    request_builder_.reset(new RequestBuilder(
        logs, metrics, labels, filter_service_.service_name(),
        filter_service_.service_config_id()));
  } else {
    request_builder_.reset(
        new RequestBuilder({"endpoints_log"}, filter_service_.service_name(),
                           filter_service_.service_config_id()));
  }

  tls_->set([this,
             &cm = context.clusterManager()](Event::Dispatcher& dispatcher)
                -> ThreadLocal::ThreadLocalObjectSharedPtr {
    return std::make_shared<ThreadLocalCache>(filter_service_, cm, dispatcher);
  });
}

FilterConfigParser::FilterConfigParser(
    const FilterConfig& config,
    Server::Configuration::FactoryContext& context) {
  for (const auto& service : config.services()) {
    service_map_[service.service_name()] =
        ServiceContextPtr(new ServiceContext(service, context));
  }

  ::google::api_proxy::path_matcher::PathMatcherBuilder<
      const RequirementContext*>
      pmb;
  for (const auto& rule : config.rules()) {
    const auto& pattern = rule.pattern();
    const auto& requirement = rule.requires();

    const auto service_it = service_map_.find(requirement.service_name());
    if (service_it == service_map_.end()) {
      throw ProtoValidationException("Invalid service name", requirement);
    }

    RequirementContextPtr require_ctx(
        new RequirementContext(requirement, *service_it->second));
    if (!pmb.Register(pattern.http_method(), pattern.uri_template(),
                      std::string(), require_ctx.get())) {
      throw ProtoValidationException("Duplicated pattern", pattern);
    }
    require_ctx_list_.push_back(std::move(require_ctx));
  }
  path_matcher_ = pmb.Build();
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
