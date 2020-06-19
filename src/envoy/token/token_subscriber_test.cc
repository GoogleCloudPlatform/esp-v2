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

#include "src/envoy/token/token_subscriber.h"
#include "src/envoy/token/mocks.h"

#include "common/http/message_impl.h"
#include "common/tracing/http_tracer_impl.h"
#include "test/mocks/init/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "gmock/gmock-generated-function-mockers.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace espv2 {
namespace envoy {
namespace token {
namespace test {

using ::Envoy::Server::Configuration::MockFactoryContext;

using ::testing::_;
using ::testing::ByMove;
using ::testing::Invoke;
using ::testing::MockFunction;
using ::testing::Return;
using ::testing::ReturnRef;

constexpr std::chrono::milliseconds kFailedExpect(2000);

class TokenSubscriberTest : public testing::Test {
 protected:
  void SetUp() override {
    // Setup mock info.
    info_ = std::make_unique<NiceMock<MockTokenInfo>>();
    mock_timer_ = new Envoy::Event::MockTimer{};
  }

  void setUp(const TokenType& token_type) {
    EXPECT_CALL(context_.init_manager_, add(_))
        .WillOnce(Invoke([this](const Envoy::Init::Target& target) {
          init_target_handle_ = target.createHandle("test");
        }));

    // Setup mock http async client.
    EXPECT_CALL(context_.cluster_manager_.async_client_, send_(_, _, _))
        .WillRepeatedly(
            Invoke([this](Envoy::Http::RequestMessagePtr& message,
                          Envoy::Http::AsyncClient::Callbacks& callback,
                          const Envoy::Http::AsyncClient::RequestOptions&) {
              call_count_++;
              message_.swap(message);
              client_callback_ = &callback;
              return nullptr;
            }));

    // Setup mock refresh timer.
    EXPECT_CALL(context_.dispatcher_, createTimer_(_))
        .WillOnce(Invoke([this](Envoy::Event::TimerCb cb) {
          timer_cb_ = cb;
          return mock_timer_;
        }))
        .RetiresOnSaturation();

    // Create token subscriber under test.
    token_sub_ = std::make_unique<TokenSubscriber>(
        context_, token_type, "token_cluster", token_url_,
        std::chrono::seconds(5), token_callback_.AsStdFunction(),
        std::move(info_));
    token_sub_->init();

    // TokenSubscriber must call `ready` to signal Init::Manager once it
    // finishes initializing.
    EXPECT_CALL(init_watcher_, ready()).WillRepeatedly(Invoke([this]() {
      init_ready_ = true;
    }));

    // Init::Manager should initialize its targets.
    init_target_handle_->initialize(init_watcher_);
  }

  // Context
  NiceMock<MockFactoryContext> context_;
  Envoy::Http::MockAsyncClientRequest client_request_{
      &context_.cluster_manager_.async_client_};

  // Params to class under test.
  std::string token_url_ = "http://iam/uri_suffix";
  MockFunction<int(absl::string_view)> token_callback_;

  // Mocks for remote request.
  Envoy::Http::AsyncClient::Callbacks* client_callback_;
  Envoy::Http::RequestMessagePtr message_;
  int call_count_ = 0;

  // Mocks for init.
  Envoy::Init::TargetHandlePtr init_target_handle_;
  Envoy::Event::MockTimer* mock_timer_;
  Envoy::Event::TimerCb timer_cb_;
  NiceMock<Envoy::Init::ExpectableWatcherImpl> init_watcher_;
  bool init_ready_ = false;

  // Our classes.
  MockTokenInfoPtr info_;
  TokenSubscriberPtr token_sub_;
};

TEST_F(TokenSubscriberTest, HandleMissingPreconditions) {
  // Setup mocks for info.
  EXPECT_CALL(*info_, prepareRequest(_))
      .Times(1)
      .WillRepeatedly(Return(ByMove(nullptr)));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::IdentityToken);

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 0);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, VerifyRemoteRequest) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr headers(
      new Envoy::Http::TestRequestHeaderMapImpl(
          {{":method", "POST"}, {":authority", "TestValue"}}));
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(headers)))));

  // Start class under test.
  setUp(TokenType::IdentityToken);

  // Assert remote call matches.
  ASSERT_EQ(call_count_, 1);
  EXPECT_EQ(message_->headers().Method()->value().getStringView(), "POST");
  EXPECT_EQ(message_->headers().Host()->value().getStringView(), "TestValue");
}

TEST_F(TokenSubscriberTest, ProcessNon200Response) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::IdentityToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "504"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, ProcessMissingStatusResponse) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::IdentityToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl());
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, BadParseIdentityToken) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseIdentityToken(_, _)).WillOnce(Return(false));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::IdentityToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, BadParseAccessToken) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseAccessToken(_, _)).WillOnce(Return(false));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, TokenSantizationCheckFails) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseAccessToken(_, _))
      .WillOnce(Invoke([](absl::string_view, TokenResult* ret) {
        ret->token = "fake-token-with-bad\n-characters";
        ret->expiry_duration = std::chrono::seconds(30);
        return true;
      }));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, TokenPastExpiryCheckFails) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseAccessToken(_, _))
      .WillOnce(Invoke([](absl::string_view, TokenResult* ret) {
        ret->token = "fake-token";
        ret->expiry_duration = std::chrono::seconds(-1);
        return true;
      }));

  // Expect subscriber does not succeed.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(token_callback_, Call(_)).Times(0);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_FALSE(init_ready_);
}

TEST_F(TokenSubscriberTest, Success) {
  // Setup fake remote request.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(1)
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseAccessToken(_, _))
      .WillOnce(Invoke([](absl::string_view, TokenResult* ret) {
        ret->token = "fake-token";
        ret->expiry_duration = std::chrono::seconds(30);
        return true;
      }));

  // Expect subscriber does succeed.
  EXPECT_CALL(*mock_timer_,
              enableTimer(std::chrono::milliseconds(25 * 1000), nullptr))
      .Times(1);
  EXPECT_CALL(token_callback_, Call("fake-token")).Times(1);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_TRUE(init_ready_);
}

TEST_F(TokenSubscriberTest, RetryMissingPreconditionThenSuccess) {
  // Part 1: Failed due to missing precondition

  // Setup mocks for info. Fail on the first time, then work later.
  Envoy::Http::RequestHeaderMapPtr req_headers(
      new Envoy::Http::TestRequestHeaderMapImpl());
  EXPECT_CALL(*info_, prepareRequest(_))
      .WillOnce(Return(ByMove(nullptr)))
      .WillRepeatedly(
          Return(ByMove(std::make_unique<Envoy::Http::RequestMessageImpl>(
              std::move(req_headers)))));

  // Setup fake parse status.
  EXPECT_CALL(*info_, parseAccessToken(_, _))
      .WillOnce(Invoke([](absl::string_view, TokenResult* ret) {
        ret->token = "fake-token";
        ret->expiry_duration = std::chrono::seconds(30);
        return true;
      }));

  // Expect subscriber does not succeed at first, but then does.
  EXPECT_CALL(*mock_timer_, enableTimer(kFailedExpect, nullptr)).Times(1);
  EXPECT_CALL(*mock_timer_,
              enableTimer(std::chrono::milliseconds(25 * 1000), nullptr))
      .Times(1);
  EXPECT_CALL(token_callback_, Call("fake-token")).Times(1);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Assert subscriber did not succeed.
  ASSERT_EQ(call_count_, 0);
  ASSERT_FALSE(init_ready_);

  // Part 2: Retry the callback for the timer, will result in a success.
  timer_cb_();

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber did succeed.
  ASSERT_EQ(call_count_, 1);
  ASSERT_TRUE(init_ready_);
}

TEST_F(TokenSubscriberTest, RetryShortExpiryTime) {
  // Setup fake remote request.
  EXPECT_CALL(*info_, prepareRequest(token_url_))
      .Times(2)
      .WillRepeatedly(Invoke([](absl::string_view) {
        Envoy::Http::RequestHeaderMapPtr req_headers(
            new Envoy::Http::TestRequestHeaderMapImpl());
        return std::make_unique<Envoy::Http::RequestMessageImpl>(
            std::move(req_headers));
      }));

  // Setup fake parse status with a low expiry time.
  EXPECT_CALL(*info_, parseAccessToken(_, _))
      .WillOnce(Invoke([](absl::string_view, TokenResult* ret) {
        ret->token = "fake-token";
        // Expiry time is below the buffer, will cause a retry.
        ret->expiry_duration = std::chrono::seconds(1);
        return true;
      }));

  // Expect subscriber does succeed, but time was not set.
  EXPECT_CALL(*mock_timer_, enableTimer(_, _)).Times(0);
  EXPECT_CALL(token_callback_, Call("fake-token")).Times(1);

  // Start class under test.
  setUp(TokenType::AccessToken);

  // Setup fake response.
  Envoy::Http::ResponseHeaderMapPtr resp_headers(
      new Envoy::Http::TestResponseHeaderMapImpl({
          {":status", "200"},
      }));
  Envoy::Http::ResponseMessagePtr response(
      new Envoy::Http::ResponseMessageImpl(std::move(resp_headers)));

  // Start the response.
  client_callback_->onSuccess(client_request_, std::move(response));

  // Assert subscriber succeeded but did a retry.
  ASSERT_EQ(call_count_, 2);
  ASSERT_TRUE(init_ready_);
}

}  // namespace test
}  // namespace token
}  // namespace envoy
}  // namespace espv2
