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

#include "src/api_proxy/service_control/check_response_converter.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {
namespace {}

using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckError_Code;
using ::google::api::servicecontrol::v1::CheckError_Code_Name;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::
    CheckResponse_ConsumerInfo_ConsumerType;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

Status CheckResponseConverter::ConvertCheckResponse(
    const CheckResponse& check_response, const std::string& service_name,
    CheckResponseInfo* check_response_info) {
  if (check_response.check_info().consumer_info().project_number() > 0) {
    // Store project id to check_response_info
    check_response_info->consumer_project_number = std::to_string(
        check_response.check_info().consumer_info().project_number());
  }

  if (check_response.check_info().consumer_info().consumer_number() > 0) {
    check_response_info->consumer_number = std::to_string(
        check_response.check_info().consumer_info().consumer_number());
  }

  if (check_response.check_info().consumer_info().type() !=
      CheckResponse_ConsumerInfo_ConsumerType::
          CheckResponse_ConsumerInfo_ConsumerType_CONSUMER_TYPE_UNSPECIFIED) {
    check_response_info->consumer_type =
        CheckResponse_ConsumerInfo_ConsumerType_Name(
            check_response.check_info().consumer_info().type());
  }

  if (check_response.check_errors().empty()) {
    return Status::OK;
  }

  // TODO: aggregate status responses for all errors (including error.detail)
  // TODO: report a detailed status to the producer project, but hide it from
  // consumer
  // TODO: unless they are the same entity
  const CheckError& error = check_response.check_errors(0);

  check_response_info->error = {CheckError_Code_Name(error.code()),
                                /*is_error_from_http_call=*/false,
                                ScResponseErrorType::ERROR_TYPE_UNSPECIFIED};

  ScResponseError& check_error = check_response_info->error;
  switch (error.code()) {
    case CheckError::NOT_FOUND:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::INVALID_ARGUMENT,
                    "Client project not found. Please pass a valid project.");
    case CheckError::RESOURCE_EXHAUSTED:
      check_error.type = ScResponseErrorType::CONSUMER_QUOTA;
      return Status(Code::RESOURCE_EXHAUSTED, "Quota check failed.");
    case CheckError::API_TARGET_BLOCKED:
      check_error.type = ScResponseErrorType::CONSUMER_BLOCKED;
      return Status(Code::PERMISSION_DENIED,
                    " The API targeted by this request is invalid for the "
                    "given API key.");
    case CheckError::API_KEY_NOT_FOUND:
      check_error.type = ScResponseErrorType::API_KEY_INVALID;
      return Status(Code::INVALID_ARGUMENT,
                    "API key not found. Please pass a valid API key.");
    case CheckError::API_KEY_EXPIRED:
      check_error.type = ScResponseErrorType::API_KEY_INVALID;
      return Status(Code::INVALID_ARGUMENT,
                    "API key expired. Please renew the API key.");
    case CheckError::API_KEY_INVALID:
      check_error.type = ScResponseErrorType::API_KEY_INVALID;
      return Status(Code::INVALID_ARGUMENT,
                    "API key not valid. Please pass a valid API key.");
    case CheckError::SERVICE_NOT_ACTIVATED:
      check_error.type = ScResponseErrorType::SERVICE_NOT_ACTIVATED;
      return Status(Code::PERMISSION_DENIED,
                    absl::StrCat("API ", service_name,
                                 " is not enabled for the project."));
    case CheckError::PERMISSION_DENIED:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED, "Permission denied.");
    case CheckError::IP_ADDRESS_BLOCKED:
      check_error.type = ScResponseErrorType::CONSUMER_BLOCKED;
      return Status(Code::PERMISSION_DENIED, "IP address blocked.");
    case CheckError::REFERER_BLOCKED:
      check_error.type = ScResponseErrorType::CONSUMER_BLOCKED;
      return Status(Code::PERMISSION_DENIED, "Referer blocked.");
    case CheckError::CLIENT_APP_BLOCKED:
      check_error.type = ScResponseErrorType::CONSUMER_BLOCKED;
      return Status(Code::PERMISSION_DENIED, "Client application blocked.");
    case CheckError::PROJECT_DELETED:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED, "Project has been deleted.");
    case CheckError::PROJECT_INVALID:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::INVALID_ARGUMENT,
                    "Client project not valid. Please pass a valid project.");
    case CheckError::BILLING_DISABLED:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED,
                    absl::StrCat("API ", service_name,
                                 " has billing disabled. Please enable it."));
    case CheckError::INVALID_CREDENTIAL:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED,
                    "The credential in the request can not be verified.");
    case CheckError::CONSUMER_INVALID:
      check_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED,
                    "The consumer from the API key does not represent"
                    " a valid consumer folder or organization.");

    case CheckError::NAMESPACE_LOOKUP_UNAVAILABLE:
    case CheckError::SERVICE_STATUS_UNAVAILABLE:
    case CheckError::BILLING_STATUS_UNAVAILABLE:
    case CheckError::CLOUD_RESOURCE_MANAGER_BACKEND_UNAVAILABLE:
      return Status(
          Code::UNAVAILABLE,
          "One or more Google Service Control backends are unavailable.");

    default:
      return Status(Code::INTERNAL,
                    std::string("Request blocked due to unsupported error code "
                                "in Google Service Control Check response: ") +
                        std::to_string(error.code()));
  }
  return Status::OK;
}

Status CheckResponseConverter::ConvertAllocateQuotaResponse(
    const ::google::api::servicecontrol::v1::AllocateQuotaResponse& response,
    const std::string&, QuotaResponseInfo* quota_response_info) {
  // response.operation_id()
  if (response.allocate_errors().empty()) {
    return Status::OK;
  }

  const ::google::api::servicecontrol::v1::QuotaError& error =
      response.allocate_errors().Get(0);

  quota_response_info->error = {QuotaError_Code_Name(error.code()),
                                /*is_error_from_http_call=*/false,
                                ScResponseErrorType::ERROR_TYPE_UNSPECIFIED};

  ScResponseError& quota_error = quota_response_info->error;
  switch (error.code()) {
    case ::google::api::servicecontrol::v1::QuotaError::UNSPECIFIED:
      // This is never used.
      break;

    case ::google::api::servicecontrol::v1::QuotaError::RESOURCE_EXHAUSTED:
      // Quota allocation failed.
      // Same as [google.rpc.Code.RESOURCE_EXHAUSTED][].
      quota_error.type = ScResponseErrorType::CONSUMER_QUOTA;
      return Status(Code::RESOURCE_EXHAUSTED, error.description());

    case ::google::api::servicecontrol::v1::QuotaError::BILLING_NOT_ACTIVE:
      // Consumer cannot access the service because billing is disabled.
      quota_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::PERMISSION_DENIED, error.description());

    case ::google::api::servicecontrol::v1::QuotaError::PROJECT_DELETED:
      // Consumer's project has been marked as deleted (soft deletion).
      quota_error.type = ScResponseErrorType::CONSUMER_ERROR;
      return Status(Code::INVALID_ARGUMENT, error.description());

    case ::google::api::servicecontrol::v1::QuotaError::API_KEY_INVALID:
      // Specified API key is invalid.
    case ::google::api::servicecontrol::v1::QuotaError::API_KEY_EXPIRED:
      // Specified API Key has expired.
      quota_error.type = ScResponseErrorType::API_KEY_INVALID;
      return Status(Code::INVALID_ARGUMENT, error.description());

    default:
      return Status(Code::INTERNAL, error.description());
  }

  return Status::OK;
}

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2