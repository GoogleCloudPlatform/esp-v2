// Copyright 2019 Google LLC
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

#include "json_struct.h"
#include "tests/fuzz/structured_inputs/json_struct.pb.validate.h"

namespace espv2 {
namespace envoy {
namespace utils {
namespace fuzz {

// Needed for logger macro expansion.
namespace Logger = Envoy::Logger;

DEFINE_PROTO_FUZZER(const espv2::tests::fuzz::protos::JsonStructInput& input) {
  ENVOY_LOG_MISC(trace, "{}", input.DebugString());

  try {
    Envoy::TestUtility::validate(input);

    JsonStruct json_struct(input.pb_struct());

    for (const auto& key_to_check : input.keys_to_check()) {
      std::string str_value;
      (void)json_struct.getString(key_to_check, &str_value);

      int int_value;
      (void)json_struct.getInteger(key_to_check, &int_value);
    }
  } catch (const Envoy::ProtoValidationException& e) {
    ENVOY_LOG_MISC(debug, "Controlled proto validation failure: {}", e.what());
  }
}

}  // namespace fuzz
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
