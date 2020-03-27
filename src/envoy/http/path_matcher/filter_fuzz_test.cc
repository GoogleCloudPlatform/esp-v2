#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"

#include "src/envoy/http/path_matcher/filter.h"
#include "tests/fuzz/structured_inputs/path_matcher_filter.pb.validate.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {
namespace fuzz {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

void doTest(
    Filter& filter,
    const espv2::tests::fuzz::protos::PathMatcherFilterInput& input) {
  // Generate the user request.
  auto headers =
      Envoy::Fuzz::fromHeaders<Envoy::Http::TestRequestHeaderMapImpl>(
          input.downstream_request().headers());

  // Functions under test.
  filter.decodeHeaders(headers, false);
}

DEFINE_PROTO_FUZZER(
    const espv2::tests::fuzz::protos::PathMatcherFilterInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    Envoy::TestUtility::validate(input);

    if (input.config().rules_size() < 1) {
      throw Envoy::ProtoValidationException("At least 1 path matcher rule needed",
                                            input);
    }
  } catch (const Envoy::ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
    return;
  }

  // Setup mocks.
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks>
      mock_decoder_callbacks;
  NiceMock<Envoy::Server::Configuration::MockFactoryContext>
      mock_factory_context;

  try {
    // Create the filter.
    FilterConfigSharedPtr config = std::make_shared<FilterConfig>(
        input.config(), "fuzz-test-stats", mock_factory_context);
    Filter filter(config);
    filter.setDecoderFilterCallbacks(mock_decoder_callbacks);

    // Run data against the filter.
    ASSERT_NO_THROW(doTest(filter, input));

  } catch (const Envoy::EnvoyException& e) {
    ENVOY_LOG_MISC(debug, "Controlled envoy exception: {}", e.what());
  }
}

}  // namespace fuzz
}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2