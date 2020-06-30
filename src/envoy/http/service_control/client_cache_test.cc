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

#include "absl/functional/bind_front.h"
#include "src/envoy/http/service_control/mocks.h"
#include "src/envoy/http/service_control/service_control_callback_func.h"
#include "test/mocks/common.h"
#include "test/mocks/event/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/stats/mocks.h"
#include "test/mocks/tracing/mocks.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace test {

using ::espv2::api::envoy::v6::http::service_control::FilterConfig;
using ::espv2::api::envoy::v6::http::service_control::Service;
using ::espv2::api_proxy::service_control::CheckResponseInfo;
using ::google::api::servicecontrol::v1::AllocateQuotaRequest;
using ::google::api::servicecontrol::v1::AllocateQuotaResponse;
using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckError_Code;
using ::google::api::servicecontrol::v1::CheckRequest;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::Operation;
using ::google::api::servicecontrol::v1::QuotaError;
using ::google::api::servicecontrol::v1::QuotaError_Code;
using ::google::api::servicecontrol::v1::ReportRequest;
using ::google::api::servicecontrol::v1::ReportResponse;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

using ::testing::_;
using ::testing::InSequence;
using ::testing::NiceMock;
using ::testing::Return;

constexpr char kServiceName[] = "bookstore.endpoints.test";
constexpr char kServiceConfigId[] = "2020-06-24r1";
constexpr char kCheckOperationId[] = "test.check.operation";

