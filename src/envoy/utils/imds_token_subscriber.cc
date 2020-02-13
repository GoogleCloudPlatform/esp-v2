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

// request timeout
const std::chrono::milliseconds kRequestTimeoutMs(5000);

// Delay after a failed fetch
const std::chrono::seconds kFailedRequestTimeout(60);

// If no expiration is provided in the response, refresh in this time.
const std::chrono::seconds kSubscriberDefaultTokenExpiry(3599);

Envoy::Http::MessagePtr prepareHeaders(const std::string& token_url) {
  absl::string_view host, path;
  Http::Utility::extractHostPathFromUri(token_url, host, path);

  Envoy::Http::HeaderMapImplPtr headers{new Envoy::Http::HeaderMapImpl{
      {Envoy::Http::Headers::get().Method, "GET"},
      {Envoy::Http::Headers::get().Host, std::string(host)},
      {Envoy::Http::Headers::get().Path, std::string(path)},
      {kMetadataFlavorKey, kMetadataFlavor}}};

  Envoy::Http::MessagePtr message(
      new Envoy::Http::RequestMessageImpl(std::move(headers)));

  return message;
}

}  // namespace

// Required header when fetching from the metadata server
const Envoy::Http::LowerCaseString kMetadataFlavorKey("Metadata-Flavor");
constexpr char kMetadataFlavor[]{"Google"};

ImdsTokenSubscriber::ImdsTokenSubscriber(
    Envoy::Server::Configuration::FactoryContext& context,
    const std::string& token_cluster, const std::string& token_url,
    const bool json_response, TokenUpdateFunc callback)
    : cm_(context.clusterManager()),
      token_cluster_(token_cluster),
      token_url_(token_url),
      json_response_(json_response),
      callback_(callback),
      active_request_(nullptr),
      init_target_("ImdsTokenSubscriber", [this] { refresh(); }) {
  refresh_timer_ =
      context.dispatcher().createTimer([this]() -> void { refresh(); });

  context.initManager().add(init_target_);
}

ImdsTokenSubscriber::~ImdsTokenSubscriber() {
  if (active_request_) {
    active_request_->cancel();
  }
}

void ImdsTokenSubscriber::refresh() {
  if (active_request_) {
    active_request_->cancel();
  }

  ENVOY_LOG(debug, "Sending GetAccessToken request");

  Envoy::Http::MessagePtr message = prepareHeaders(token_url_);

  const struct Envoy::Http::AsyncClient::RequestOptions options =
      Envoy::Http::AsyncClient::RequestOptions()
          .setTimeout(kRequestTimeoutMs)
          // Metadata server rejects X-Forwarded-For requests
          .setSendXff(false);

  active_request_ = cm_.httpAsyncClientForCluster(token_cluster_)
                        .send(std::move(message), *this, options);
}

void ImdsTokenSubscriber::onSuccess(Envoy::Http::MessagePtr&& response) {
  ENVOY_LOG(debug, "GetAccessToken got response: {}", response->bodyAsString());
  active_request_ = nullptr;

  processResponse(std::move(response));
  init_target_.ready();
}

void ImdsTokenSubscriber::processResponse(Envoy::Http::MessagePtr&& response) {
  const uint64_t status_code =
      Envoy::Http::Utility::getResponseStatus(response->headers());

  if (status_code == enumToInt(Envoy::Http::Code::OK)) {
    ENVOY_LOG(debug, "GetAccessToken success");
  } else {
    ENVOY_LOG(debug, "GetAccessToken failed: {}", status_code);
    return;
  }

  std::string token;
  std::chrono::seconds expires_in;

  if (json_response_) {
    // access token response is a JSON payload
    /*
    {
      "access_token": "string",
      "expires_in": uint
    }
    */
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

    parse_status = json_struct.getString("access_token", &token);
    if (!parse_status.ok()) {
      ENVOY_LOG(error, "Parsing response failed for field `access_token`: {}",
                parse_status.ToString());
      return;
    }

    int expires_seconds;
    parse_status = json_struct.getInteger("expires_in", &expires_seconds);
    if (!parse_status.ok()) {
      ENVOY_LOG(error, "Parsing response failed for field `expires_in`: {}",
                parse_status.ToString());
      return;
    }
    expires_in = std::chrono::seconds(expires_seconds);
  } else {
    // identity response is a string in the body
    token = response->bodyAsString();
    expires_in = kSubscriberDefaultTokenExpiry;
  }

  // Update the token 5 seconds before the expiration
  if (expires_in.count() <= 5) {
    refresh();
  } else {
    refresh_timer_->enableTimer(expires_in - std::chrono::seconds(5));
  }

  ENVOY_LOG(debug, "Got token: {}", token);
  callback_(token);
}

void ImdsTokenSubscriber::onFailure(
    Envoy::Http::AsyncClient::FailureReason reason) {
  init_target_.ready();
  active_request_ = nullptr;
  ENVOY_LOG(debug, "GetAccessToken failed with code: {}, {}",
            enumToInt(reason));
  refresh_timer_->enableTimer(kFailedRequestTimeout);
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
