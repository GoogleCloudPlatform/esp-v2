// Copyright 2020 Google LLC
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

#include "src/envoy/http/service_control/client_cache.h"

#include "common/common/empty_string.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "src/envoy/http/service_control/service_control_callback_func.h"
#include "test/mocks/common.h"
#include "test/mocks/event/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/stats/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace test {

using ::espv2::api_proxy::service_control::CheckResponseInfo;
using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::api::envoy::http::service_control::Service;
using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckError_Code;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
using ::testing::NiceMock;

class ClientCacheCheckResponseTest : public ::testing::Test {
 protected:
  ClientCacheCheckResponseTest() : stats_base_("test", context_.scope_) {
    token_fn_ = []() { return Envoy::EMPTY_STRING; };
  }

  void SetUp() override {
    cache_ = std::make_unique<ClientCache>(
        service_config_, filter_config_, stats_base_.stats(), cm_, time_source_,
        dispatcher_, token_fn_, token_fn_);
  }

  void checkAndReset(Envoy::Stats::Counter& counter, const int expected_value) {
    EXPECT_EQ(counter.value(), expected_value);
    counter.reset();
  }

  void runTest(Code got_http_code, CheckResponse* got_response,
               Code want_client_code) {
    CheckDoneFunc on_done = [&](const Status& status,
                                const CheckResponseInfo&) {
      EXPECT_EQ(status.code(), want_client_code);
    };

    const Status http_status(got_http_code, Envoy::EMPTY_STRING);
    cache_->handleCheckResponse(http_status, got_response, on_done);
  }

  void TearDown() override {
    checkAndReset(stats_base_.stats().filter_.allowed_control_plane_fault_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_control_plane_fault_, 0);
  }

  // Helpers for SetUp.
  Service service_config_;
  FilterConfig filter_config_;
  NiceMock<Envoy::Upstream::MockClusterManager> cm_;
  NiceMock<Envoy::Event::MockDispatcher> dispatcher_;
  NiceMock<Envoy::MockTimeSystem> time_source_;
  NiceMock<Envoy::Server::Configuration::MockFactoryContext> context_;
  ServiceControlFilterStatBase stats_base_;
  std::function<const std::string&()> token_fn_;

  // Class under test.
  std::unique_ptr<ClientCache> cache_;
};

TEST_F(ClientCacheCheckResponseTest, Http5xxAllowed) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::UNAVAILABLE, response, Code::OK);
  checkAndReset(stats_base_.stats().filter_.allowed_control_plane_fault_, 1);
}

TEST_F(ClientCacheCheckResponseTest, Http4xxTranslatedAndBlocked) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::PERMISSION_DENIED, response, Code::INTERNAL);
}

TEST_F(ClientCacheCheckResponseTest, Sc5xxAllowed) {
  CheckResponse* response = new CheckResponse();
  CheckError* check_error = response->mutable_check_errors()->Add();
  check_error->set_code(CheckError::NAMESPACE_LOOKUP_UNAVAILABLE);

  runTest(Code::OK, response, Code::OK);
  checkAndReset(stats_base_.stats().filter_.allowed_control_plane_fault_, 1);
}

TEST_F(ClientCacheCheckResponseTest, Sc4xxBlocked) {
  CheckResponse* response = new CheckResponse();
  CheckError* check_error = response->mutable_check_errors()->Add();
  check_error->set_code(CheckError::CLIENT_APP_BLOCKED);

  runTest(Code::OK, response, Code::PERMISSION_DENIED);
}

TEST_F(ClientCacheCheckResponseTest, ScOkAllowed) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::OK, response, Code::OK);
}

class ClientCacheCheckResponseNetworkFailClosedTest
    : public ClientCacheCheckResponseTest {
  void SetUp() override {
    filter_config_.mutable_sc_calling_config()
        ->mutable_network_fail_open()
        ->set_value(false);
    cache_ = std::make_unique<ClientCache>(
        service_config_, filter_config_, stats_base_.stats(), cm_, time_source_,
        dispatcher_, token_fn_, token_fn_);
  }
};

TEST_F(ClientCacheCheckResponseNetworkFailClosedTest, Http5xxBlocked) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::UNAVAILABLE, response, Code::UNAVAILABLE);
  checkAndReset(stats_base_.stats().filter_.denied_control_plane_fault_, 1);
}

TEST_F(ClientCacheCheckResponseNetworkFailClosedTest, Sc5xxBlocked) {
  CheckResponse* response = new CheckResponse();
  CheckError* check_error = response->mutable_check_errors()->Add();
  check_error->set_code(CheckError::NAMESPACE_LOOKUP_UNAVAILABLE);

  runTest(Code::OK, response, Code::UNAVAILABLE);
  checkAndReset(stats_base_.stats().filter_.denied_control_plane_fault_, 1);
}

class ClientCacheCheckResponseErrorTypeTest
    : public ClientCacheCheckResponseTest {
 protected:
  void runTest(CheckError_Code got_check_error_code) {
    CheckResponse* response = new CheckResponse();
    CheckError* check_error = response->mutable_check_errors()->Add();
    check_error->set_code(got_check_error_code);

    CheckDoneFunc on_done = [&](const Status&, const CheckResponseInfo&) {};
    const Status http_status(Code::OK, Envoy::EMPTY_STRING);
    cache_->handleCheckResponse(http_status, response, on_done);
  }
};

TEST_F(ClientCacheCheckResponseErrorTypeTest, ConsumerBlocked) {
  runTest(CheckError::CLIENT_APP_BLOCKED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_blocked_, 1);
}

TEST_F(ClientCacheCheckResponseErrorTypeTest, ConsumerError) {
  runTest(CheckError::BILLING_DISABLED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

TEST_F(ClientCacheCheckResponseErrorTypeTest, ApiKeyInvalid) {
  runTest(CheckError::API_KEY_NOT_FOUND);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

TEST_F(ClientCacheCheckResponseErrorTypeTest, ServiceNotActivated) {
  runTest(CheckError::SERVICE_NOT_ACTIVATED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

}  // namespace test
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2