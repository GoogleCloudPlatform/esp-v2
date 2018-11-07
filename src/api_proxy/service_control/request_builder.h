
#pragma once

#include "google/api/label.pb.h"
#include "google/api/metric.pb.h"
#include "google/api/servicecontrol/v1/quota_controller.pb.h"
#include "google/api/servicecontrol/v1/service_controller.pb.h"
#include "google/protobuf/stubs/status.h"
#include "google/protobuf/timestamp.pb.h"

#include "src/api_proxy/service_control/request_info.h"

namespace google {
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
      ::google::api::servicecontrol::v1::CheckRequest* request);

  ::google::protobuf::util::Status FillAllocateQuotaRequest(
      const QuotaRequestInfo& info,
      ::google::api::servicecontrol::v1::AllocateQuotaRequest* request);

  // Fills the CheckRequest protobuf from info.
  // FillReportRequest function should copy the strings pointed by info.
  // These buffers may be freed after the FillReportRequest call.
  ::google::protobuf::util::Status FillReportRequest(
      const ReportRequestInfo& info,
      ::google::api::servicecontrol::v1::ReportRequest* request);

  // Append a new consumer project Operations to the ReportRequest, if customer
  // project id from the CheckResponse is not empty
  ::google::protobuf::util::Status AppendByConsumerOperations(
      const ReportRequestInfo& info,
      ::google::api::servicecontrol::v1::ReportRequest* request,
      ::google::protobuf::Timestamp current_time);

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
      const std::string& service_name);

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
}  // namespace google
