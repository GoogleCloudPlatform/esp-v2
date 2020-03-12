// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "test/fuzz/fuzz_runner.h"
#include "test/fuzz/utility.h"

#include "src/envoy/token/iam_token_info.h"
#include "tests/fuzz/structured_inputs/iam_token_info.pb.validate.h"

namespace Envoy {
namespace Extensions {
namespace Token {
namespace Test {

DEFINE_PROTO_FUZZER(const tests::fuzz::protos::IamTokenInfoInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    TestUtility::validate(input);

    Token::GetTokenFunc access_token_fn = [&input]() {
      return input.access_token();
    };

    IamTokenInfo token_info(input.delegates(), input.scopes(),
                            input.include_email(), access_token_fn);

    // Call functions under test.
    TokenResult ret;
    (void)token_info.prepareRequest(
        Envoy::Fuzz::replaceInvalidHostCharacters(input.token_url()));
    (void)token_info.parseAccessToken(input.resp_body(), &ret);
    (void)token_info.parseIdentityToken(input.resp_body(), &ret);

  } catch (const ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
  }
}

}  // namespace Test
}  // namespace Token
}  // namespace Extensions
}  // namespace Envoy