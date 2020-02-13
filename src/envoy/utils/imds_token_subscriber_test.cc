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

#include "src/envoy/utils/imds_token_subscriber.h"

#include "common/http/message_impl.h"
#include "common/tracing/http_tracer_impl.h"
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
using ::testing::Invoke;
using ::testing::MockFunction;
using ::testing::ReturnRef;

class ImdsTokenSubscriberTest : public testing::Test {
 protected:
  void SetUp() override {
    Init::TargetHandlePtr init_target_handle;
    EXPECT_CALL(context_.init_manager_, add(_))
        .WillOnce(Invoke([&init_target_handle](const Init::Target& target) {
          init_target_handle = target.createHandle("test");
        }));

    token_sub_.reset(new ImdsTokenSubscriber(
        context_, "fake_token_cluster", "http://fake_token_server/uri_suffix",
        true, token_callback_.AsStdFunction()));

    raw_mock_client_ =
        std::make_unique<NiceMock<Envoy::Http::MockAsyncClient>>();
    EXPECT_CALL(context_.cluster_manager_, httpAsyncClientForCluster(_))
        .WillRepeatedly(ReturnRef(*raw_mock_client_));
    EXPECT_CALL(*raw_mock_client_, send_(_, _, _))
        .WillRepeatedly(
            Invoke([this](Envoy::Http::MessagePtr& message,
                          Envoy::Http::AsyncClient::Callbacks& callback,
                          const Envoy::Http::AsyncClient::RequestOptions&) {
              call_count_++;
              message_.swap(message);
              client_callback_ = &callback;
              return nullptr;
            }));

    // ImdsTokenSubscriber must call `ready` to signal Init::Manager once it
    // finishes initializing.
    EXPECT_CALL(init_watcher_, ready());
    // Init::Manager should initialize its targets.
    init_target_handle->initialize(init_watcher_);
  }

  void checkRequestHeaders() {
    EXPECT_EQ(message_->headers()
                  .get(Envoy::Http::Headers::get().Method)
                  ->value()
                  .getStringView(),
              "GET");
    EXPECT_EQ(message_->headers()
                  .get(Envoy::Http::Headers::get().Host)
                  ->value()
                  .getStringView(),
              "fake_token_server");
    EXPECT_EQ(message_->headers()
                  .get(Envoy::Http::Headers::get().Path)
                  ->value()
                  .getStringView(),
              "/uri_suffix");
    EXPECT_EQ(
        message_->headers().get(kMetadataFlavorKey)->value().getStringView(),
        kMetadataFlavor);
  }

  int call_count_ = 0;

  NiceMock<Init::ExpectableWatcherImpl> init_watcher_;
  NiceMock<MockFactoryContext> context_;
  Envoy::Http::MessagePtr message_;
  MockFunction<int(std::string)> token_callback_;
  Envoy::Http::AsyncClient::Callbacks* client_callback_{};
  std::unique_ptr<NiceMock<Envoy::Http::MockAsyncClient>> raw_mock_client_;
  ImdsTokenSubscriberPtr token_sub_;
};

TEST_F(ImdsTokenSubscriberTest, CallOnTokenUpdateOnSuccess) {
  EXPECT_CALL(token_callback_, Call(std::string("TOKEN")));
  EXPECT_EQ(call_count_, 1);

  Envoy::Http::HeaderMapImplPtr headers{new Envoy::Http::TestHeaderMapImpl{
      {":status", "200"},
  }};

  // Send a good token
  Envoy::Http::MessagePtr response(
      new Envoy::Http::RequestMessageImpl(std::move(headers)));

  std::string str_body(R"({
    "access_token":"TOKEN",
    "expires_in":3598797
  })");
  response->body().reset(
      new Buffer::OwnedImpl(str_body.data(), str_body.size()));

  client_callback_->onSuccess(std::move(response));
  checkRequestHeaders();
}

TEST_F(ImdsTokenSubscriberTest, DoNotCallOnTokenUpdateOnFailure) {
  // Not called on failure.
  EXPECT_CALL(token_callback_, Call(_)).Times(0);
  EXPECT_EQ(call_count_, 1);

  // Send a bad token
  client_callback_->onFailure(Envoy::Http::AsyncClient::FailureReason::Reset);
  checkRequestHeaders();
}

TEST_F(ImdsTokenSubscriberTest, RefreshOnceTokenExpires) {
  // Send a good token `TOKEN1`
  EXPECT_EQ(call_count_, 1);
  EXPECT_CALL(token_callback_, Call(std::string("TOKEN1")));
  Envoy::Http::HeaderMapImplPtr headers1{new Envoy::Http::TestHeaderMapImpl{
      {":status", "200"},
  }};
  Envoy::Http::MessagePtr response1(
      new Envoy::Http::RequestMessageImpl(std::move(headers1)));
  std::string str_body1(R"({
    "access_token":"TOKEN1",
    "expires_in":1
  })");
  response1->body().reset(
      new Buffer::OwnedImpl(str_body1.data(), str_body1.size()));
  client_callback_->onSuccess(std::move(response1));

  // the onSuccess handler should immediately call refresh
  EXPECT_EQ(call_count_, 2);

  // Send a good token `TOKEN2`
  EXPECT_CALL(token_callback_, Call(std::string("TOKEN2")));
  Envoy::Http::HeaderMapImplPtr headers2{new Envoy::Http::TestHeaderMapImpl{
      {":status", "200"},
  }};
  Envoy::Http::MessagePtr response2(
      new Envoy::Http::RequestMessageImpl(std::move(headers2)));
  std::string str_body2(R"({
    "access_token":"TOKEN2",
    "expires_in":100
  })");
  response2->body().reset(
      new Buffer::OwnedImpl(str_body2.data(), str_body2.size()));
  client_callback_->onSuccess(std::move(response2));
  checkRequestHeaders();
}

}  // namespace
}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
