#include <fstream>
#include <stdexcept>
#include <string>

#include "api/envoy/v9/http/service_control/config.pb.validate.h"
#include "common/http/message_impl.h"
#include "common/tracing/http_tracer_impl.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/filter_config.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/extensions/filters/http/common/fuzz/uber_filter.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "tests/fuzz/structured_inputs/service_control_filter.pb.validate.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace fuzz {

namespace filter_api = ::espv2::api::envoy::v9::http::service_control;
namespace sc_api = ::google::api::servicecontrol::v1;

using ::Envoy::TestStreamInfo;
using ::Envoy::Http::ResponseMessageImpl;
using ::Envoy::Server::Configuration::MockFactoryContext;
using ::testing::MockFunction;
using ::testing::Return;
using ::testing::ReturnRef;

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

void doTest(
    ServiceControlFilter& filter, Envoy::TestStreamInfo& stream_info,
    const espv2::tests::fuzz::protos::ServiceControlFilterInput& input) {
  static Envoy::Extensions::HttpFilters::UberFilterFuzzer fuzzer;
  fuzzer.runData(static_cast<Envoy::Http::StreamDecoderFilter*>(&filter),
                 input.downstream_request());
  fuzzer.accessLog(static_cast<Envoy::AccessLog::Instance*>(&filter),
                   stream_info);
  fuzzer.reset();
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
  NiceMock<Envoy::Upstream::MockThreadLocalCluster> thread_local_cluster;
  NiceMock<Envoy::Http::MockAsyncClientRequest> request(
      &context.cluster_manager_.thread_local_cluster_.async_client_);
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks;

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
  EXPECT_CALL(context.cluster_manager_, getThreadLocalCluster(_))
      .WillRepeatedly(Return(&thread_local_cluster));
  EXPECT_CALL(thread_local_cluster.async_client_, send_(_, _, _))
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

        auto msg =
            std::make_unique<ResponseMessageImpl>(std::move(headers_ptr));
        msg->trailers(std::move(trailers_ptr));
        if (response_data.has_http_body() &&
            response_data.http_body().data_size() > 0) {
          // FIXME(nareddyt): For now, just grab 1 HTTP body data.
          msg->body().add(response_data.http_body().data().Get(0));
        }

        // Callback.
        callback.onSuccess(request, std::move(msg));
        return &request;
      }));

  try {
    // Fuzz the stream info.
    std::unique_ptr<TestStreamInfo> stream_info =
        Envoy::Fuzz::fromStreamInfo(input.stream_info());
    EXPECT_CALL(mock_decoder_callbacks, streamInfo())
        .WillRepeatedly(ReturnRef(*stream_info));

    // Create filter config.
    ServiceControlFilterConfig filter_config(input.config(), "fuzz-test-stats",
                                             context);

    ::espv2::api::envoy::v9::http::service_control::PerRouteFilterConfig
        per_route_cfg;
    per_route_cfg.set_operation_name(
        input.config().requirements(0).operation_name());
    auto per_route = std::make_shared<PerRouteFilterConfig>(per_route_cfg);

    testing::NiceMock<Envoy::Router::MockRouteEntry> mock_route_entry;
    stream_info->route_entry_ = &mock_route_entry;
    EXPECT_CALL(mock_route_entry, perFilterConfig(kFilterName))
        .WillRepeatedly(
            Invoke([per_route](const std::string&)
                       -> const Envoy::Router::RouteSpecificFilterConfig* {
              return per_route.get();
            }));

    // Create filter.
    ServiceControlFilter filter(filter_config.stats(),
                                filter_config.handler_factory());
    filter.setDecoderFilterCallbacks(mock_decoder_callbacks);

    if (onReadyCallback != nullptr) {
      // Filter config is valid enough to start the token subscriber.
      onReadyCallback();
    }

    // Run data against the filter.
    ASSERT_NO_THROW(doTest(filter, *stream_info, input));

  } catch (const Envoy::EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
  }
}

}  // namespace fuzz
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
