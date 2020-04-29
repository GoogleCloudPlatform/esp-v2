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
#include "envoy/common/pure.h"
#include "envoy/tracing/http_tracer.h"
#include "src/envoy/http/service_control/service_control_callback_func.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

class ServiceControlCall {
 public:
  virtual ~ServiceControlCall() = default;

  virtual CancelFunc callCheck(
      const ::espv2::api_proxy::service_control::CheckRequestInfo& request_info,
      Envoy::Tracing::Span& parent_span, CheckDoneFunc on_done) PURE;

  virtual void callQuota(
      const ::espv2::api_proxy::service_control::QuotaRequestInfo& request_info,
      QuotaDoneFunc on_done) PURE;

  virtual void callReport(
      const ::espv2::api_proxy::service_control::ReportRequestInfo&
          request_info) PURE;
};

using ServiceControlCallPtr = std::unique_ptr<ServiceControlCall>;

class ServiceControlCallFactory {
 public:
  virtual ~ServiceControlCallFactory() = default;

  virtual ServiceControlCallPtr create(
      const ::google::api::envoy::http::service_control::Service& config) PURE;
};

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
