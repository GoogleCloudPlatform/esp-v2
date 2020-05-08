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

void ServiceControlFilterStats::collectCheckStatus(
    ServiceControlFilterStats& filter_stats, const Code& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case Code::OK:filter_stats.check_count_0_.inc();
      return;
    case Code::CANCELLED:filter_stats.check_count_1_.inc();
      return;
    case Code::UNKNOWN:filter_stats.check_count_2_.inc();
      return;
    case Code::INVALID_ARGUMENT:filter_stats.check_count_3_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:filter_stats.check_count_4_.inc();
      return;
    case Code::NOT_FOUND:filter_stats.check_count_5_.inc();
      return;
    case Code::ALREADY_EXISTS:filter_stats.check_count_6_.inc();
      return;
    case Code::PERMISSION_DENIED:filter_stats.check_count_7_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:filter_stats.check_count_8_.inc();
      return;
    case Code::FAILED_PRECONDITION:filter_stats.check_count_9_.inc();
      return;
    case Code::ABORTED:filter_stats.check_count_10_.inc();
      return;
    case Code::OUT_OF_RANGE:filter_stats.check_count_11_.inc();
      return;
    case Code::UNIMPLEMENTED:filter_stats.check_count_12_.inc();
      return;
    case Code::INTERNAL:filter_stats.check_count_13_.inc();
      return;
    case Code::UNAVAILABLE:filter_stats.check_count_14_.inc();
      return;
    case Code::DATA_LOSS:filter_stats.check_count_15_.inc();
      return;
    case Code::UNAUTHENTICATED:filter_stats.check_count_16_.inc();
      return;
    default:return;
  }
}
void ServiceControlFilterStats::collectReportStatus(
    ServiceControlFilterStats& filter_stats, const Code& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case Code::OK:filter_stats.report_count_0_.inc();
      return;
    case Code::CANCELLED:filter_stats.report_count_1_.inc();
      return;
    case Code::UNKNOWN:filter_stats.report_count_2_.inc();
      return;
    case Code::INVALID_ARGUMENT:filter_stats.report_count_3_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:filter_stats.report_count_4_.inc();
      return;
    case Code::NOT_FOUND:filter_stats.report_count_5_.inc();
      return;
    case Code::ALREADY_EXISTS:filter_stats.report_count_6_.inc();
      return;
    case Code::PERMISSION_DENIED:filter_stats.report_count_7_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:filter_stats.report_count_8_.inc();
      return;
    case Code::FAILED_PRECONDITION:filter_stats.report_count_9_.inc();
      return;
    case Code::ABORTED:filter_stats.report_count_10_.inc();
      return;
    case Code::OUT_OF_RANGE:filter_stats.report_count_11_.inc();
      return;
    case Code::UNIMPLEMENTED:filter_stats.report_count_12_.inc();
      return;
    case Code::INTERNAL:filter_stats.report_count_13_.inc();
      return;
    case Code::UNAVAILABLE:filter_stats.report_count_14_.inc();
      return;
    case Code::DATA_LOSS:filter_stats.report_count_15_.inc();
      return;
    case Code::UNAUTHENTICATED:filter_stats.report_count_16_.inc();
      return;
    default:return;
  }
}
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2