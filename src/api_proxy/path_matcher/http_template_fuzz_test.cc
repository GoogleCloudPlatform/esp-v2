#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"
#include "src/api_proxy/path_matcher/http_template.h"

#include "tests/fuzz/structured_inputs/http_template.pb.validate.h"

namespace google {
namespace api_proxy {
namespace path_matcher {
namespace fuzz {
DEFINE_PROTO_FUZZER(
    const tests::fuzz::protos::HttpTemplateInput& input) {

  for (const auto& path : input.paths()) {
    HttpTemplate::Parse(path);
  }

}

}  //fuzz
}  // namespace path_matcher
}  // namespace api_proxy
}  // namespace google