#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"

#include "src/envoy/http/backend_routing/filter.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "tests/fuzz/structured_inputs/backend_routing_filter.pb.validate.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendRouting {

void doTest(Filter& filter,
            const tests::fuzz::protos::BackendRoutingFilterInput& input) {
  // Generate the user request.
  auto headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestHeaderMapImpl>(
          input.user_request().headers());

  // Functions under test.
  filter.decodeHeaders(headers, false);
}

DEFINE_PROTO_FUZZER(
    const tests::fuzz::protos::BackendRoutingFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    if (input.config().rules_size() < 1) {
      throw ProtoValidationException("At least 1 backend rule needed", input);
    }
  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  // Setup mocks.
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks;
  NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context;

  // Set the operation name using the first backend routing rule.
  Utils::setStringFilterState(
      *mock_decoder_callbacks.stream_info_.filter_state_, Utils::kOperation,
      input.config().rules(0).operation());

  // Set the variable binding query params.
  Utils::setStringFilterState(
      *mock_decoder_callbacks.stream_info_.filter_state_, Utils::kQueryParams,
      input.binding_query_params());

  try {
    // Create the filter.
    FilterConfigSharedPtr config = std::make_shared<FilterConfig>(
        input.config(), "fuzz-test-stats", mock_factory_context);
    Filter filter(config);
    filter.setDecoderFilterCallbacks(mock_decoder_callbacks);

    // Run data against the filter.
    ASSERT_NO_THROW(doTest(filter, input));

  } catch (const EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
  }
}

}  // namespace BackendRouting
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
