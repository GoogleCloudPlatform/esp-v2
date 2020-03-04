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

DEFINE_PROTO_FUZZER(
    const tests::fuzz::protos::BackendRoutingFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    // This fuzz test only requires a single backend routing rule.
    // All other rules are ignored. So improve performance by only allowing
    // configs with one rule through.
    if (input.config().rules_size() != 1) {
      throw ProtoValidationException("Only 1 backend rule is allowed", input);
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

    // Create the filter.
    FilterConfigSharedPtr config = std::make_shared<FilterConfig>(
        input.config(), "fuzz-test-stats", mock_factory_context);
    Filter filter(config);
    filter.setDecoderFilterCallbacks(mock_decoder_callbacks);

    // Generate the user request.
    auto headers =
        Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestHeaderMapImpl>(
            input.user_request().headers());

    // Functions under test.
    filter.decodeHeaders(headers, false);

  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
  }
}

}  // namespace BackendRouting
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
