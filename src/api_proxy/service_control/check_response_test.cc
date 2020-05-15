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

#include "gtest/gtest.h"
#include "src/api_proxy/service_control/request_builder.h"

namespace gasv1 = ::google::api::servicecontrol::v1;

using ::google::api::servicecontrol::v1::CheckError;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace espv2 {
namespace api_proxy {
namespace service_control {

namespace {

Status ConvertCheckErrorToStatus(gasv1::CheckError::Code code,
                                 const char* error_detail,
                                 const char* service_name,
                                 CheckResponseInfo* info) {
  gasv1::CheckResponse response;
  gasv1::CheckError* check_error = response.add_check_errors();
  check_error->set_code(code);
  check_error->set_detail(error_detail);
  return RequestBuilder::ConvertCheckResponse(response, service_name, info);
}

Status ConvertCheckErrorToStatus(gasv1::CheckError::Code code,
                                 CheckResponseInfo* info) {
  gasv1::CheckResponse response;
  std::string service_name;
  response.add_check_errors()->set_code(code);
  return RequestBuilder::ConvertCheckResponse(response, service_name, info);
}

}  // namespace

TEST(CheckResponseTest, AbortedWithInvalidArgumentWhenRespIsKeyInvalid) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::API_KEY_INVALID, &info);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::API_KEY_INVALID);
}

TEST(CheckResponseTest, AbortedWithInvalidArgumentWhenRespIsKeyExpired) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::API_KEY_EXPIRED, &info);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::API_KEY_INVALID);
}

TEST(CheckResponseTest,
     AbortedWithInvalidArgumentWhenRespIsBlockedWithNotFound) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::NOT_FOUND, &info);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest,
     AbortedWithInvalidArgumentWhenRespIsBlockedWithKeyNotFound) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::API_KEY_NOT_FOUND, &info);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::API_KEY_INVALID);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithServiceNotActivated) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::SERVICE_NOT_ACTIVATED,
                                "Service not activated.", "api_xxxx", &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(result.message(), "API api_xxxx is not enabled for the project.");
  EXPECT_EQ(info.error_type, CheckResponseErrorType::SERVICE_NOT_ACTIVATED);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithPermissionDenied) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::PERMISSION_DENIED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithIpAddressBlocked) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::IP_ADDRESS_BLOCKED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithRefererBlocked) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::REFERER_BLOCKED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithClientAppBlocked) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::CLIENT_APP_BLOCKED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectDeleted) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::PROJECT_DELETED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectInvalid) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::PROJECT_INVALID, &info);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithBillingDisabled) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::BILLING_DISABLED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithSecurityPolicyViolated) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::SECURITY_POLICY_VIOLATED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithInvalidCredentail) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::INVALID_CREDENTIAL, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithLocationPolicyViolated) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::LOCATION_POLICY_VIOLATED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithConsumerInvalid) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::CONSUMER_INVALID, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithResourceExhuasted) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::RESOURCE_EXHAUSTED, &info);
  EXPECT_EQ(Code::RESOURCE_EXHAUSTED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithAbuserDetected) {
  CheckResponseInfo info;
  Status result = ConvertCheckErrorToStatus(CheckError::ABUSER_DETECTED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_ERROR);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithApiTargetBlocked) {
  CheckResponseInfo info;
  Status result =
      ConvertCheckErrorToStatus(CheckError::API_TARGET_BLOCKED, &info);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::CONSUMER_BLOCKED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithNamespaceLookup) {
  CheckResponseInfo info;
  const Status result = ConvertCheckErrorToStatus(
      CheckError::NAMESPACE_LOOKUP_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithBillingStatus) {
  CheckResponseInfo info;
  const Status result =
      ConvertCheckErrorToStatus(CheckError::BILLING_STATUS_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithServiceStatus) {
  CheckResponseInfo info;
  const Status result =
      ConvertCheckErrorToStatus(CheckError::SERVICE_STATUS_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithQuotaCheck) {
  CheckResponseInfo info;
  const Status result =
      ConvertCheckErrorToStatus(CheckError::QUOTA_CHECK_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithCloudResourceManager) {
  CheckResponseInfo info;
  const Status result = ConvertCheckErrorToStatus(
      CheckError::CLOUD_RESOURCE_MANAGER_BACKEND_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithSecurityPolicy) {
  CheckResponseInfo info;
  const Status result = ConvertCheckErrorToStatus(
      CheckError::SECURITY_POLICY_BACKEND_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithLocationPolicy) {
  CheckResponseInfo info;
  const Status result = ConvertCheckErrorToStatus(
      CheckError::LOCATION_POLICY_BACKEND_UNAVAILABLE, &info);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
  EXPECT_EQ(info.error_type, CheckResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
