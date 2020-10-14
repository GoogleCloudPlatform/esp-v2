#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "src/envoy/http/path_matcher/filter.h"
#include "src/envoy/utils/filter_state_utils.h"
#include "test/extensions/filters/http/common/fuzz/uber_filter.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "tests/fuzz/structured_inputs/path_matcher_filter.pb.validate.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {
namespace fuzz {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

DEFINE_PROTO_FUZZER(
    const espv2::tests::fuzz::protos::PathMatcherFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    Envoy::TestUtility::validate(input);
  } catch (const Envoy::ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  if (input.config().rules_size() < 1) {
    ENVOY_LOG_MISC(debug, "Need at least one path matcher rule");
    return;
  }

  // Setup mocks.
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks;
  NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context;

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

  // Ensure the query param filter state is valid.
  absl::string_view query_params = utils::getStringFilterState(
      *mock_decoder_callbacks.stream_info_.filter_state_,
      utils::kFilterStateQueryParams);
  ASSERT_TRUE(Envoy::Http::validHeaderString(query_params));
}

}  // namespace fuzz
}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2