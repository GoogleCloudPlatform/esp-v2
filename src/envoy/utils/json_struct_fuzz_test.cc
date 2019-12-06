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

#include "json_struct.h"
#include "common/common/base64.h"

namespace Envoy {
namespace Fuzz {

DEFINE_FUZZER(const uint8_t* buf, size_t len) {
  std::string input(reinterpret_cast<const char*>(buf), len);

  ::google::protobuf::util::JsonParseOptions options;
  ::google::protobuf::Struct response_pb;
  const auto parse_status = ::google::protobuf::util::JsonStringToMessage(
      input, &response_pb, options);

  if (!parse_status.ok()) {
    return;
  }
  Envoy::Extensions::Utils::JsonStruct json_struct(response_pb);
  auto* str_value = new std::string();
  auto* int_value = new std::string();

  (void)json_struct.get_string("key-1", str_value);
  (void)json_struct.get_string("key-2", str_value);
  (void)json_struct.get_string("key-1", int_value);
  (void)json_struct.get_string("key-2", int_value);

  delete str_value;
  delete int_value;
}

} // namespace Fuzz
} // namespace Envoy