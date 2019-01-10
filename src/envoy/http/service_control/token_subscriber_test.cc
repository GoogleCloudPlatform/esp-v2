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

#include "src/envoy/http/service_control/token_subscriber.h"
#include "common/tracing/http_tracer_impl.h"
#include "test/mocks/grpc/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

using Envoy::Server::Configuration::MockFactoryContext;
using ::google::api_proxy::agent::GetAccessTokenResponse;

using ::testing::_;
using ::testing::Invoke;

class MockTokenSubscriberCallback : public TokenSubscriber::Callback {
 public:
  MOCK_METHOD1(onTokenUpdate, void(const std::string& token));
};

class TokenSubscriberTest : public testing::Test {
 public:
  TokenSubscriberTest() {
    raw_mock_client_ = new Grpc::MockAsyncClient();
    raw_mock_client_factory_ = new Grpc::MockAsyncClientFactory();
    token_sub_.reset(new TokenSubscriber(
        context_, Grpc::AsyncClientFactoryPtr(raw_mock_client_factory_),
        token_callback_));

    EXPECT_CALL(*raw_mock_client_factory_, create())
        .WillOnce(Invoke([this]() -> Grpc::AsyncClientPtr {
          return Grpc::AsyncClientPtr(raw_mock_client_);
        }));

    EXPECT_CALL(*raw_mock_client_, send(_, _, _, _, _))
        .WillOnce(Invoke(
            [this](const Protobuf::MethodDescriptor&, const Protobuf::Message&,
                   Grpc::AsyncRequestCallbacks& callback, Tracing::Span&,
                   const absl::optional<std::chrono::milliseconds>&)
                -> Grpc::AsyncRequest* {
              client_callback_ = &callback;
              return nullptr;
            }));

    // InitManager should call this function.
    token_sub_->initialize([this]() { init_done_called_++; });
  }

  testing::NiceMock<MockFactoryContext> context_;
  MockTokenSubscriberCallback token_callback_;
  Grpc::AsyncRequestCallbacks* client_callback_{};
  Grpc::MockAsyncClient* raw_mock_client_{};
  Grpc::MockAsyncClientFactory* raw_mock_client_factory_{};
  int init_done_called_{};
  TokenSubscriberPtr token_sub_;
};

TEST_F(TokenSubscriberTest, TestSuccess) {
  EXPECT_CALL(token_callback_, onTokenUpdate(std::string("TOKEN")));

  // Send a Good token
  GetAccessTokenResponse* token_response = new GetAccessTokenResponse;
  token_response->set_access_token("TOKEN");
  token_response->mutable_expires_in()->set_seconds(100);
  client_callback_->onSuccessUntyped(ProtobufTypes::MessagePtr(token_response),
                                     Tracing::NullSpan::instance());

  EXPECT_EQ(init_done_called_, 1);
}

TEST_F(TokenSubscriberTest, TestFailure) {
  // Not called on failure.
  EXPECT_CALL(token_callback_, onTokenUpdate(_)).Times(0);

  // Send a Good token
  client_callback_->onFailure(Grpc::Status::GrpcStatus::Internal, "",
                              Tracing::NullSpan::instance());

  EXPECT_EQ(init_done_called_, 1);
}

TEST_F(TokenSubscriberTest, TestUpdate) {
  EXPECT_CALL(token_callback_, onTokenUpdate(std::string("TOKEN1")));

  auto* raw_mock_client1 = new Grpc::MockAsyncClient;
  EXPECT_CALL(*raw_mock_client_factory_, create())
      .WillOnce(Invoke([raw_mock_client1]() -> Grpc::AsyncClientPtr {
        return Grpc::AsyncClientPtr(raw_mock_client1);
      }));
  EXPECT_CALL(*raw_mock_client1, send(_, _, _, _, _)).Times(1);

  // Send a Good token1
  GetAccessTokenResponse* token_response = new GetAccessTokenResponse;
  token_response->set_access_token("TOKEN1");
  // Will refresh right away if less than 5s
  token_response->mutable_expires_in()->set_seconds(1);
  client_callback_->onSuccessUntyped(ProtobufTypes::MessagePtr(token_response),
                                     Tracing::NullSpan::instance());

  EXPECT_CALL(token_callback_, onTokenUpdate(std::string("TOKEN2")));

  token_response = new GetAccessTokenResponse;
  token_response->set_access_token("TOKEN2");
  token_response->mutable_expires_in()->set_seconds(100);
  client_callback_->onSuccessUntyped(ProtobufTypes::MessagePtr(token_response),
                                     Tracing::NullSpan::instance());

  EXPECT_EQ(init_done_called_, 1);
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
