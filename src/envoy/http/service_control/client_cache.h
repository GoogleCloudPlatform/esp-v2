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

#pragma once

#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "envoy/event/dispatcher.h"
#include "envoy/upstream/cluster_manager.h"
#include "include/service_control_client.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/check_done_func.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The class to cache check and batch report.
class ClientCache : public Logger::Loggable<Logger::Id::filter> {
 public:
  ClientCache(
      const ::google::api::envoy::http::service_control::Service& proto_config,
      Upstream::ClusterManager& cm, Event::Dispatcher& dispatcher,
      std::function<const std::string&()> token_fn);

  void callCheck(const ::google::api::servicecontrol::v1::CheckRequest& request,
                 CheckDoneFunc on_done);

  void callReport(
      const ::google::api::servicecontrol::v1::ReportRequest& request);

 private:
  const ::google::api::envoy::http::service_control::Service& config_;
  Upstream::ClusterManager& cm_;
  Event::Dispatcher& dispatcher_;
  std::function<const std::string&()> token_fn_;
  std::unique_ptr<::google::service_control_client::ServiceControlClient>
      client_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
