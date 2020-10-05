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

#include <chrono>

#include "google/api/label.pb.h"
#include "google/api/metric.pb.h"
#include "google/api/servicecontrol/v1/quota_controller.pb.h"
#include "google/api/servicecontrol/v1/service_controller.pb.h"
#include "google/protobuf/stubs/status.h"
#include "google/protobuf/timestamp.pb.h"

#include "src/api_proxy/service_control/request_info.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {

class RequestBuilder final {
 public:
  // Initializes RequestBuilder with all supported metrics and labels.
  RequestBuilder(const std::set<std::string>& logs,
                 const std::string& service_name,
                 const std::string& service_config_id);

  // Initializes RequestBuilder with specified (and supported) metrics and
  // labels.
  RequestBuilder(const std::set<std::string>& logs,
                 const std::set<std::string>& metrics,
                 const std::set<std::string>& labels,
                 const std::string& service_name,
                 const std::string& service_config_id);

  // Fills the CheckRequest protobuf from info.
  // There are some logic inside the Fill functions beside just filling
  // the fields, such as if both consumer_projecd_id and api_key present,
  // one has to set to operation.producer_project_id and the other has to
  // set to label.
  // FillCheckRequest function should copy the strings pointed by info.
  // These buffers may be freed after the FillCheckRequest call.
  ::google::protobuf::util::Status FillCheckRequest(
      const CheckRequestInfo& info,
      ::google::api::servicecontrol::v1::CheckRequest* request) const;

  ::google::protobuf::util::Status FillAllocateQuotaRequest(
      const QuotaRequestInfo& info,
      ::google::api::servicecontrol::v1::AllocateQuotaRequest* request) const;

  // Fills the CheckRequest protobuf from info.
  // FillReportRequest function should copy the strings pointed by info.
  // These buffers may be freed after the FillReportRequest call.
  ::google::protobuf::util::Status FillReportRequest(
      const ReportRequestInfo& info,
      ::google::api::servicecontrol::v1::ReportRequest* request) const;

  // Append a new consumer project Operations to the ReportRequest, if customer
  // project id from the CheckResponse is not empty
  ::google::protobuf::util::Status AppendByConsumerOperations(
      const ReportRequestInfo& info,
      ::google::api::servicecontrol::v1::ReportRequest* request,
      ::google::protobuf::Timestamp current_time) const;

  static bool IsMetricSupported(const ::google::api::MetricDescriptor& metric);
  static bool IsLabelSupported(const ::google::api::LabelDescriptor& label);
  const std::string& service_name() const { return service_name_; }
  const std::string& service_config_id() const { return service_config_id_; }

 private:
  const std::vector<std::string> logs_;
  const std::vector<const struct SupportedMetric*> metrics_;
  const std::vector<const struct SupportedLabel*> labels_;
  const std::string service_name_;
  const std::string service_config_id_;
};

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
