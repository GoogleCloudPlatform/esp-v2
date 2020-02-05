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

#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "envoy/server/filter_config.h"
#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/filter_stats.h"
#include "src/envoy/http/service_control/handler_impl.h"
#include "src/envoy/http/service_control/service_control_call_impl.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The Envoy filter config for ESPv2 service control client.
class ServiceControlFilterConfig : public Logger::Loggable<Logger::Id::filter>,
                                   public ServiceControlFilterStatBase {
 public:
  ServiceControlFilterConfig(
      const ::google::api::envoy::http::service_control::FilterConfig&
          proto_config,
      const std::string& stats_prefix,
      Server::Configuration::FactoryContext& context)
      : ServiceControlFilterStatBase(stats_prefix, context.scope()),
        proto_config_(
            std::make_shared<
                ::google::api::envoy::http::service_control::FilterConfig>(
                proto_config)),
        call_factory_(proto_config_, context),
        config_parser_(*proto_config_, call_factory_),
        handler_factory_(context.random(), config_parser_) {}

  const ServiceControlHandlerFactory& handler_factory() const {
    return handler_factory_;
  }

 private:
  FilterConfigProtoSharedPtr proto_config_;
  ServiceControlCallFactoryImpl call_factory_;
  FilterConfigParser config_parser_;
  ServiceControlHandlerFactoryImpl handler_factory_;
};

typedef std::shared_ptr<ServiceControlFilterConfig> FilterConfigSharedPtr;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
