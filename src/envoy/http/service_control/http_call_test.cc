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

#include "src/envoy/http/service_control/http_call.h"
#include "absl/strings/string_view.h"
#include "common/http/headers.h"
#include "common/tracing/http_tracer_impl.h"
#include "envoy/http/async_client.h"
#include "google/api/servicecontrol/v1/service_controller.pb.h"
#include "google/protobuf/stubs/status.h"

#include <vector>

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "test/mocks/common.h"
#include "test/mocks/event/mocks.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/tracing/mocks.h"
#include "test/test_common/utility.h"

using ::testing::_;
using ::testing::AtLeast;
using ::testing::ByMove;
using ::testing::Invoke;
using ::testing::MockFunction;
using ::testing::Return;

using ::google::api::envoy::http::common::HttpUri;
using ::google::api::servicecontrol::v1::CheckRequest;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

class HttpCallTest : public testing::Test {
 protected:
  HttpCallTest()
      : async_callbacks_(),
        fake_token_("fake-token-value"),
        fake_trace_operation_name_("fake-trace-operation-name"),
        fake_suffix_url_("fake-suffix-url"),
        timeout_ms_(5000),
        retries_(0) {}

  void SetUp() override {
    http_uri_.set_cluster("test_cluster");
    http_uri_.set_uri("http://test_host/test_path");

    ON_CALL(cm_, httpAsyncClientForCluster("test_cluster"))
        .WillByDefault(ReturnRef(http_client_));
    ON_CALL(http_client_, send_(_, _, _))
        .WillByDefault(Invoke([this](Http::MessagePtr& message_ptr,
                                     Http::AsyncClient::Callbacks& callbacks,
                                     const Http::AsyncClient::RequestOptions)
                                  -> Http::AsyncClient::Request* {
          // Check token is correctly set
          auto token_header =
              message_ptr->headers().get(Http::Headers::get().Authorization);
          EXPECT_EQ(token_header->value().getStringView(),
                    "Bearer " + fake_token_);

          // Make callback and request
          async_callbacks_.push_back(&callbacks);
          auto request =
              new NiceMock<Http::MockAsyncClientRequest>(&http_client_);
          http_requests_.push_back(request);
          return request;
        }));

    fake_token_fn_ = [this]() -> const std::string& { return fake_token_; };

    fake_request_ = CheckRequest{};
    http_call_factory_ = std::make_unique<HttpCallFactory>(
        cm_, dispatcher_, http_uri_, fake_suffix_url_, fake_token_fn_,
        timeout_ms_, retries_, mock_time_source_, fake_trace_operation_name_);
  }

  void TearDown() override {
    for (auto request : http_requests_) {
      delete (request);
    }
  }

  NiceMock<Tracing::MockSpan>* makeMockChildSpan() {
    auto span_name = http_requests_.empty()
                         ? fake_trace_operation_name_
                         : absl::StrCat(fake_trace_operation_name_, " - Retry ",
                                        http_requests_.size());

    auto mock_child_span_ptr = new NiceMock<Tracing::MockSpan>();

    EXPECT_CALL(*mock_child_span_ptr, setTag(_, _)).Times(AtLeast(1));
    EXPECT_CALL(*mock_child_span_ptr, finishSpan())
        .Times(0);  // Span should not be finished until response
    EXPECT_CALL(mock_parent_span_, spawnChild_(_, span_name, _))
        .WillOnce(Return(mock_child_span_ptr));

    return mock_child_span_ptr;
  }

  static Http::MessagePtr makeResponseWithStatus(const uint64_t status_code) {
    // Headers with status code
    Http::HeaderMapPtr header_map = std::make_unique<Http::HeaderMapImpl>();
    header_map->setStatus(status_code);

    // Message with no body
    return std::make_unique<Http::ResponseMessageImpl>(std::move(header_map));
  }

  // Callback for HttpCall. Expectations must be set by each test
  MockFunction<void(const ::google::protobuf::util::Status& status,
                    const std::string& response_body)>
      mock_done_fn_;

