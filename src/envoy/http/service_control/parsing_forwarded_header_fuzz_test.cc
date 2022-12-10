#include "src/envoy/http/service_control/handler_utils.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "tests/fuzz/structured_inputs/parsing_forwarded_header.pb.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace fuzz {

DEFINE_PROTO_FUZZER(
    const espv2::tests::fuzz::protos::ParsingForwardedHeaderInput& input) {
  Envoy::Http::TestRequestHeaderMapImpl headers;
  for (const auto& value : input.values()) {
    headers.addCopy("forwarded", value);
  }
  (void)extractIPFromForwardedHeader(headers);
}

}  // namespace fuzz
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
