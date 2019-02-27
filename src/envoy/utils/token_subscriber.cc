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

#include "src/envoy/utils/token_subscriber.h"
#include "common/grpc/async_client_manager_impl.h"

using ::google::api_proxy::agent::GetAccessTokenRequest;
using ::google::api_proxy::agent::GetIdentityJWTTokenRequest;
using ::google::api_proxy::agent::GetTokenResponse;

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {

// gRPC request timeout
const std::chrono::milliseconds kGrpcRequestTimeoutMs(5000);

// Delay after a failed fetch
const std::chrono::seconds kFailedRequestTimeout(60);

const google::protobuf::MethodDescriptor& getAccessTokenDescriptor() {
  static const google::protobuf::MethodDescriptor* descriptor =
      ::google::api_proxy::agent::AgentService::descriptor()->FindMethodByName(
          "GetAccessToken");
  ASSERT(descriptor);
  return *descriptor;
}

const google::protobuf::MethodDescriptor& getIdentityJWTTokenDescriptor() {
  static const google::protobuf::MethodDescriptor* descriptor =
      ::google::api_proxy::agent::AgentService::descriptor()->FindMethodByName(
          "GetIdentityJWTToken");
  ASSERT(descriptor);
  return *descriptor;
}

}  // namespace

Envoy::Grpc::AsyncClientFactoryPtr makeClientFactory(
    Envoy::Server::Configuration::FactoryContext& context,
    const std::string& token_cluster) {
  ::envoy::api::v2::core::GrpcService grpc_service;
  grpc_service.mutable_envoy_grpc()->set_cluster_name(token_cluster);
  return std::make_unique<Envoy::Grpc::AsyncClientFactoryImpl>(
      context.clusterManager(), grpc_service, true, context.timeSource());
}

TokenSubscriber::TokenSubscriber(
    Envoy::Server::Configuration::FactoryContext& context,
    Envoy::Grpc::AsyncClientFactoryPtr client_factory,
    TokenSubscriber::Callback& callback, const std::string* audience)
    : client_factory_(std::move(client_factory)),
      token_callback_(callback),
      audience_(audience) {
  refresh_timer_ =
      context.dispatcher().createTimer([this]() -> void { refresh(); });

  context.initManager().registerTarget(*this);
}

TokenSubscriber::~TokenSubscriber() {
  if (active_request_) {
    active_request_->cancel();
  }
}

void TokenSubscriber::initialize(std::function<void()> callback) {
  initialize_callback_ = callback;
  refresh();
}

void TokenSubscriber::refresh() {
  if (active_request_) {
    active_request_->cancel();
  }
  async_client_ = client_factory_->create();
  if (audience_ == nullptr) {
    ENVOY_LOG(debug, "Sending GetAccessToken request");
    GetAccessTokenRequest req;
    active_request_ = async_client_->send(
        getAccessTokenDescriptor(), req, *this,
        Envoy::Tracing::NullSpan::instance(),
        absl::optional<std::chrono::milliseconds>(kGrpcRequestTimeoutMs));
  } else {
    ENVOY_LOG(debug, "Sending GetIdentityJWTToken request");
    GetIdentityJWTTokenRequest req;
    req.set_audience(*audience_);
    active_request_ = async_client_->send(
        getIdentityJWTTokenDescriptor(), req, *this,
        Envoy::Tracing::NullSpan::instance(),
        absl::optional<std::chrono::milliseconds>(kGrpcRequestTimeoutMs));
  }
}

void TokenSubscriber::onSuccess(std::unique_ptr<GetTokenResponse>&& response,
                                Envoy::Tracing::Span&) {
  active_request_ = nullptr;
  ENVOY_LOG(debug, "GetAccessToken got response: {}", response->DebugString());
  token_callback_.onTokenUpdate(response->access_token());
  runInitializeCallbackIfAny();
  // Update the token 5 seconds before the expiration
  if (response->expires_in().seconds() <= 5) {
    refresh();
  } else {
    refresh_timer_->enableTimer(
        std::chrono::seconds(response->expires_in().seconds() - 5));
  }
}

void TokenSubscriber::onFailure(Envoy::Grpc::Status::GrpcStatus status,
                                const std::string& message,
                                Envoy::Tracing::Span&) {
  active_request_ = nullptr;
  ENVOY_LOG(debug, "GetAccessToken failed with code: {}, {}", status, message);
  runInitializeCallbackIfAny();
  refresh_timer_->enableTimer(kFailedRequestTimeout);
}

void TokenSubscriber::runInitializeCallbackIfAny() {
  if (initialize_callback_) {
    initialize_callback_();
    initialize_callback_ = nullptr;
  }
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
