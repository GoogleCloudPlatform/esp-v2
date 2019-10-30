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
#include "envoy/event/dispatcher.h"
#include "envoy/server/filter_config.h"
#include "envoy/upstream/cluster_manager.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// The class generates an access_token with 1 hour expiration from a service
// account json for an audience and re-generating it before it is expired.
class ServiceAccountToken
    : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  using TokenUpdateFunc = std::function<void(const std::string& token)>;
  ServiceAccountToken(Envoy::Server::Configuration::FactoryContext& context,
                      const std::string& service_account_key,
                      const std::string& audience, TokenUpdateFunc callback);

 private:
  void refresh();

  const std::string& service_account_key_;
  const std::string audience_;

  TokenUpdateFunc callback_;
  Envoy::Event::TimerPtr refresh_timer_;
};
typedef std::unique_ptr<ServiceAccountToken> ServiceAccountTokenPtr;

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
