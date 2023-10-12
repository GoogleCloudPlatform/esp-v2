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

#include "tests/endpoints/grpc_echo/proto/grpc-test.grpc.pb.h"

namespace test {
namespace grpc {

// Options for JSON output. These should be OR'd together so must be 2^n.
enum JsonOptions {
  // Use the default behavior (useful when no options are needed).
      DEFAULT = 0,

  // Enables pretty printing of the output.
      PRETTY_PRINT = 1,

  // Prints default values for primitive fields.
      OUTPUT_DEFAULTS = 2,
};

// Converts a protobuf into a JSON string. The options field is a OR'd set of
// the available JsonOptions.
absl::Status JsonToProto(const std::string &json,
                                              ::google::protobuf::Message *message);

// Converts a protobuf into a JSON string and writes it into the output stream.
// The options parameter is an OR'd set of the available JsonOptions.
absl::Status ProtoToJson(const ::google::protobuf::Message& message,
                   ::google::protobuf::io::ZeroCopyOutputStream* json,
                   int options);

// Runs the supplied test plans, adding one result for each test.
void RunTestPlans(const TestPlans &plans, TestResults *results);
}  // namespace grpc

}  // namespace test
