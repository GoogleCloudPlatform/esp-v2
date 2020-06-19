#include "test/extensions/filters/http/common/fuzz/uber_filter.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"

#include "src/envoy/http/backend_routing/filter.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "tests/fuzz/structured_inputs/backend_routing_filter.pb.validate.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {
namespace fuzz {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

DEFINE_PROTO_FUZZER(
    const espv2::tests::fuzz::protos::BackendRoutingFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    Envoy::TestUtility::validate(input);
  } catch (const Envoy::ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  if (input.config().rules_size() < 1) {
    ENVOY_LOG_MISC(debug, "Need at least one backend routing rule");
    return;
  }

  // Setup mocks.
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks;
  NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context;

  // Set the operation name using the first backend routing rule.
  utils::setStringFilterState(
      *mock_decoder_callbacks.stream_info_.filter_state_, utils::kOperation,
      input.config().rules(0).operation());

  // Set the variable binding query params.
  utils::setStringFilterState(
      *mock_decoder_callbacks.stream_info_.filter_state_, utils::kQueryParams,
      input.binding_query_params());

  // Create the filter.
  FilterConfigSharedPtr config;
  try {
    config = std::make_shared<FilterConfig>(input.config(), "fuzz-test-stats",
                                            mock_factory_context);
  } catch (const Envoy::EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
    return;
  }

  Filter filter(config);
  filter.setDecoderFilterCallbacks(mock_decoder_callbacks);

  // Run data against the filter.
  static Envoy::Extensions::HttpFilters::UberFilterFuzzer fuzzer;
  fuzzer.runData(static_cast<Envoy::Http::StreamDecoderFilter*>(&filter),
                 input.downstream_request());
  fuzzer.reset();
}

}  // namespace fuzz
}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2