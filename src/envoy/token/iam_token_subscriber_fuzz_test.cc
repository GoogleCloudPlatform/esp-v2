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

#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/init/mocks.h"
#include "test/mocks/server/mocks.h"

#include "src/envoy/token/iam_token_info.h"
#include "src/envoy/token/token_subscriber.h"
#include "tests/fuzz/structured_inputs/iam_token_subscriber.pb.validate.h"

#include "gmock/gmock.h"

namespace Envoy {
namespace Extensions {
namespace Token {
namespace Test {

using ::Envoy::Server::Configuration::MockFactoryContext;

using ::testing::MockFunction;

DEFINE_PROTO_FUZZER(const tests::fuzz::protos::IamTokenSubscriberInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    Envoy::Event::TimerCb onReadyCallback;

    // Setup mocks
    NiceMock<MockFactoryContext> context;
    NiceMock<Http::MockAsyncClientRequest> request(
        &context.cluster_manager_.async_client_);
    EXPECT_CALL(context.cluster_manager_.async_client_, send_(_, _, _))
        .WillRepeatedly(
            Invoke([&input, &request](
                       const Envoy::Http::MessagePtr&,
                       Envoy::Http::AsyncClient::Callbacks& callback,
                       const Envoy::Http::AsyncClient::RequestOptions&) {
              ENVOY_LOG_MISC(trace, "Mocking async client send");

              // Generate upstream response
              auto headers = Envoy::Fuzz::fromHeaders(
                  input.async_client_response().headers());
              auto headers_ptr =
                  std::make_unique<Envoy::Http::TestHeaderMapImpl>(headers);
              auto trailers = Envoy::Fuzz::fromHeaders(
                  input.async_client_response().trailers());
              auto trailers_ptr =
                  std::make_unique<Envoy::Http::TestHeaderMapImpl>(trailers);

              auto msg = std::make_unique<Envoy::Http::ResponseMessageImpl>(
                  std::move(headers_ptr));
              msg->trailers(std::move(trailers_ptr));
              msg->body() = std::make_unique<Buffer::OwnedImpl>(
                  input.async_client_response().data());

              // Callback
              callback.onSuccess(std::move(msg));
              return &request;
            }));

    EXPECT_CALL(context.dispatcher_, createTimer_(_))
        .WillRepeatedly(
            Invoke([&onReadyCallback](const Envoy::Event::TimerCb& cb) {
              ENVOY_LOG_MISC(trace, "Mocking dispatcher createTimer");
              onReadyCallback = cb;
              return new NiceMock<Event::MockTimer>();
            }));

    // Setup fakes
    Token::GetTokenFunc access_token_fn = [&input]() {
      return input.access_token();
    };
    Token::UpdateTokenCallback id_token_callback = [](absl::string_view) {};

    // Class under test
    ::google::protobuf::RepeatedPtrField<std::string> delegates;
    ::google::protobuf::RepeatedPtrField<std::string> scopes;
    TokenInfoPtr info = std::make_unique<IamTokenInfo>(delegates, scopes, true,
                                                       access_token_fn);
    TokenSubscriber subscriber(context, TokenType::IdentityToken,
                               "token_cluster", "http://iam/uri_suffix",
                               id_token_callback, std::move(info));
    subscriber.init();

    onReadyCallback();

  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
  }
}

}  // namespace Test
}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy