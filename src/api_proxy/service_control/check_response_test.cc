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

#include "src/api_proxy/service_control/request_builder.h"

#include "gtest/gtest.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {
namespace {

using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckError_Code;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

class ConvertCheckResponseTest : public ::testing::Test {
 protected:
  void runTest(CheckError_Code got_check_error_code, Code want_code,
               ScResponseErrorType want_error_type) {
    CheckResponseInfo info;
    CheckResponse response;
    response.add_check_errors()->set_code(got_check_error_code);

    Status result = RequestBuilder::ConvertCheckResponse(response, "", &info);

    EXPECT_EQ(want_code, result.code());
    EXPECT_EQ(info.error_type, want_error_type);
  }
};

TEST_F(ConvertCheckResponseTest,
       AbortedWithInvalidArgumentWhenRespIsKeyInvalid) {
  runTest(CheckError::API_KEY_INVALID, Code::INVALID_ARGUMENT,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithInvalidArgumentWhenRespIsKeyExpired) {
  runTest(CheckError::API_KEY_EXPIRED, Code::INVALID_ARGUMENT,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithInvalidArgumentWhenRespIsBlockedWithKeyNotFound) {
  runTest(CheckError::API_KEY_NOT_FOUND, Code::INVALID_ARGUMENT,
          ScResponseErrorType::API_KEY_INVALID);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithInvalidArgumentWhenRespIsBlockedWithNotFound) {
  runTest(CheckError::NOT_FOUND, Code::INVALID_ARGUMENT,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithPermissionDenied) {
  runTest(CheckError::PERMISSION_DENIED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithIpAddressBlocked) {
  runTest(CheckError::IP_ADDRESS_BLOCKED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithRefererBlocked) {
  runTest(CheckError::REFERER_BLOCKED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithClientAppBlocked) {
  runTest(CheckError::CLIENT_APP_BLOCKED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectDeleted) {
  runTest(CheckError::PROJECT_DELETED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithProjectInvalid) {
  runTest(CheckError::PROJECT_INVALID, Code::INVALID_ARGUMENT,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenResponseIsBlockedWithBillingDisabled) {
  runTest(CheckError::BILLING_DISABLED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithInvalidCredentail) {
  runTest(CheckError::INVALID_CREDENTIAL, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithConsumerInvalid) {
  runTest(CheckError::CONSUMER_INVALID, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_ERROR);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithResourceExhuasted) {
  runTest(CheckError::RESOURCE_EXHAUSTED, Code::RESOURCE_EXHAUSTED,
          ScResponseErrorType::CONSUMER_QUOTA);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithApiTargetBlocked) {
  runTest(CheckError::API_TARGET_BLOCKED, Code::PERMISSION_DENIED,
          ScResponseErrorType::CONSUMER_BLOCKED);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithNamespaceLookup) {
  runTest(CheckError::NAMESPACE_LOOKUP_UNAVAILABLE, Code::UNAVAILABLE,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithBillingStatus) {
  runTest(CheckError::BILLING_STATUS_UNAVAILABLE, Code::UNAVAILABLE,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(ConvertCheckResponseTest, WhenResponseIsBlockedWithServiceStatus) {
  runTest(CheckError::SERVICE_STATUS_UNAVAILABLE, Code::UNAVAILABLE,
          ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(ConvertCheckResponseTest,
       WhenResponseIsBlockedWithCloudResourceManager) {
  runTest(CheckError::CLOUD_RESOURCE_MANAGER_BACKEND_UNAVAILABLE,
          Code::UNAVAILABLE, ScResponseErrorType::ERROR_TYPE_UNSPECIFIED);
}

TEST_F(ConvertCheckResponseTest,
       AbortedWithPermissionDeniedWhenRespIsBlockedWithServiceNotActivated) {
  CheckResponseInfo info;
  CheckResponse response;
  CheckError* check_error = response.add_check_errors();
  check_error->set_code(CheckError::SERVICE_NOT_ACTIVATED);
  check_error->set_detail("Service not activated.");

  Status result =
      RequestBuilder::ConvertCheckResponse(response, "api_xxxx", &info);

  EXPECT_EQ(Code::PERMISSION_DENIED, result.code());
  EXPECT_EQ(result.message(), "API api_xxxx is not enabled for the project.");
  EXPECT_EQ(info.error_type, ScResponseErrorType::SERVICE_NOT_ACTIVATED);
}

}  // namespace
}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
