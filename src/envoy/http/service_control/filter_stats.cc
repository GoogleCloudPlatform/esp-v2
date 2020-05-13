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

#include "google/protobuf/stubs/status.h"

#include "src/envoy/http/service_control/filter_stats.h"

using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

void ServiceControlFilterStats::collectCallStatus(CallStatusStats& stats,
                                                  const Code& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case Code::OK:
      stats.OK_.inc();
      return;
    case Code::CANCELLED:
      stats.CANCELLED_.inc();
      return;
    case Code::UNKNOWN:
      stats.UNKNOWN_.inc();
      return;
    case Code::INVALID_ARGUMENT:
      stats.INVALID_ARGUMENT_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:
      stats.DEADLINE_EXCEEDED_.inc();
      return;
    case Code::NOT_FOUND:
      stats.NOT_FOUND_.inc();
      return;
    case Code::ALREADY_EXISTS:
      stats.ALREADY_EXISTS_.inc();
      return;
    case Code::PERMISSION_DENIED:
      stats.PERMISSION_DENIED_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:
      stats.RESOURCE_EXHAUSTED_.inc();
      return;
    case Code::FAILED_PRECONDITION:
      stats.FAILED_PRECONDITION_.inc();
      return;
    case Code::ABORTED:
      stats.ABORTED_.inc();
      return;
    case Code::OUT_OF_RANGE:
      stats.OUT_OF_RANGE_.inc();
      return;
    case Code::UNIMPLEMENTED:
      stats.UNIMPLEMENTED_.inc();
      return;
    case Code::INTERNAL:
      stats.INTERNAL_.inc();
      return;
    case Code::UNAVAILABLE:
      stats.UNAVAILABLE_.inc();
      return;
    case Code::DATA_LOSS:
      stats.DATA_LOSS_.inc();
      return;
    case Code::UNAUTHENTICATED:
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