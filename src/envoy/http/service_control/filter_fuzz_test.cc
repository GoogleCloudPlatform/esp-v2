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
  // Decode headers.
  bool end_stream = false;
  auto downstream_headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestHeaderMapImpl>(
          input.downstream_request().headers());
  if (input.downstream_request().data().size() == 0 &&
      !input.downstream_request().has_trailers()) {
    end_stream = true;
  }
  filter.decodeHeaders(downstream_headers, end_stream);

  // Decode body (if needed).
  for (int i = 0; i < input.downstream_request().data().size(); i++) {
    if (i == input.downstream_request().data().size() - 1 &&
        !input.downstream_request().has_trailers()) {
      end_stream = true;
    }
    Buffer::OwnedImpl buffer(input.downstream_request().data().Get(i));
    filter.decodeData(buffer, end_stream);
  }

  // Decode trailers (if needed).
  auto downstream_trailers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestTrailerMapImpl>(
          input.downstream_request().trailers());
  if (input.downstream_request().has_trailers()) {
    filter.decodeTrailers(downstream_trailers);
  }

  // Encode headers.
  end_stream = false;
  auto upstream_headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseHeaderMapImpl>(
          input.upstream_response().headers());
  if (input.upstream_response().data().size() == 0 &&
      !input.upstream_response().has_trailers()) {
    end_stream = true;
  }
  filter.encodeHeaders(upstream_headers, end_stream);

  // Encode body (if needed).
  for (int i = 0; i < input.upstream_response().data().size(); i++) {
    if (i == input.upstream_response().data().size() - 1 &&
        !input.upstream_response().has_trailers()) {
      end_stream = true;
    }
    Buffer::OwnedImpl buffer(input.upstream_response().data().Get(i));
    filter.encodeData(buffer, end_stream);
  }

  // Encode trailers (if needed).
  auto upstream_trailers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestResponseTrailerMapImpl>(
          input.upstream_response().trailers());
  if (input.upstream_response().has_trailers()) {
    filter.encodeTrailers(upstream_trailers);
  }

  // Access log (report).
  filter.log(&downstream_headers, &upstream_headers, &upstream_trailers,
             stream_info);
}

DEFINE_PROTO_FUZZER(
    const tests::fuzz::protos::ServiceControlFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    // Validate nested protos with stricter requirements for the fuzz test.
    // We need at least 1 requirement in the config to match a selector.
    if (input.config().requirements_size() < 1) {
      throw ProtoValidationException("Need at least 1 requirement", input);
    }
  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  // Setup mocks.
  NiceMock<MockFactoryContext> context;
  NiceMock<Http::MockAsyncClientRequest> request(
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
      .WillRepeatedly(Invoke([&request, &input, &resp_num](
                                 const Envoy::Http::RequestMessagePtr&,
                                 Envoy::Http::AsyncClient::Callbacks& callback,
                                 const Envoy::Http::AsyncClient::
                                     RequestOptions&) {
        // FIXME(nareddyt): For now, just increment the counter for
        // response numbers.
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
            std::make_unique<Envoy::Http::TestResponseTrailerMapImpl>(trailers);

        auto msg = std::make_unique<Envoy::Http::ResponseMessageImpl>(
            std::move(headers_ptr));
        msg->trailers(std::move(trailers_ptr));
        if (response_data.data_size() > 0) {
          // FIXME(nareddyt): For now, just grab 1 data item from the
          // proto.
          msg->body() =
              std::make_unique<Buffer::OwnedImpl>(response_data.data().Get(0));
        } else {
          msg->body() = std::make_unique<Buffer::OwnedImpl>();
        }

        // Callback.
        callback.onSuccess(std::move(msg));
        return &request;
      }));

  try {
    // Fuzz the stream info.
    TestStreamInfo stream_info =
        Envoy::Fuzz::fromStreamInfo(input.stream_info());
    EXPECT_CALL(mock_decoder_callbacks, streamInfo())
        .WillRepeatedly(ReturnRef(stream_info));
    EXPECT_CALL(mock_encoder_callbacks, streamInfo())
        .WillRepeatedly(ReturnRef(stream_info));

    // Create filter config.
    ServiceControlFilterConfig filter_config(input.config(), "fuzz-test-stats",
                                             context);

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
    ASSERT_NO_THROW(doTest(filter, stream_info, input));

  } catch (const EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
  }
}

}  // namespace Fuzz
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy