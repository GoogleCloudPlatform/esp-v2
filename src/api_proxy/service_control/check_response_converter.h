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

#include "absl/strings/str_cat.h"
#include "google/api/servicecontrol/v1/quota_controller.pb.h"
#include "google/api/servicecontrol/v1/service_controller.pb.h"
#include "google/protobuf/stubs/status.h"
#include "src/api_proxy/service_control/request_info.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {

class CheckResponseConverter {
 public:
  CheckResponseConverter();
  virtual ~CheckResponseConverter();

  // Converts the response status information in the CheckResponse protocol
  // buffer into util::Status and returns and returns 'check_response_info'
  // subtracted from this CheckResponse.
  // project_id is used when generating error message for project_id related
  // failures.
  static ::google::protobuf::util::Status ConvertCheckResponse(
      const ::google::api::servicecontrol::v1::CheckResponse& response,
      const std::string& service_name, CheckResponseInfo* check_response_info);

  static ::google::protobuf::util::Status ConvertAllocateQuotaResponse(
      const ::google::api::servicecontrol::v1::AllocateQuotaResponse& response,
      const std::string& service_name, QuotaResponseInfo* quota_response_info);

  static std::string CheckRcDetail(absl::string_view rc_detail);

  static std::string QuotaRcDetail(absl::string_view rc_detail);

  // CheckResponseConverter is neither copyable nor movable.
  CheckResponseConverter(const CheckResponseConverter&) = delete;
  CheckResponseConverter& operator=(const CheckResponseConverter&) = delete;
};

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