class ClientCacheTestBase : public ::testing::Test {
 protected:
  ClientCacheTestBase() : stats_base_("test", context_.scope_) {
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

  void TearDown() override {
    // All stats that are verified in tests below should be reset here.
    // Response tests.
    checkAndReset(stats_base_.stats().filter_.allowed_control_plane_fault_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_control_plane_fault_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_consumer_blocked_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_consumer_quota_, 0);
    checkAndReset(stats_base_.stats().filter_.denied_producer_error_, 0);

    // Check request tests.
    checkAndReset(stats_base_.stats().check_.OK_, 0);
    checkAndReset(stats_base_.stats().check_.CANCELLED_, 0);
    checkAndReset(stats_base_.stats().check_.INVALID_ARGUMENT_, 0);
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

class ClientCacheCheckResponseTest : public ClientCacheTestBase {
 protected:
  void runTest(Code got_http_code, CheckResponse* got_response,
               Code want_client_code) {
    CheckDoneFunc on_done = [&](const Status& status,
                                const CheckResponseInfo&) {
      EXPECT_EQ(status.code(), want_client_code);
    };

    const Status http_status(got_http_code, Envoy::EMPTY_STRING);
    cache_->handleCheckResponse(http_status, got_response, on_done);
  }
};

TEST_F(ClientCacheCheckResponseTest, Http5xxAllowed) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::UNAVAILABLE, response, Code::OK);
  checkAndReset(stats_base_.stats().filter_.allowed_control_plane_fault_, 1);
}

TEST_F(ClientCacheCheckResponseTest, Http4xxTranslatedAndBlocked) {
  CheckResponse* response = new CheckResponse();

  runTest(Code::PERMISSION_DENIED, response, Code::INTERNAL);
  checkAndReset(stats_base_.stats().filter_.denied_producer_error_, 1);
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
  checkAndReset(stats_base_.stats().filter_.denied_consumer_blocked_, 1);
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

class ClientCacheCheckResponseErrorTypeTest : public ClientCacheTestBase {
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

// This should never happen since we use quota calls, but test it for
// completeness.
TEST_F(ClientCacheCheckResponseErrorTypeTest, ConsumerQuota) {
  runTest(CheckError::RESOURCE_EXHAUSTED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_quota_, 1);
}

TEST_F(ClientCacheCheckResponseErrorTypeTest, ApiKeyInvalid) {
  runTest(CheckError::API_KEY_NOT_FOUND);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

TEST_F(ClientCacheCheckResponseErrorTypeTest, ServiceNotActivated) {
  runTest(CheckError::SERVICE_NOT_ACTIVATED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

class ClientCacheQuotaResponseTest : public ClientCacheTestBase {
 protected:
  void runTest(Code got_http_code, AllocateQuotaResponse* got_response,
               Code want_client_code) {
    QuotaDoneFunc on_done = [&](const Status& status) {
      EXPECT_EQ(status.code(), want_client_code);
    };

    const Status http_status(got_http_code, Envoy::EMPTY_STRING);
    cache_->handleQuotaOnDone(http_status, got_response, on_done);
  }
};

TEST_F(ClientCacheQuotaResponseTest, HttpErrorBlocked) {
  AllocateQuotaResponse* response = new AllocateQuotaResponse();

  runTest(Code::INTERNAL, response, Code::INTERNAL);
  checkAndReset(stats_base_.stats().filter_.denied_producer_error_, 1);
}

TEST_F(ClientCacheQuotaResponseTest, ScErrorBlocked) {
  AllocateQuotaResponse* response = new AllocateQuotaResponse();
  QuotaError* quota_error = response->mutable_allocate_errors()->Add();
  quota_error->set_code(QuotaError::RESOURCE_EXHAUSTED);

  runTest(Code::OK, response, Code::RESOURCE_EXHAUSTED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_quota_, 1);
}

TEST_F(ClientCacheQuotaResponseTest, ScOkAllowed) {
  AllocateQuotaResponse* response = new AllocateQuotaResponse();

  runTest(Code::OK, response, Code::OK);
}

class ClientCacheQuotaResponseErrorTypeTest : public ClientCacheTestBase {
 protected:
  void runTest(QuotaError_Code got_quota_error_code) {
    AllocateQuotaResponse* response = new AllocateQuotaResponse();
    QuotaError* quota_error = response->mutable_allocate_errors()->Add();
    quota_error->set_code(got_quota_error_code);

    QuotaDoneFunc on_done = [&](const Status&) {};
    const Status http_status(Code::OK, Envoy::EMPTY_STRING);
    cache_->handleQuotaOnDone(http_status, response, on_done);
  }
};

TEST_F(ClientCacheQuotaResponseErrorTypeTest, ConsumerError) {
  runTest(QuotaError::PROJECT_DELETED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

TEST_F(ClientCacheQuotaResponseErrorTypeTest, ConsumerQuota) {
  runTest(QuotaError::RESOURCE_EXHAUSTED);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_quota_, 1);
}

TEST_F(ClientCacheQuotaResponseErrorTypeTest, ApiKeyInvalid) {
  runTest(QuotaError::API_KEY_INVALID);
  checkAndReset(stats_base_.stats().filter_.denied_consumer_error_, 1);
}

class ClientCacheHttpRequestTest : public ClientCacheTestBase {
 public:
  void SetUp() override {
    service_config_.set_service_name(kServiceName);
    service_config_.set_service_config_id(kServiceConfigId);

    cache_ = std::make_unique<ClientCache>(
        service_config_, filter_config_, stats_base_.stats(), cm_, time_source_,
        dispatcher_, token_fn_, token_fn_);

    // Setup mock http call.
    http_call_ = std::make_unique<MockHttpCall>();
    check_call_factory_ = std::make_unique<MockHttpCallFactory>();
    quota_call_factory_ = std::make_unique<MockHttpCallFactory>();
    report_call_factory_ = std::make_unique<MockHttpCallFactory>();
  }

  void TearDown() override { ClientCacheTestBase::TearDown(); }

  void injectFactoryMocks() {
    cache_->check_call_factory_ = std::move(check_call_factory_);
    cache_->quota_call_factory_ = std::move(quota_call_factory_);
    cache_->report_call_factory_ = std::move(report_call_factory_);
  }

  int got_num_callbacks_ = 0;
  NiceMock<Envoy::Tracing::MockSpan> mock_parent_span_;
  std::unique_ptr<MockHttpCall> http_call_;

  // Ownership of these is passed to ClientCache after calling
  // `injectFactoryMocks`.
  std::unique_ptr<MockHttpCallFactory> check_call_factory_;
  std::unique_ptr<MockHttpCallFactory> quota_call_factory_;
  std::unique_ptr<MockHttpCallFactory> report_call_factory_;
};

class ClientCacheCheckHttpRequestTest : public ClientCacheHttpRequestTest {
 public:
  void SetUp() override { ClientCacheHttpRequestTest::SetUp(); }

  void setupHttpMocks(int want_http_calls_on_check,
                      int want_http_calls_on_flush) {
    EXPECT_CALL(*http_call_, call())
        .Times(want_http_calls_on_check + want_http_calls_on_flush);

    // This is for cache misses.
    InSequence s;
    EXPECT_CALL(*check_call_factory_, createHttpCall(_, _, _))
        .Times(want_http_calls_on_check)
        .WillRepeatedly(
            Invoke([this](const Envoy::Protobuf::Message&,
                          Envoy::Tracing::Span&, HttpCall::DoneFunc on_done) {
              http_done_ = on_done;
              return http_call_.get();
            }));

    // This is for cache flushes on destruction of the cache.
    EXPECT_CALL(*check_call_factory_, createHttpCall(_, _, _))
        .Times(want_http_calls_on_flush)
        .WillRepeatedly(
            Invoke([this](const Envoy::Protobuf::Message&,
                          Envoy::Tracing::Span&, HttpCall::DoneFunc on_done) {
              // Similar to production behavior of the HttpCallFactory.
              on_done(Status(Code::CANCELLED, "Request cancelled"),
                      Envoy::EMPTY_STRING);
              return http_call_.get();
            }));

    injectFactoryMocks();
  }

  CheckRequest getValidCheckRequest() {
    CheckRequest request;
    request.set_service_name(kServiceName);
    request.set_service_config_id(kServiceConfigId);
    Operation* op = request.mutable_operation();
    op->set_operation_id(kCheckOperationId);
    op->set_operation_name("test_check_operation_name");
    op->set_consumer_id("test-api-key");
    return request;
  }

  CheckResponse getValidCheckResponse() {
    CheckResponse response;
    response.set_operation_id(kCheckOperationId);
    response.set_service_config_id(kServiceConfigId);
    return response;
  }

  HttpCall::DoneFunc http_done_;
};

// Cache miss occurs, so cache makes HttpCall to SC Check.
// Call is successful, and the CheckDoneFunc is called.
TEST_F(ClientCacheCheckHttpRequestTest, OneSuccessfulHttpCall) {
  setupHttpMocks(1, 0);

  const CheckRequest request = getValidCheckRequest();
  cache_->callCheck(request, mock_parent_span_,
                    [this](const Status& got_status, const CheckResponseInfo&) {
                      got_num_callbacks_++;
                      EXPECT_EQ(got_status.code(), Code::OK);
                    });

  // RPC is pending, no callback invoked until http is done.
  EXPECT_EQ(got_num_callbacks_, 0);

  // Stimulate successful http response.
  // Test tear down will check the check callback is invoked.
  std::string response_body;
  const CheckResponse response = getValidCheckResponse();
  response.SerializeToString(&response_body);
  http_done_(Status::OK, response_body);

  // RPC finished and invoked callback.
  EXPECT_EQ(got_num_callbacks_, 1);

  // Force destructor on cache.
  cache_.reset(nullptr);

  // Check stats.
  checkAndReset(stats_base_.stats().check_.OK_, 1);
}

// Cache miss occurs, so cache makes HttpCall to SC Check.
// HttpCall is successful but returns a bad body.
// The CheckDoneFunc is called.
TEST_F(ClientCacheCheckHttpRequestTest, OneHttpCallWithBadBody) {
  setupHttpMocks(1, 0);

  const CheckRequest request = getValidCheckRequest();
  cache_->callCheck(request, mock_parent_span_,
                    [this](const Status& got_status, const CheckResponseInfo&) {
                      got_num_callbacks_++;
                      EXPECT_EQ(got_status.code(), Code::INTERNAL);
                    });

  // RPC is pending, no callback invoked until http is done.
  EXPECT_EQ(got_num_callbacks_, 0);

  // Stimulate bad http response body.
  http_done_(Status::OK, "this http body does not parse into a CheckResponse");

  // RPC finished and invoked callback.
  EXPECT_EQ(got_num_callbacks_, 1);

  // Force destructor on cache.
  cache_.reset(nullptr);

  // Check stats.
  checkAndReset(stats_base_.stats().check_.INVALID_ARGUMENT_, 1);
  // TODO(nareddyt): This should probably be treated as a control plane fault.
  checkAndReset(stats_base_.stats().filter_.denied_producer_error_, 1);
}

// Cache miss occurs, so cache makes HttpCall to SC Check.
// HttpCall is cancelled while it's still pending.
// The CheckDoneFunc is called.
TEST_F(ClientCacheCheckHttpRequestTest, OnePendingHttpCallCancelled) {
  setupHttpMocks(1, 0);

  const CheckRequest request = getValidCheckRequest();
  CancelFunc cancel_func = cache_->callCheck(
      request, mock_parent_span_,
      [this](const Status& got_status, const CheckResponseInfo&) {
        got_num_callbacks_++;
        EXPECT_EQ(got_status.code(), Code::INTERNAL);
      });

  // RPC is pending, no callback invoked until http is done.
  EXPECT_EQ(got_num_callbacks_, 0);

  // Cancel the pending RPC.
  EXPECT_CALL(*http_call_, cancel()).WillOnce(Invoke([this]() {
    http_done_(Status(Code::CANCELLED, "Request cancelled"),
               Envoy::EMPTY_STRING);
  }));
  cancel_func();

  // RPC cancelled and invoked callback.
  EXPECT_EQ(got_num_callbacks_, 1);

  // Force destructor on cache.
  cache_.reset(nullptr);

  // Check stats.
  checkAndReset(stats_base_.stats().check_.CANCELLED_, 1);
  // TODO(nareddyt): This should probably be treated as a control plane fault.
  checkAndReset(stats_base_.stats().filter_.denied_producer_error_, 1);
}

// Check call 1: Cache miss occurs, so cache makes HttpCall to SC Check.
// HttpCall is successful, and the onCheckDone callback is called.
// Check call 2 & 3: Cache hit, the CheckDoneFunc is called again.
TEST_F(ClientCacheCheckHttpRequestTest, SuccessfulHttpCallWithCache) {
  // First http call is due to the first miss, the second call is for cache
  // flush on destruction.
  setupHttpMocks(1, 1);

  CheckDoneFunc on_check_done = [this](const Status& got_status,
                                       const CheckResponseInfo&) {
    got_num_callbacks_++;
    EXPECT_EQ(got_status.code(), Code::OK);
  };

  // Check call 1.
  const CheckRequest request = getValidCheckRequest();
  cache_->callCheck(request, mock_parent_span_, on_check_done);

  // Stimulate successful http response.
  // Test tear down will check the check callback is invoked.
  std::string response_body;
  const CheckResponse response = getValidCheckResponse();
  response.SerializeToString(&response_body);
  http_done_(Status::OK, response_body);

  // Check call 2 & 3.
  cache_->callCheck(request, mock_parent_span_, on_check_done);
  cache_->callCheck(request, mock_parent_span_, on_check_done);

  // 2nd + 3rd call successful due to cache, but only 1 http call was made.
  EXPECT_EQ(got_num_callbacks_, 3);

  // Force destructor on cache. This will result in a cache flush.
  cache_.reset(nullptr);

  // No more callbacks invoked during destructor.
  EXPECT_EQ(got_num_callbacks_, 3);

  // Stats. Account for cancellation on cache flush.
  checkAndReset(stats_base_.stats().check_.OK_, 1);
  checkAndReset(stats_base_.stats().check_.CANCELLED_, 1);
}

}  // namespace test
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2