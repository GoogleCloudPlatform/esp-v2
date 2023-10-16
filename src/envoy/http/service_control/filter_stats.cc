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

#include "src/envoy/http/service_control/filter_stats.h"

using ::absl::Status;
using ::absl::StatusCode;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

void ServiceControlFilterStats::collectCallStatus(CallStatusStats& stats,
                                                  const StatusCode& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case StatusCode::kOk:
      stats.OK_.inc();
      return;
    case StatusCode::kCancelled:
      stats.CANCELLED_.inc();
      return;
    case StatusCode::kUnknown:
      stats.UNKNOWN_.inc();
      return;
    case StatusCode::kInvalidArgument:
      stats.INVALID_ARGUMENT_.inc();
      return;
    case StatusCode::kDeadlineExceeded:
      stats.DEADLINE_EXCEEDED_.inc();
      return;
    case StatusCode::kNotFound:
      stats.NOT_FOUND_.inc();
      return;
    case StatusCode::kAlreadyExists:
      stats.ALREADY_EXISTS_.inc();
      return;
    case StatusCode::kPermissionDenied:
      stats.PERMISSION_DENIED_.inc();
      return;
    case StatusCode::kResourceExhausted:
      stats.RESOURCE_EXHAUSTED_.inc();
      return;
    case StatusCode::kFailedPrecondition:
      stats.FAILED_PRECONDITION_.inc();
      return;
    case StatusCode::kAborted:
      stats.ABORTED_.inc();
      return;
    case StatusCode::kOutOfRange:
      stats.OUT_OF_RANGE_.inc();
      return;
    case StatusCode::kUnimplemented:
      stats.UNIMPLEMENTED_.inc();
      return;
    case StatusCode::kInternal:
      stats.INTERNAL_.inc();
      return;
    case StatusCode::kUnavailable:
      stats.UNAVAILABLE_.inc();
      return;
    case StatusCode::kDataLoss:
      stats.DATA_LOSS_.inc();
      return;
    case StatusCode::kUnauthenticated:
      stats.UNAUTHENTICATED_.inc();
      return;
    default:
      return;
  }
}

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
