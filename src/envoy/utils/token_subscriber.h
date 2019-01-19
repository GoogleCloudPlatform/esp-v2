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

#pragma once

#include "api/agent/agent_service.pb.h"
#include "common/common/logger.h"
#include "common/grpc/async_client_impl.h"
#include "envoy/common/pure.h"
#include "envoy/common/time.h"
#include "envoy/event/dispatcher.h"
#include "envoy/grpc/async_client_manager.h"
#include "envoy/init/init.h"
#include "envoy/server/filter_config.h"
#include "envoy/upstream/cluster_manager.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// This class fetches a token at the config time in the main thread.
// It also registers a timer to fetch a new token before expiration.
//
// It is using InitManager object. This is how InitManager works:
//
// * If your filter needs to make an async remote call, and needs to
//   wait for the response to continue the data flow, you need to
//   implement a Init::Target and register your Init::Target with InitManager.
//
// * InitManager calls each InitTarget initialize() function at the main thread.
//   Each target starts to make its remote call. This function passes in a
//   callback function which should be called when the remote call response is
//   received.
//
class TokenSubscriber
    : public Envoy::Init::Target,
      public Envoy::Grpc::TypedAsyncRequestCallbacks<
          ::google::api_proxy::agent::GetTokenResponse>,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::grpc> {
 public:
  class Callback {
   public:
    virtual ~Callback() {}
    virtual void onTokenUpdate(const std::string& token) PURE;
  };

  TokenSubscriber(Envoy::Server::Configuration::FactoryContext& context,
                  Envoy::Grpc::AsyncClientFactoryPtr client_factory,
                  Callback& callback, const std::string* audience);

  virtual ~TokenSubscriber();

  // Init::Target function
  void initialize(std::function<void()> callback) override;

  // Grpc::TypedAsyncRequestCallbacks functions
  void onCreateInitialMetadata(Envoy::Http::HeaderMap&) override {}
  void onSuccess(
      std::unique_ptr<::google::api_proxy::agent::GetTokenResponse>&& response,
      Envoy::Tracing::Span&) override;
  void onFailure(Envoy::Grpc::Status::GrpcStatus status,
                 const std::string& message, Envoy::Tracing::Span&) override;

 private:
  void runInitializeCallbackIfAny();
  void refresh();

  Envoy::Grpc::AsyncClientFactoryPtr client_factory_;
  Callback& token_callback_;

  std::function<void()> initialize_callback_;

  Envoy::Grpc::AsyncClientPtr async_client_;
  Envoy::Grpc::AsyncRequest* active_request_{};

  Envoy::Event::TimerPtr refresh_timer_;
  const std::string* audience_;
};
typedef std::unique_ptr<TokenSubscriber> TokenSubscriberPtr;

// Create Async Client Factory
Envoy::Grpc::AsyncClientFactoryPtr makeClientFactory(
    Envoy::Server::Configuration::FactoryContext& context,
    const std::string& token_cluster);

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
