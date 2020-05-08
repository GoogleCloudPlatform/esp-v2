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
                                 const char* service_name) {
  gasv1::CheckResponse response;
  gasv1::CheckError* check_error = response.add_check_errors();
  CheckRequestInfo info;
  check_error->set_code(code);
  check_error->set_detail(error_detail);
  return RequestBuilder::ConvertCheckResponse(response, service_name, nullptr);
}

Status ConvertCheckErrorToStatus(gasv1::CheckError::Code code) {
  gasv1::CheckResponse response;
  std::string service_name;
  response.add_check_errors()->set_code(code);
  return RequestBuilder::ConvertCheckResponse(response, service_name, nullptr);
}

}  // namespace

TEST(CheckResponseTest, AbortedWithInvalidArgumentWhenRespIsKeyInvalid) {
  Status result = ConvertCheckErrorToStatus(CheckError::API_KEY_INVALID);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
}

TEST(CheckResponseTest, AbortedWithInvalidArgumentWhenRespIsKeyExpired) {
  Status result = ConvertCheckErrorToStatus(CheckError::API_KEY_EXPIRED);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
}

TEST(CheckResponseTest,
     AbortedWithInvalidArgumentWhenRespIsBlockedWithNotFound) {
  Status result = ConvertCheckErrorToStatus(CheckError::NOT_FOUND);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
}

TEST(CheckResponseTest,
     AbortedWithInvalidArgumentWhenRespIsBlockedWithKeyNotFound) {
  Status result = ConvertCheckErrorToStatus(CheckError::API_KEY_NOT_FOUND);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithServiceNotActivated) {
  Status result = ConvertCheckErrorToStatus(
      CheckError::SERVICE_NOT_ACTIVATED, "Service not activated.", "api_xxxx");
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(result.message(), "API api_xxxx is not enabled for the project.");
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithPermissionDenied) {
  Status result = ConvertCheckErrorToStatus(CheckError::PERMISSION_DENIED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithIpAddressBlocked) {
  Status result = ConvertCheckErrorToStatus(CheckError::IP_ADDRESS_BLOCKED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithRefererBlocked) {
  Status result = ConvertCheckErrorToStatus(CheckError::REFERER_BLOCKED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenRespIsBlockedWithClientAppBlocked) {
  Status result = ConvertCheckErrorToStatus(CheckError::CLIENT_APP_BLOCKED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectDeleted) {
  Status result = ConvertCheckErrorToStatus(CheckError::PROJECT_DELETED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectInvalid) {
  Status result = ConvertCheckErrorToStatus(CheckError::PROJECT_INVALID);
  EXPECT_EQ(Code::INVALID_ARGUMENT, result.code());
}

TEST(CheckResponseTest,
     AbortedWithPermissionDeniedWhenResponseIsBlockedWithBillingDisabled) {
  Status result = ConvertCheckErrorToStatus(CheckError::BILLING_DISABLED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithSecurityPolicyViolated) {
  Status result =
      ConvertCheckErrorToStatus(CheckError::SECURITY_POLICY_VIOLATED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithInvalidCredentail) {
  Status result = ConvertCheckErrorToStatus(CheckError::INVALID_CREDENTIAL);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithLocationPolicyViolated) {
  Status result =
      ConvertCheckErrorToStatus(CheckError::LOCATION_POLICY_VIOLATED);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithConsumerInvalid) {
  Status result = ConvertCheckErrorToStatus(CheckError::CONSUMER_INVALID);
  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithNamespaceLookup) {
  const Status result =
      ConvertCheckErrorToStatus(CheckError::NAMESPACE_LOOKUP_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithBillingStatus) {
  const Status result =
      ConvertCheckErrorToStatus(CheckError::BILLING_STATUS_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithServiceStatus) {
  const Status result =
      ConvertCheckErrorToStatus(CheckError::SERVICE_STATUS_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithQuotaCheck) {
  const Status result =
      ConvertCheckErrorToStatus(CheckError::QUOTA_CHECK_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithCloudResourceManager) {
  const Status result = ConvertCheckErrorToStatus(
      CheckError::CLOUD_RESOURCE_MANAGER_BACKEND_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithSecurityPolicy) {
  const Status result = ConvertCheckErrorToStatus(
      CheckError::SECURITY_POLICY_BACKEND_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

TEST(CheckResponseTest, WhenResponseIsBlockedWithLocationPolicy) {
  const Status result = ConvertCheckErrorToStatus(
      CheckError::LOCATION_POLICY_BACKEND_UNAVAILABLE);
  EXPECT_EQ(Code::UNAVAILABLE, result.code());
}

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