  // Underlying http client mocks
  HttpUri http_uri_;
  NiceMock<Upstream::MockClusterManager> cm_;
  NiceMock<Event::MockDispatcher> dispatcher_;
  NiceMock<Http::MockAsyncClient> http_client_;

  // Keep track of all underlying http client callbacks and http requests
  std::vector<Http::AsyncClient::Callbacks*> async_callbacks_;
  std::vector<Http::MockAsyncClientRequest*> http_requests_;

  // Token
  std::string fake_token_;
  std::function<const std::string&()> fake_token_fn_;

  // Tracing
  std::string fake_trace_operation_name_;
  NiceMock<Tracing::MockSpan> mock_parent_span_;
  NiceMock<MockTimeSystem> mock_time_source_;

  // Other hardcoded fake parameters
  CheckRequest fake_request_;
  std::string fake_suffix_url_;
  uint32_t timeout_ms_;
  uint32_t retries_;

  std::unique_ptr<HttpCallFactory> http_call_factory_;
};

TEST_F(HttpCallTest, TestSingleCallSuccessHttpOk) {
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response
  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate successful http response
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_, Call(Status::OK, _)).Times(1);

  async_callbacks_[0]->onSuccess(makeResponseWithStatus(200));
}

TEST_F(HttpCallTest, TestSingleCallSuccessHttpNotFound) {
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate successful http response, but a bad status code
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_,
              Call(Status(Code::INTERNAL, "Failed to call service control"), _))
      .Times(1);

  async_callbacks_[0]->onSuccess(makeResponseWithStatus(503));
}

TEST_F(HttpCallTest, TestSingleCallFailure) {
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate failure in http call
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_,
              Call(Status(Code::INTERNAL, "Failed to call service control"), _))
      .Times(1);

  async_callbacks_[0]->onFailure(Http::AsyncClient::FailureReason::Reset);
}

TEST_F(HttpCallTest, TestEmptyTokenCallFailure) {
  // If take_token is empty, on_done is called within call()
  EXPECT_CALL(mock_done_fn_,
              Call(Status(Code::INTERNAL,
                          "Missing access token for service control call"),
                   _))
      .Times(1);

  fake_token_.clear();
  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(0, async_callbacks_.size());
  EXPECT_EQ(0, http_requests_.size());
}

TEST_F(HttpCallTest, TestRetryCallSuccess) {
  // Set request to retry 2 more times
  retries_ = 2;
  http_call_factory_ = std::make_unique<HttpCallFactory>(
      cm_, dispatcher_, http_uri_, fake_suffix_url_, fake_token_fn_,
      timeout_ms_, retries_, mock_time_source_, fake_trace_operation_name_);
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span_1 = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate successful http response, but with a bad status code
  EXPECT_CALL(*mock_child_span_1, finishSpan()).Times(1);
  auto mock_child_span_2 = makeMockChildSpan();
  async_callbacks_[0]->onSuccess(makeResponseWithStatus(504));
  EXPECT_EQ(2, async_callbacks_.size());

  // Phase 3: Emulate another successful http response (on retry), but with a
  // bad status code
  EXPECT_CALL(*mock_child_span_2, finishSpan()).Times(1);
  auto mock_child_span_3 = makeMockChildSpan();
  async_callbacks_[1]->onSuccess(makeResponseWithStatus(503));
  EXPECT_EQ(3, async_callbacks_.size());

  // Phase 4: Emulate successful http response on last retry
  EXPECT_CALL(*mock_child_span_3, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_, Call(Status::OK, _)).Times(1);
  async_callbacks_[2]->onSuccess(makeResponseWithStatus(200));
}

