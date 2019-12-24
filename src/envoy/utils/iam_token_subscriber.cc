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

#include "src/envoy/utils/iam_token_subscriber.h"
#include "absl/strings/str_cat.h"
#include "common/common/enum_to_int.h"
#include "common/http/headers.h"
#include "common/http/message_impl.h"
#include "common/http/utility.h"
#include "src/envoy/utils/json_struct.h"

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {

// Required header when fetching from the iam server
const Envoy::Http::LowerCaseString kAuthorizationKey("Authorization");

// request timeout
const std::chrono::milliseconds kRequestTimeoutMs(5000);

// Delay after a failed fetch
const std::chrono::seconds kFailedRequestTimeout(60);

const std::chrono::milliseconds kAccessTokenWaitPeriod(10);

// If no expiration is provided in the response, refresh in this time.
const std::chrono::seconds kSubscriberDefaultTokenExpiry(3599);

Envoy::Http::MessagePtr prepareHeaders(const std::string& token_uri,
                                       const std::string& access_token) {
  absl::string_view host, path;
  Http::Utility::extractHostPathFromUri(token_uri, host, path);
  Envoy::Http::HeaderMapImplPtr headers{new Envoy::Http::HeaderMapImpl{
      {Envoy::Http::Headers::get().Method, "POST"},
      {Envoy::Http::Headers::get().Host, std::string(host)},
      {Envoy::Http::Headers::get().Path, std::string(path)},
      {kAuthorizationKey, "Bearer " + access_token}}};

  Envoy::Http::MessagePtr message(
      new Envoy::Http::RequestMessageImpl(std::move(headers)));
  return message;
}

}  // namespace

IamTokenSubscriber::IamTokenSubscriber(
    Envoy::Server::Configuration::FactoryContext& context,
    TokenGetFunc access_token_fn, const std::string& iam_service_cluster,
    const std::string& iam_service_uri, TokenUpdateFunc callback)
    : cm_(context.clusterManager()),
      access_token_fn_(access_token_fn),
      iam_service_cluster_(iam_service_cluster),
      iam_service_uri_(iam_service_uri),
      callback_(callback),
      active_request_(nullptr),
      init_target_("IamTokenSubscriber", [this] { refresh(); }) {
  refresh_timer_ =
      context.dispatcher().createTimer([this]() -> void { refresh(); });
  context.initManager().add(init_target_);
}

IamTokenSubscriber::~IamTokenSubscriber() {
  if (active_request_) {
    active_request_->cancel();
  }
}

void IamTokenSubscriber::refresh() {
  std::string access_token = access_token_fn_();

  // Wait the access token to be set.
  if (access_token.empty()) {
    // This codes depends on access_token. This periodical pulling is not ideal.
    // But when both imds_token_subscriber and iam_token_subscriber register to
    // init_manager,  it will trigger both at the same time. For
    // easy implementation,  just using periodical pulling for now
    ENVOY_LOG(debug, "sleep since access token is not ready");
    resetTimer(kAccessTokenWaitPeriod);
    return;
  }

  if (active_request_) {
    active_request_->cancel();
  }

  ENVOY_LOG(debug, "Sending getIdentityToken request");

  Envoy::Http::MessagePtr message =
      prepareHeaders(iam_service_uri_, access_token);

  const struct Envoy::Http::AsyncClient::RequestOptions options =
      Envoy::Http::AsyncClient::RequestOptions().setTimeout(kRequestTimeoutMs);

  active_request_ = cm_.httpAsyncClientForCluster(iam_service_cluster_)
                        .send(std::move(message), *this, options);
}

void IamTokenSubscriber::onSuccess(Envoy::Http::MessagePtr&& response) {
  ENVOY_LOG(debug, "GetAccessToken got response: {}", response->bodyAsString());
  active_request_ = nullptr;

  processResponse(std::move(response));
  init_target_.ready();
}

void IamTokenSubscriber::processResponse(Envoy::Http::MessagePtr&& response) {
  try {
    const uint64_t status_code =
        Envoy::Http::Utility::getResponseStatus(response->headers());
    if (status_code != enumToInt(Envoy::Http::Code::OK)) {
      ENVOY_LOG(error, "getIdentityToken is not 200 OK, got: {}", status_code);
      return;
    }
  } catch (const EnvoyException& e) {
    // This occurs if the status header is missing.
    // Catch the exception to prevent unwinding and skipping cleanup.
    ENVOY_LOG(error, "getIdentityToken failed: {}", e.what());
    return;
  }
  ENVOY_LOG(debug, "getIdentityToken success");

  // identity token response is a JSON payload
  /*
  {
    "token": "string",
  }
  */
  std::string token;
  ::google::protobuf::util::JsonParseOptions options;
  ::google::protobuf::Struct response_pb;
  ::google::protobuf::util::Status parse_status =
      ::google::protobuf::util::JsonStringToMessage(response->bodyAsString(),
                                                    &response_pb, options);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed: {}", parse_status.ToString());
    return;
  }

  JsonStruct json_struct(response_pb);
  parse_status = json_struct.getString("token", &token);
  if (!parse_status.ok()) {
    ENVOY_LOG(error, "Parsing response failed for field `token`: {}",
              parse_status.ToString());
    return;
  }

  // Use the default 1hr token expiry.
  resetTimer(kSubscriberDefaultTokenExpiry - std::chrono::seconds(5));

  ENVOY_LOG(debug, "Got identity token: {}", token);
  callback_(token);
}

void IamTokenSubscriber::onFailure(
    Envoy::Http::AsyncClient::FailureReason reason) {
  active_request_ = nullptr;
  ENVOY_LOG(error, "getIdentityToken failed with code: {}", enumToInt(reason));

  resetTimer(kFailedRequestTimeout);
  init_target_.ready();
}

void IamTokenSubscriber::resetTimer(const std::chrono::milliseconds& ms) {
  refresh_timer_->enableTimer(ms);
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
