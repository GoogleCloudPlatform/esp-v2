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

#include "api/envoy/http/common/base.pb.h"
#include "envoy/common/pure.h"
#include "envoy/tracing/http_tracer.h"
#include "envoy/upstream/cluster_manager.h"
#include "google/protobuf/stubs/status.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class HttpCall {
 public:
  using DoneFunc =
      std::function<void(const ::google::protobuf::util::Status& status,
                         const std::string& response_body)>;

  virtual ~HttpCall() {}
  /*
   * Cancel any in-flight request.
   */
  virtual void cancel() PURE;

  virtual void call() PURE;
  /*
   * Factory method for creating a HttpCall.
   * @param cm the cluster manager to use during Token retrieval
   * @return a HttpCall instance
   */
  static HttpCall* create(
      Upstream::ClusterManager& cm,
      const ::google::api::envoy::http::common::HttpUri& uri,
      const std::string& suffix_url,
      std::function<const std::string&()> token_fn,
      const Protobuf::Message& body, uint32_t timeout_ms, uint32_t retries,
      Envoy::Tracing::Span& parent_span, Envoy::TimeSource& time_source,
      const std::string& trace_operation_name, HttpCall::DoneFunc on_done);
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
