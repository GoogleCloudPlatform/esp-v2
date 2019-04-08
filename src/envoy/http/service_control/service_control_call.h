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

#pragma once

#include <functional>

#include "envoy/common/pure.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/envoy/http/service_control/check_done_func.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class ServiceControlCall {
 public:
  virtual ~ServiceControlCall() {}

  virtual void callCheck(
      const ::google::api_proxy::service_control::CheckRequestInfo& request,
      CheckDoneFunc on_done) PURE;

  virtual void callReport(
      const ::google::api_proxy::service_control::ReportRequestInfo& request)
      PURE;
};

typedef std::unique_ptr<ServiceControlCall> ServiceControlCallPtr;

class ServiceControlCallFactory {
 public:
  virtual ~ServiceControlCallFactory() {}

  virtual ServiceControlCallPtr create(
      const ::google::api::envoy::http::service_control::Service& config) PURE;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