TEST_F(HttpCallTest, TestThreeRetriesWithLastSuccess) {
  // Set request to retry 2 more times
  retries_ = 2;
  http_call_factory_ = std::make_unique<HttpCallFactory>(
      cm_, dispatcher_, http_uri_, fake_suffix_url_, fake_token_fn_,
      timeout_ms_, retries_, mock_time_source_, fake_trace_operation_name_);

  // Phase 1: Create HttpCall and send the request
  auto mock_child_span_1 = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());

  call->call();
  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate successful http response, but with a bad status code
  EXPECT_CALL(*mock_child_span_1, finishSpan()).Times(1);
  auto mock_child_span_2 = makeMockChildSpan();
  async_callbacks_[0]->onFailure(Http::AsyncClient::FailureReason::Reset);
  EXPECT_EQ(2, async_callbacks_.size());

  // Phase 3: Emulate another successful http response (on retry), but with a
  // bad status code
  EXPECT_CALL(*mock_child_span_2, finishSpan()).Times(1);
  auto mock_child_span_3 = makeMockChildSpan();
  async_callbacks_[1]->onFailure(Http::AsyncClient::FailureReason::Reset);
  EXPECT_EQ(3, async_callbacks_.size());

  // Phase 4: Emulate successful http response on last retry
  EXPECT_CALL(*mock_child_span_3, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_, Call(Status::OK, _)).Times(1);
  async_callbacks_[2]->onSuccess(makeResponseWithStatus(200));
}

TEST_F(HttpCallTest, TestThreeRetriesWithLastFailure) {
  // Set request to retry 2 more times
  retries_ = 2;
  http_call_factory_ = std::make_unique<HttpCallFactory>(
      cm_, dispatcher_, http_uri_, fake_suffix_url_, fake_token_fn_,
      timeout_ms_, retries_, mock_time_source_, fake_trace_operation_name_);

  // Phase 1: Create HttpCall and send the request
  auto mock_child_span_1 = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(0);  // Callback does not occur until response

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();
  EXPECT_EQ(1, async_callbacks_.size());

  // Phase 2: Emulate successful http response, but with a bad status code
  EXPECT_CALL(*mock_child_span_1, finishSpan()).Times(1);
  auto mock_child_span_2 = makeMockChildSpan();
  async_callbacks_[0]->onFailure(Http::AsyncClient::FailureReason::Reset);
  EXPECT_EQ(2, async_callbacks_.size());

  // Phase 3: Emulate another successful http response (on retry), but with a
  // bad status code
  EXPECT_CALL(*mock_child_span_2, finishSpan()).Times(1);
  auto mock_child_span_3 = makeMockChildSpan();
  async_callbacks_[1]->onFailure(Http::AsyncClient::FailureReason::Reset);
  EXPECT_EQ(3, async_callbacks_.size());

  // Phase 4: Emulate successful http response on last retry
  EXPECT_CALL(*mock_child_span_3, finishSpan()).Times(1);
  EXPECT_CALL(mock_done_fn_,
              Call(Status(Code::INTERNAL, "Failed to call service control"), _))
      .Times(1);
  async_callbacks_[2]->onSuccess(makeResponseWithStatus(504));
}

TEST_F(HttpCallTest, TestActiveCallCancel) {
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span = makeMockChildSpan();

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();

  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(1);  // Callback will still be called in cancel.

  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate destruct factory
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(1);
  EXPECT_CALL(*http_requests_[0], cancel()).Times(1);
  http_call_factory_.reset();
}

TEST_F(HttpCallTest, TestSingleCallCancel) {
  // Phase 1: Create HttpCall and send the request
  auto mock_child_span = makeMockChildSpan();
  EXPECT_CALL(mock_done_fn_, Call(_, _))
      .Times(1);  // Callback will still be called in cancel.

  HttpCall* call = http_call_factory_->createHttpCall(
      fake_request_, mock_parent_span_, mock_done_fn_.AsStdFunction());
  call->call();

  EXPECT_EQ(1, async_callbacks_.size());
  EXPECT_EQ(1, http_requests_.size());

  // Phase 2: Emulate cancellation
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(1);
  EXPECT_CALL(*http_requests_[0], cancel()).Times(1);
  call->cancel();

  // Phase 3: Check the cancelled calls not cancelled again
  EXPECT_CALL(*mock_child_span, finishSpan()).Times(0);
  EXPECT_CALL(*http_requests_[0], cancel()).Times(0);
  http_call_factory_.reset();
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
