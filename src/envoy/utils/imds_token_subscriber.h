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

#pragma once

#include "common/common/logger.h"
#include "common/init/target_impl.h"
#include "envoy/common/pure.h"
#include "envoy/common/time.h"
#include "envoy/event/dispatcher.h"
#include "envoy/http/async_client.h"
#include "envoy/http/message.h"
#include "envoy/server/filter_config.h"
#include "envoy/upstream/cluster_manager.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// Required header when fetching from the metadata server
extern const Envoy::Http::LowerCaseString kMetadataFlavorKey;
extern const char kMetadataFlavor[];

// `ImdsTokenSubscriber` class fetches a token at the config time in the main
// thread. It fetches it from the instance metadata server.
// It also registers a timer to fetch a new token before expiration.
//
// It uses `Init::Manager` object. This is how `Init::Manager` works:
//
// * If your filter needs to make an async remote call, and needs to wait for
//   the response to continue the data flow, you need register your
//   `Init::Target` with `Init::Manager::add`.
//
// * `Init::Manager` initializes registered `Init::Target`s at the main thread.
//   Each target starts to make its remote call and signals `ready` to manager
//   when it is initialized.
class ImdsTokenSubscriber
    : public Envoy::Http::AsyncClient::Callbacks,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  using TokenUpdateFunc = std::function<void(const std::string& token)>;
  // TODO(kyuc): Maybe add a name that gets passed to Init::TargetImpl.
  ImdsTokenSubscriber(Envoy::Server::Configuration::FactoryContext& context,
                      const std::string& token_cluster,
                      const std::string& token_url, const bool json_response,
                      TokenUpdateFunc callback);
  virtual ~ImdsTokenSubscriber();

 private:
  // Envoy::Http::AsyncClient::Callbacks
  void onSuccess(Envoy::Http::MessagePtr&& response) override;
  void onFailure(Envoy::Http::AsyncClient::FailureReason reason) override;

  void processResponse(Envoy::Http::MessagePtr&& response);
  void refresh();

  Upstream::ClusterManager& cm_;
  const std::string& token_cluster_;
  const std::string token_url_;
  const bool json_response_;

  TokenUpdateFunc callback_;
  Envoy::Http::AsyncClient::Request* active_request_{};

  Envoy::Event::TimerPtr refresh_timer_;
  // init_target_.ready() need be called at the end of request callbacks.
  Envoy::Init::TargetImpl init_target_;
};
typedef std::unique_ptr<ImdsTokenSubscriber> ImdsTokenSubscriberPtr;

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
