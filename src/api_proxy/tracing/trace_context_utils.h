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

#include <string>
#include "absl/strings/string_view.h"
#include "absl/types/optional.h"

namespace espv2 {
namespace api_proxy {
namespace tracing {

class TraceContextUtils {
public:
    // Translates an x-cloud-trace-context header string to a W3C traceparent string.
    static absl::optional<std::string> XCloudTraceContextToTraceParent(
        absl::string_view x_cloud_trace_context);

    // Translates a W3C traceparent header string to an x-cloud-trace-context string.
    static absl::optional<std::string> TraceParentToXCloudTraceContext(
        absl::string_view traceparent);

    // Translates a base64-encoded grpc-trace-bin header string to a W3C traceparent string.
    static absl::optional<std::string> GrpcTraceBinToTraceParent(
        absl::string_view grpc_trace_bin);

    // Translates a W3C traceparent header string to a base64-encoded grpc-trace-bin string.
    static absl::optional<std::string> TraceParentToGrpcTraceBin(
        absl::string_view traceparent);
};

} // namespace tracing
} // namespace api_proxy
} // namespace espv2
