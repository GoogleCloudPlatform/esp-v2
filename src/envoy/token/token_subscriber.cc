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

#include "absl/strings/str_cat.h"
#include "api/envoy/v10/http/common/base.pb.h"
#include "envoy/http/async_client.h"
#include "envoy/http/header_map.h"
#include "source/common/common/assert.h"
#include "source/common/common/enum_to_int.h"
#include "source/common/http/message_impl.h"
#include "source/common/http/utility.h"

namespace espv2 {
namespace envoy {
namespace token {

using ::espv2::api::envoy::v10::http::common::DependencyErrorBehavior;

// Delay after a failed fetch.
constexpr std::chrono::seconds kFailedRequestRetryTime(2);

// Update the token `n` seconds before the expiration.
constexpr std::chrono::seconds kRefreshBuffer(5);

TokenSubscriber::TokenSubscriber(
    Envoy::Server::Configuration::FactoryContext& context,
    const TokenType& token_type, const std::string& token_cluster,
    const std::string& token_url, std::chrono::seconds fetch_timeout,
    DependencyErrorBehavior error_behavior, UpdateTokenCallback callback,
    TokenInfoPtr token_info)
    : context_(context),
      token_type_(token_type),
      token_cluster_(token_cluster),
      token_url_(token_url),
      fetch_timeout_(fetch_timeout),
      error_behavior_(error_behavior),
      callback_(callback),
      token_info_(std::move(token_info)),
      active_request_(nullptr),
      init_target_(nullptr) {
  debug_name_ = absl::StrCat("TokenSubscriber(", token_url_, ")");
}

void TokenSubscriber::init() {
  init_target_ = std::make_unique<Envoy::Init::TargetImpl>(
      debug_name_, [this] { refresh(); });
  refresh_timer_ = context_.mainThreadDispatcher().createTimer(
      [this]() -> void { refresh(); });

  context_.initManager().add(*init_target_);
}

TokenSubscriber::~TokenSubscriber() {
  if (active_request_) {
    active_request_->cancel();
  }
}

void TokenSubscriber::handleFailResponse() {
  active_request_ = nullptr;
  refresh_timer_->enableTimer(kFailedRequestRetryTime);

  switch (error_behavior_) {
    case DependencyErrorBehavior::ALWAYS_INIT:
      ENVOY_LOG(debug,
                "{}: Response failed, but signalling ready due to "
                "DependencyErrorBehavior config.");
      init_target_->ready();
      break;
    default:
      break;
  }
}

void TokenSubscriber::handleSuccessResponse(absl::string_view token,
                                            std::chrono::seconds expires_in) {
  active_request_ = nullptr;

  // Signal that we are ready for initialization.
  ENVOY_LOG(debug, "{}: Got token and expiry duration: {} , {} seconds",
            debug_name_, token, expires_in.count());
  callback_(token);
  init_target_->ready();

  if (expires_in <= kRefreshBuffer) {
    // Handle low expiry time by retrying immediately.
    refresh();
  } else {
    refresh_timer_->enableTimer(expires_in - kRefreshBuffer);
  }
}

void TokenSubscriber::refresh() {
  if (active_request_) {
    active_request_->cancel();
  }

  ENVOY_LOG(debug, "{}: Sending TokenSubscriber request", debug_name_);

  Envoy::Http::RequestMessagePtr message =
      token_info_->prepareRequest(token_url_);
  if (message == nullptr) {
    // Preconditions in TokenInfo are not met, not an error.
    ENVOY_LOG(warn, "{}: preconditions not met, retrying later", debug_name_);
    handleFailResponse();
    return;
  }

  const struct Envoy::Http::AsyncClient::RequestOptions options =
      Envoy::Http::AsyncClient::RequestOptions()
          .setTimeout(std::chrono::duration_cast<std::chrono::milliseconds>(
              fetch_timeout_))
          // Metadata server rejects X-Forwarded-For requests.
          // https://cloud.google.com/compute/docs/storing-retrieving-metadata#x-forwarded-for_header
          .setSendXff(false);

  const auto thread_local_cluster =
      context_.clusterManager().getThreadLocalCluster(token_cluster_);
  if (thread_local_cluster) {
    active_request_ = thread_local_cluster->httpAsyncClient().send(
        std::move(message), *this, options);
  }
}

void TokenSubscriber::processResponse(
    Envoy::Http::ResponseMessagePtr&& response) {
  try {
    const uint64_t status_code =
        Envoy::Http::Utility::getResponseStatus(response->headers());

    if (status_code != Envoy::enumToInt(Envoy::Http::Code::OK)) {
      ENVOY_LOG(error, "{}: failed: {}", debug_name_, status_code);
      handleFailResponse();
      return;
    }
  } catch (const Envoy::EnvoyException& e) {
    // This occurs if the status header is missing.
    // Catch the exception to prevent unwinding and skipping cleanup.
    ENVOY_LOG(error, "{}: failed: {}", debug_name_, e.what());
    handleFailResponse();
    return;
  }

  // Delegate parsing the HTTP response.
  TokenResult result{};
  bool success;
  switch (token_type_) {
    case IdentityToken:
      success =
          token_info_->parseIdentityToken(response->bodyAsString(), &result);
      break;
    case AccessToken:
      success =
          token_info_->parseAccessToken(response->bodyAsString(), &result);
      break;
    default:
      NOT_REACHED_GCOVR_EXCL_LINE;
  }

  // Determine status.
  if (!success) {
    handleFailResponse();
    return;
  }

  // Token will be used as a HTTP_HEADER_VALUE in the future. Ensure it is
  // sanitized. Otherwise, special characters will cause a runtime failure
  // in other components.
  if (!Envoy::Http::validHeaderString(result.token)) {
    ENVOY_LOG(error,
              "{}: failed because invalid characters were detected in token {}",
              debug_name_, result.token);
    handleFailResponse();
    return;
  }

  // Tokens that have already expired are treated as failures.
  if (result.expiry_duration.count() <= 0) {
    ENVOY_LOG(error,
              "{}: failed because token has already expired, it expired {} "
              "seconds ago",
              debug_name_, result.expiry_duration.count());
    handleFailResponse();
    return;
  }

  handleSuccessResponse(result.token, result.expiry_duration);
}

void TokenSubscriber::onSuccess(const Envoy::Http::AsyncClient::Request&,
                                Envoy::Http::ResponseMessagePtr&& response) {
  ENVOY_LOG(debug, "{}: got response: {}", debug_name_,
            response->bodyAsString());
  processResponse(std::move(response));
}

void TokenSubscriber::onFailure(
    const Envoy::Http::AsyncClient::Request&,
    Envoy::Http::AsyncClient::FailureReason reason) {
  switch (reason) {
    case Envoy::Http::AsyncClient::FailureReason::Reset:
      ENVOY_LOG(error, "{}: failed with error: the stream has been reset",
                debug_name_);

      break;
    default:
      ENVOY_LOG(error, "{}: failed with an unknown network failure",
                debug_name_);
      break;
  }

  handleFailResponse();
}

}  // namespace token
}  // namespace envoy
}  // namespace espv2
