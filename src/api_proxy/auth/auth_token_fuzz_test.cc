#include "src/api_proxy/auth/auth_token.h"
#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"

#include "tests/fuzz/structured_inputs/auth_token.pb.validate.h"

namespace google {
namespace api_proxy {
namespace auth {
namespace fuzz {

DEFINE_PROTO_FUZZER(const tests::fuzz::protos::AuthTokenInput& input) {
  char* token = get_auth_token(input.secret().c_str(), input.audience().c_str());
  if (token != nullptr) {
    grpc_free(token);
  }
}

}  // namespace fuzz
}  // namespace auth
}  // namespace api_proxy
}  // namespace google