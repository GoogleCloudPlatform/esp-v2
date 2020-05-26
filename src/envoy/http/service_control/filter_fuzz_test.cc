#include "google/protobuf/text_format.h"
#include "test/extensions/filters/http/common/fuzz/uber_filter.h"
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

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace fuzz {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

void doTest(
    ServiceControlFilter& filter, Envoy::TestStreamInfo&,
    const espv2::tests::fuzz::protos::ServiceControlFilterInput& input) {
  static Envoy::Extensions::HttpFilters::UberFilterFuzzer fuzzer;
  fuzzer.runData(static_cast<Envoy::Http::StreamDecoderFilter*>(&filter),
                 input.downstream_request());
  fuzzer.runData(static_cast<Envoy::Http::StreamEncoderFilter*>(&filter),
                 input.upstream_response());

  // TODO(nareddyt): Fuzz access log once #11288 is in upstream envoy.
}

DEFINE_PROTO_FUZZER(
    const espv2::tests::fuzz::protos::ServiceControlFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    Envoy::TestUtility::validate(input);
  } catch (const Envoy::ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  // Validate nested protos with stricter requirements for the fuzz test.
  // We need at least 1 requirement in the config to match a selector.
  if (input.config().requirements_size() < 1) {
    ENVOY_LOG_MISC(debug, "Need at least 1 requirement");
    return;
  }

  // Setup mocks.
  NiceMock<MockFactoryContext> context;
  NiceMock<Envoy::Http::MockAsyncClientRequest> request(
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
            return new NiceMock<Envoy::Event::MockTimer>();
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
        if (response_data.has_http_body() &&
            response_data.http_body().data_size() > 0) {
          // FIXME(nareddyt): For now, just grab 1 HTTP body data.
          msg->body() = std::make_unique<Envoy::Buffer::OwnedImpl>(
              response_data.http_body().data().Get(0));
        } else {
          msg->body() = std::make_unique<Envoy::Buffer::OwnedImpl>();
        }

        // Callback.
        callback.onSuccess(request, std::move(msg));
        return &request;
      }));

  try {
    // Fuzz the stream info.
    Envoy::TestStreamInfo stream_info =
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
    utils::setStringFilterState(
        *stream_info.filter_state_, utils::kOperation,
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

  } catch (const Envoy::EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
  }
}

}  // namespace fuzz
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
