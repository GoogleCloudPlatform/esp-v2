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

#pragma once

#include "common/common/logger.h"
#include "common/init/target_impl.h"
#include "envoy/common/time.h"
#include "envoy/event/dispatcher.h"
#include "envoy/http/message.h"
#include "envoy/server/filter_config.h"
#include "envoy/upstream/cluster_manager.h"
#include "src/envoy/token/token_info.h"

namespace espv2 {
namespace envoy {
namespace token {

enum TokenType { AccessToken, IdentityToken };

using UpdateTokenCallback = std::function<void(absl::string_view)>;

// `TokenSubscriber` class contains platform logic to initiate token refreshes
// and callback to the clients.
//
// It must be provided a `TokenInfo` adapter that knows how to parse the
// token response.
class TokenSubscriber
    : public Envoy::Http::AsyncClient::Callbacks,
      public Envoy::Logger::Loggable<Envoy::Logger::Id::init> {
 public:
  TokenSubscriber(Envoy::Server::Configuration::FactoryContext& context,
                  const TokenType& token_type, const std::string& token_cluster,
                  const std::string& token_url, UpdateTokenCallback callback,
                  TokenInfoPtr token_info);
  void init();

  ~TokenSubscriber();

 private:
  void handleFailResponse();
  void handleSuccessResponse(absl::string_view token,
                             const std::chrono::seconds& expires_in);
  void processResponse(Envoy::Http::ResponseMessagePtr&& response);
  void refresh();

  // Envoy::Http::AsyncClient::Callbacks implemented by this class.
  void onSuccess(const Envoy::Http::AsyncClient::Request& request,
                 Envoy::Http::ResponseMessagePtr&& response) override;
  void onFailure(const Envoy::Http::AsyncClient::Request& request,
                 Envoy::Http::AsyncClient::FailureReason reason) override;

  Envoy::Server::Configuration::FactoryContext& context_;
  const TokenType token_type_;
  const std::string token_cluster_;
  const std::string token_url_;
  const UpdateTokenCallback callback_;
  TokenInfoPtr token_info_;

  Envoy::Http::AsyncClient::Request* active_request_{};

  // This uses `Init::Manager` object. This is how `Init::Manager` works:
  //
  // * If your filter needs to make an async remote call, and needs to wait for
  //   the response to continue the data flow, you need register your
  //   `Init::Target` with `Init::Manager::add`.
  //
  // * `Init::Manager` initializes registered `Init::Target`s at the main
  // thread.
  //   Each target starts to make its remote call and signals `ready` to manager
  //   when it is initialized.
  Envoy::Event::TimerPtr refresh_timer_;
  std::unique_ptr<Envoy::Init::TargetImpl> init_target_;

  // Used in logs.
  std::string debug_name_;
};

using TokenSubscriberPtr = std::unique_ptr<TokenSubscriber>;

}  // namespace token
}  // namespace envoy
}  // namespace espv2
