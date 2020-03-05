#include "google/protobuf/text_format.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"

#include "api/envoy/http/service_control/config.pb.validate.h"
#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/filter_config.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "tests/fuzz/structured_inputs/service_control_filter.pb.validate.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include <fstream>
#include <stdexcept>
#include <string>

namespace filter_api = ::google::api::envoy::http::service_control;
namespace sc_api = ::google::api::servicecontrol::v1;
using ::Envoy::Server::Configuration::MockFactoryContext;
using ::testing::MockFunction;
using ::testing::Return;
using ::testing::ReturnRef;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace Fuzz {

void doTest(ServiceControlFilter& filter, TestStreamInfo& stream_info,
            const tests::fuzz::protos::ServiceControlFilterInput& input) {
  // Setup downstream request.
  auto downstream_headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestHeaderMapImpl>(
          input.downstream_request().headers());
  auto downstream_body =
      Buffer::OwnedImpl(input.downstream_request().data().Get(0));
  auto downstream_trailers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestTrailerMapImpl>(
          input.downstream_request().trailers());
  // TODO(b/146671523): Uncomment this when implemented upstream.
  // stream_info.addBytesReceived(downstream_body.length());

  // Downstream functions under test.
  filter.decodeHeaders(downstream_headers, false);
  filter.decodeData(downstream_body, false);
  filter.decodeData(downstream_body, true);
  filter.decodeTrailers(downstream_trailers);

  // Setup upstream response.
  auto upstream_headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseHeaderMapImpl>(
          input.upstream_response().headers());
  auto upstream_body =
      Buffer::OwnedImpl(input.upstream_response().data().Get(0));
  auto upstream_trailers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseTrailerMapImpl>(
          input.upstream_response().trailers());
  // TODO(b/146671523): Uncomment this when implemented upstream.
  // stream_info.addBytesSent(upstream_body.length());

  // Upstream functions under test.
  filter.encodeHeaders(upstream_headers, false);
  filter.encodeData(upstream_body, false);
  filter.encodeData(upstream_body, true);
  filter.encodeTrailers(upstream_trailers);

  // Report function under test.
  filter.log(&downstream_headers, &upstream_headers, nullptr, stream_info);
}

DEFINE_PROTO_FUZZER(
    const tests::fuzz::protos::ServiceControlFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    // Validate nested protos with stricter requirements for the fuzz test.
    // We only need 1 requirement in the config, others will just add noise.
    if (input.config().requirements_size() != 1) {
      throw ProtoValidationException("requirements", input);
    }
    // We only expect 1 buffer in the body to simplify setup.
    if (input.downstream_request().data().size() != 1) {
      throw ProtoValidationException("downstream data", input);
    }
    if (input.upstream_response().data().size() != 1) {
      throw ProtoValidationException("upstream data", input);
    }
    for (auto& sidestream_response : input.sidestream_response()) {
      if (sidestream_response.data().size() != 1) {
        throw ProtoValidationException("sidestream data", input);
      }
    }
    // There should be at least 1 sidestream response, otherwise no point.
    if (input.sidestream_response().size() < 1) {
      throw ProtoValidationException("num sidestream", input);
    }

    // Setup mocks.
    NiceMock<MockFactoryContext> context;
    NiceMock<Http::MockAsyncClientRequest> response(
        &context.cluster_manager_.async_client_);
    NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
        mock_decoder_callbacks;
    NiceMock<Envoy::Http::MockStreamEncoderFilterCallbacks>
        mock_encoder_callbacks;

    // Return a fake span.
    EXPECT_CALL(mock_decoder_callbacks, activeSpan())
        .WillRepeatedly(ReturnRef(Envoy::Tracing::NullSpan::instance()));

    // Callback for token subscriber to start.
    Envoy::Event::TimerCb onReadyCallback;
    EXPECT_CALL(context.dispatcher_, createTimer_(_))
        .WillRepeatedly(
            Invoke([&onReadyCallback](const Envoy::Event::TimerCb& cb) {
              ENVOY_LOG_MISC(trace, "Mocking dispatcher createTimer");
              onReadyCallback = cb;
              return new NiceMock<Event::MockTimer>();
            }));

    // Mock the http async client.
    int resp_num = 0;
    EXPECT_CALL(context.cluster_manager_.async_client_, send_(_, _, _))
        .WillRepeatedly(Invoke([&response, &input, &resp_num](
                                   const Envoy::Http::RequestMessagePtr&,
                                   Envoy::Http::AsyncClient::Callbacks&
                                       callback,
                                   const Envoy::Http::AsyncClient::
                                       RequestOptions&) {
          // FIXME(nareddyt): For now, just increment the counter for response
          // numbers.
          auto& response_data = input.sidestream_response().Get(
              resp_num++ % input.sidestream_response().size());

          // Create the response message.
          auto headers =
              Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseHeaderMapImpl>(
                  response_data.headers());
          auto headers_ptr =
              std::make_unique<Envoy::Http::TestResponseHeaderMapImpl>(headers);
          auto trailers =
              Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseTrailerMapImpl>(
                  response_data.trailers());
          auto trailers_ptr =
              std::make_unique<Envoy::Http::TestResponseTrailerMapImpl>(
                  trailers);

          auto msg = std::make_unique<Envoy::Http::ResponseMessageImpl>(
              std::move(headers_ptr));
          msg->trailers(std::move(trailers_ptr));
          msg->body() =
              std::make_unique<Buffer::OwnedImpl>(response_data.data().Get(0));

          // Callback.
          callback.onSuccess(std::move(msg));
          return &response;
        }));

    // Fuzz the stream info.
    TestStreamInfo stream_info =
        Envoy::Fuzz::fromStreamInfo(input.stream_info());
    EXPECT_CALL(mock_decoder_callbacks, streamInfo())
        .WillRepeatedly(ReturnRef(stream_info));
    EXPECT_CALL(mock_encoder_callbacks, streamInfo())
        .WillRepeatedly(ReturnRef(stream_info));

    try {
      // Create filter config.
      ServiceControlFilterConfig filter_config(input.config(),
                                               "fuzz-test-stats", context);

      // Set the operation name to match an endpoint that requires API keys
      // and has configured metric costs.
      // This ensures both CHECK and QUOTA are called.
      Utils::setStringFilterState(
          *stream_info.filter_state_, Utils::kOperation,
          input.config().requirements(0).operation_name());

      // Create filter.
      ServiceControlFilter filter(filter_config.stats(),
                                  filter_config.handler_factory());
      filter.setDecoderFilterCallbacks(mock_decoder_callbacks);
      filter.setEncoderFilterCallbacks(mock_encoder_callbacks);

      if (onReadyCallback != nullptr) {
        // Filter config is valid enough to start the token subscriber.
        onReadyCallback();
      }

      // Run data against the filter.
      doTest(filter, stream_info, input);

    } catch (const EnvoyException& e) {
      ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
    }

  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
  }
}

}  // namespace Fuzz
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy