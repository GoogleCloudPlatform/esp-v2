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
    case Code::OK:
      filter_stats.check_count_OK_.inc();
      return;
    case Code::CANCELLED:
      filter_stats.check_count_CANCELLED_.inc();
      return;
    case Code::UNKNOWN:
      filter_stats.check_count_UNKNOWN_.inc();
      return;
    case Code::INVALID_ARGUMENT:
      filter_stats.check_count_INVALID_ARGUMENT_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:
      filter_stats.check_count_DEADLINE_EXCEEDED_.inc();
      return;
    case Code::NOT_FOUND:
      filter_stats.check_count_NOT_FOUND_.inc();
      return;
    case Code::ALREADY_EXISTS:
      filter_stats.check_count_ALREADY_EXISTS_.inc();
      return;
    case Code::PERMISSION_DENIED:
      filter_stats.check_count_PERMISSION_DENIED_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:
      filter_stats.check_count_RESOURCE_EXHAUSTED_.inc();
      return;
    case Code::FAILED_PRECONDITION:
      filter_stats.check_count_FAILED_PRECONDITION_.inc();
      return;
    case Code::ABORTED:
      filter_stats.check_count_ABORTED_.inc();
      return;
    case Code::OUT_OF_RANGE:
      filter_stats.check_count_OUT_OF_RANGE_.inc();
      return;
    case Code::UNIMPLEMENTED:
      filter_stats.check_count_UNIMPLEMENTED_.inc();
      return;
    case Code::INTERNAL:
      filter_stats.check_count_INTERNAL_.inc();
      return;
    case Code::UNAVAILABLE:
      filter_stats.check_count_UNAVAILABLE_.inc();
      return;
    case Code::DATA_LOSS:
      filter_stats.check_count_DATA_LOSS_.inc();
      return;
    case Code::UNAUTHENTICATED:
      filter_stats.check_count_UNAUTHENTICATED_.inc();
      return;
    default:
      return;
  }
}

void ServiceControlFilterStats::collectQuotaStatus(
    ServiceControlFilterStats& filter_stats, const Code& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case Code::OK:
      filter_stats.quota_count_OK_.inc();
      return;
    case Code::CANCELLED:
      filter_stats.quota_count_CANCELLED_.inc();
      return;
    case Code::UNKNOWN:
      filter_stats.quota_count_UNKNOWN_.inc();
      return;
    case Code::INVALID_ARGUMENT:
      filter_stats.quota_count_INVALID_ARGUMENT_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:
      filter_stats.quota_count_DEADLINE_EXCEEDED_.inc();
      return;
    case Code::NOT_FOUND:
      filter_stats.quota_count_NOT_FOUND_.inc();
      return;
    case Code::ALREADY_EXISTS:
      filter_stats.quota_count_ALREADY_EXISTS_.inc();
      return;
    case Code::PERMISSION_DENIED:
      filter_stats.quota_count_PERMISSION_DENIED_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:
      filter_stats.quota_count_RESOURCE_EXHAUSTED_.inc();
      return;
    case Code::FAILED_PRECONDITION:
      filter_stats.quota_count_FAILED_PRECONDITION_.inc();
      return;
    case Code::ABORTED:
      filter_stats.quota_count_ABORTED_.inc();
      return;
    case Code::OUT_OF_RANGE:
      filter_stats.quota_count_OUT_OF_RANGE_.inc();
      return;
    case Code::UNIMPLEMENTED:
      filter_stats.quota_count_UNIMPLEMENTED_.inc();
      return;
    case Code::INTERNAL:
      filter_stats.quota_count_INTERNAL_.inc();
      return;
    case Code::UNAVAILABLE:
      filter_stats.quota_count_UNAVAILABLE_.inc();
      return;
    case Code::DATA_LOSS:
      filter_stats.quota_count_DATA_LOSS_.inc();
      return;
    case Code::UNAUTHENTICATED:
      filter_stats.quota_count_UNAUTHENTICATED_.inc();
      return;
    default:
      return;
  }
}

void ServiceControlFilterStats::collectReportStatus(
    ServiceControlFilterStats& filter_stats, const Code& code) {
  // The status error code cases must match the error codes defined by
  // https://github.com/protocolbuffers/protobuf/blob/4b4e66743503bf927cfb0f27a267ecd077250667/src/google/protobuf/stubs/status.h#L45
  switch (code) {
    case Code::OK:
      filter_stats.report_count_OK_.inc();
      return;
    case Code::CANCELLED:
      filter_stats.report_count_CANCELLED_.inc();
      return;
    case Code::UNKNOWN:
      filter_stats.report_count_UNKNOWN_.inc();
      return;
    case Code::INVALID_ARGUMENT:
      filter_stats.report_count_INVALID_ARGUMENT_.inc();
      return;
    case Code::DEADLINE_EXCEEDED:
      filter_stats.report_count_DEADLINE_EXCEEDED_.inc();
      return;
    case Code::NOT_FOUND:
      filter_stats.report_count_NOT_FOUND_.inc();
      return;
    case Code::ALREADY_EXISTS:
      filter_stats.report_count_ALREADY_EXISTS_.inc();
      return;
    case Code::PERMISSION_DENIED:
      filter_stats.report_count_PERMISSION_DENIED_.inc();
      return;
    case Code::RESOURCE_EXHAUSTED:
      filter_stats.report_count_RESOURCE_EXHAUSTED_.inc();
      return;
    case Code::FAILED_PRECONDITION:
      filter_stats.report_count_FAILED_PRECONDITION_.inc();
      return;
    case Code::ABORTED:
      filter_stats.report_count_ABORTED_.inc();
      return;
    case Code::OUT_OF_RANGE:
      filter_stats.report_count_OUT_OF_RANGE_.inc();
      return;
    case Code::UNIMPLEMENTED:
      filter_stats.report_count_UNIMPLEMENTED_.inc();
      return;
    case Code::INTERNAL:
      filter_stats.report_count_INTERNAL_.inc();
      return;
    case Code::UNAVAILABLE:
      filter_stats.report_count_UNAVAILABLE_.inc();
      return;
    case Code::DATA_LOSS:
      filter_stats.report_count_DATA_LOSS_.inc();
      return;
    case Code::UNAUTHENTICATED:
      filter_stats.report_count_UNAUTHENTICATED_.inc();
      return;
    default:
      return;
  }
}
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2