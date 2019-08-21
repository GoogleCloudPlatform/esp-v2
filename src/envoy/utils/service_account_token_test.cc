// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include <memory>

#include "src/envoy/utils/json_struct.h"
#include "src/envoy/utils/service_account_token.h"

#include "test/mocks/init/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "gmock/gmock-generated-function-mockers.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {

using ::Envoy::Server::Configuration::MockFactoryContext;

using ::testing::_;
using ::testing::MockFunction;

const std::string kTestServiceAccountKey = R"({
  "type": "service_account",
  "project_id": "cloudesf-dummy",
  "private_key_id": "dcb3a004c6c21f45e3b65625979b11e9e96fc0f6",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCuIIFSp1UvHAmn\n0SMYb8tPXaOr6DP+f3vVymQ7VI9ZHC8WWDV9A89NrVp2FKMyYqlN+nyBnJQ8Vnx/\ntmhEoPIJXvTwtC3xVHeJOaNNe/JfaANMy7FNGJzwtGXpFa8ZjUUqD+5CXFAq0wNR\njih/iUmUnoDMjOwNz5IFT9sfnmIsrYCXqhw62ki9hgb5KWTOpuVqbS7KhOwfpnhY\nX6kH4+zFWglsWdMbISwRiMeIlQDa8XC+YM9id60EPIfILbnJO/TayGFD/ukxslaP\nlH580f/vGuJ6ahChf7tYd2J/4MKU/2P0WWlOO5KKsz/17BEH6iCqMmECUNXeYVl4\nnvWmymD1AgMBAAECggEAPo+yNzqkyfTGaVOkSuLbxsurgxe+Gpm+Ke16PrDeghM0\nvc/6g8yrHksC/frjObamArzVIBJcViNyvsYQR1wWKhTCZ3stKJCDFDwvtqaqSeoK\niXyD2uHVfUwrc2fVjhYqO/cWUSRurzw6bIJpfY0bcTjTqOqW401pNtxeq8kRl+A0\nL13Vli71UjGKMhROIAPiMQAuEE8zQUYD/LuB/YyePIPxX/8f/tkxQ/yshSmmfVED\nJtRiW3K/LSIZbBe7VJihFG2CgVS/3kR7ZQmOsLyqeHsb+6P4M1qjmkwuy8dXRB/h\nMQwbze3o0bfF1LdZ5wvWso+8CpToTzQViSRYqc73/wKBgQDojm6zU5Imb98ppXNT\nrGpOzupptHiTIyg8Dh2CFfmeSfFvFwSLJ8EDtfvK7HhS0iCIAjlIGKEnOvaG6p+j\nLFcHf9LqlufVRisTalcJAUcGgYDwGyQOaL7eosv7KE7P4io8KdxnzPLEyJcxNiv8\nrnQMjIWBCWo0dwgqivQQvss+NwKBgQC/rjBZEWqHISuMhjOElikPTMmNT/YIKzaJ\nXXr3omgIroQz8MWHuguC+udgmjJaCXO5g4Kmpp0rwA0Ib49ujycAHrRlD0Mx1DnD\noFhuHRDbSf/xnSRGNlo1TTqgVukuDLzoPot03lYeju8AyhOrGmlKU0frvOX37ZqM\nmvyKVDzkMwKBgQCtw+dxdQtqTwMXsjmHFvhkJHXBQAksIAPrQ7zGu7bFkIinMjLB\n65VsOWmHycNqVvnZxpeYiFa54nPcgamAmhv5TYiCovldQc3j9vxLjTnN4aw/PHhn\nj9q2rjvuUcL50Asw4zJ+GQR5B0z5h3m8l3m8+q6yqR9DToG6kBMoA/gHZwKBgHeO\nQS+80jIYuV378rQ3KMMXRPu0LSQpN+nz+Zftn3AS0fjHq50dqMJ4lsrFQrSwApNq\neJpTf+Li9f4V/2OZPF0xyZjjLSkuUx02rRF5ZaMxg8eDGTYF/rwSQIfzzZtgbI97\nO2aYqySCSIa4hA4L+jJWwZxDBTlf5S7gGLZ7FkPLAoGAT5e9tG2wQhmHpdIhx1Jd\nQqTfa6a9MRfmjsGeybilODTO1fGScj1Rq9clAg9ugaBdOv8CIUa5tve0lCB/tJEN\nCTqUYvcWxy/fWxZjlj+ovlNfgyXguXI2v0HOxY+nTGhybK3A0tyBPl53yHY8Xf9X\ne1HSaGWkxgTgfJUM9bGELB8=\n-----END PRIVATE KEY-----\n",
  "client_email": "dummy-73@cloudesf-dummy.iam.gserviceaccount.com",
  "client_id": "118366583695942267742",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/dummy-73%40cloudesf-dummy.iam.gserviceaccount.com"
})";

class ServiceAccountTokenTest : public testing::Test {
 protected:
  NiceMock<MockFactoryContext> context_;
  MockFunction<int(std::string)> token_callback_;
  ServiceAccountTokenPtr sc_token_;

};  // namespace

TEST_F(ServiceAccountTokenTest, MakeCallbackOnRefresh) {
  EXPECT_CALL(token_callback_, Call(_)).Times(1);
  sc_token_ = std::make_unique<ServiceAccountToken>(
      context_, kTestServiceAccountKey, "audience",
      token_callback_.AsStdFunction());
}
}  // namespace
}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
