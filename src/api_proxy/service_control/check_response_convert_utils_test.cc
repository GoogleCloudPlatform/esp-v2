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

#include "src/api_proxy/service_control/check_response_convert_utils.h"

#include "gtest/gtest.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {
namespace {

using ::absl::OkStatus;
using absl::Status;
using absl::StatusCode;
using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckError_Code;
using ::google::api::servicecontrol::v1::CheckError_Code_Name;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::
    CheckResponse_ConsumerInfo_ConsumerType;

class CheckResponseConverterTest : public ::testing::Test {
 protected:
  void runTest(CheckError_Code got_check_error_code, StatusCode want_code,
               ScResponseErrorType want_error_type) {
    CheckResponseInfo info;
    CheckResponse response;
    response.add_check_errors()->set_code(got_check_error_code);

    Status result = ConvertCheckResponse(response, "", &info);

    EXPECT_EQ(result.code(), want_code);
    EXPECT_EQ(info.error.type, want_error_type);
    EXPECT_EQ(info.error.name, CheckError_Code_Name(got_check_error_code));
  }
};

TEST_F(CheckResponseConverterTest,
       AbortedWithInvalidArgumentWhenRespIsKeyInvalid) {
  runTest(CheckError::API_KEY_INVALID, StatusCode::kInvalidArgument,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithInvalidArgumentWhenRespIsKeyExpired) {
  runTest(CheckError::API_KEY_EXPIRED, StatusCode::kInvalidArgument,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithInvalidArgumentWhenRespIsBlockedWithKeyNotFound) {
  runTest(CheckError::API_KEY_NOT_FOUND, StatusCode::kInvalidArgument,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithInvalidArgumentWhenRespIsBlockedWithNotFound) {
  runTest(CheckError::NOT_FOUND, StatusCode::kInvalidArgument,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithPermissionDenied) {
  runTest(CheckError::PERMISSION_DENIED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithIpAddressBlocked) {
  runTest(CheckError::IP_ADDRESS_BLOCKED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithRefererBlocked) {
  runTest(CheckError::REFERER_BLOCKED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithClientAppBlocked) {
  runTest(CheckError::CLIENT_APP_BLOCKED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectDeleted) {
  runTest(CheckError::PROJECT_DELETED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectInvalid) {
  runTest(CheckError::PROJECT_INVALID, StatusCode::kInvalidArgument,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithBillingDisabled) {
  runTest(CheckError::BILLING_DISABLED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithInvalidCredentail) {
  runTest(CheckError::INVALID_CREDENTIAL, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithConsumerInvalid) {
  runTest(CheckError::CONSUMER_INVALID, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithResourceExhuasted) {
  runTest(CheckError::RESOURCE_EXHAUSTED, StatusCode::kResourceExhausted,
          ScResponseErrorType::CONSUMER_QUOTA);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithApiTargetBlocked) {
  runTest(CheckError::API_TARGET_BLOCKED, StatusCode::kPermissionDenied,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithNamespaceLookup) {
  runTest(CheckError::NAMESPACE_LOOKUP_UNAVAILABLE, StatusCode::kUnavailable,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithBillingStatus) {
  runTest(CheckError::BILLING_STATUS_UNAVAILABLE, StatusCode::kUnavailable,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(CheckResponseConverterTest, WhenResponseIsBlockedWithServiceStatus) {
  runTest(CheckError::SERVICE_STATUS_UNAVAILABLE, StatusCode::kUnavailable,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(CheckResponseConverterTest,
       WhenResponseIsBlockedWithCloudResourceManager) {
  runTest(CheckError::CLOUD_RESOURCE_MANAGER_BACKEND_UNAVAILABLE,
          StatusCode::kUnavailable,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(CheckResponseConverterTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithServiceNotActivated) {
  CheckResponseInfo info;
  CheckResponse response;
  CheckError* check_error = response.add_check_errors();
  check_error->set_code(CheckError::SERVICE_NOT_ACTIVATED);
  check_error->set_detail("Service not activated.");

  Status result = ConvertCheckResponse(response, "api_xxxx", &info);

  EXPECT_EQ(StatusCode::kPermissionDenied, result.code());
  EXPECT_EQ(result.message(), "API api_xxxx is not enabled for the project.");
  EXPECT_EQ(info.error.type, ScResponseErrorType::SERVICE_NOT_ACTIVATED);
}

TEST_F(CheckResponseConverterTest, ConvertConsumerInfo) {
  CheckResponseInfo info;
  CheckResponse response;
  int consumer_number = 123456;
  CheckResponse_ConsumerInfo_ConsumerType type =
      CheckResponse_ConsumerInfo_ConsumerType::
          CheckResponse_ConsumerInfo_ConsumerType_PROJECT;
  response.mutable_check_info()->mutable_consumer_info()->set_project_number(
      consumer_number);
  response.mutable_check_info()->mutable_consumer_info()->set_type(type);
  response.mutable_check_info()->mutable_consumer_info()->set_consumer_number(
      consumer_number);

  Status result = ConvertCheckResponse(response, "api_xxxx", &info);

  EXPECT_EQ(info.consumer_project_number, std::to_string(consumer_number));
  EXPECT_EQ(info.consumer_type,
            CheckResponse_ConsumerInfo_ConsumerType_Name(type));
  EXPECT_EQ(info.consumer_number, std::to_string(consumer_number));
}

}  // namespace
}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
