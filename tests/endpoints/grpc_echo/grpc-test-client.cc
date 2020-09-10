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

#include <iostream>

#include "google/protobuf/io/zero_copy_stream_impl.h"
#include "google/protobuf/text_format.h"
#include "tests/endpoints/grpc_echo/client-test-lib.h"
#include "tests/endpoints/grpc_echo/proto/grpc-test.grpc.pb.h"

using ::google::protobuf::Message;
using ::google::protobuf::TextFormat;
using ::google::protobuf::io::IstreamInputStream;
using ::google::protobuf::io::OstreamOutputStream;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::TypeResolver;
using ::google::protobuf::util::error::Code;

namespace {
std::string ReadInput(std::istream& src) {
  std::string contents;
  src.seekg(0, std::ios::end);
  auto size = src.tellg();
  if (size > 0) {
    contents.reserve(size);
  }
  src.seekg(0, std::ios::beg);
  contents.assign((std::istreambuf_iterator<char>(src)),
                  (std::istreambuf_iterator<char>()));
  return contents;
}
}  // namespace

int main(int argc, char** argv) {
  if (argc != 1) {
    std::cerr << "Usage: grpc-test-client" << std::endl;
    std::cerr << "Supply a text TestPlans proto on stdin to describe the tests."
              << std::endl;
    return EXIT_FAILURE;
  }

  ::test::grpc::TestPlans plans;
  ::test::grpc::TestResults results;
  std::cerr << "Parsing stdin" << std::endl;
  bool json_format = false;
  {
    std::string contents = ReadInput(std::cin);
    // Try Json
    if (JsonToProto(contents, &plans).ok()) {
      json_format = true;
    } else {
      // Try Text
      plans.Clear();
      if (!TextFormat::ParseFromString(contents, &plans)) {
        std::cerr << "Failed to parse text TestPlans proto from stdin:"
                  << contents << std::endl;
        return EXIT_FAILURE;
      }
    }
  }
  std::cerr << "Running tests" << std::endl;

  ::test::grpc::RunTestPlans(plans, &results);

  std::cerr << "Writing test outputs" << std::endl;

  {
    OstreamOutputStream out(&std::cout);
    if (json_format) {
      ProtoToJson(results, &out, test::grpc::PRETTY_PRINT);
    } else {
      TextFormat::Print(results, &out);
    }
  }

  int failed_count = 0;
  for (const auto& r : results.results()) {
    if (r.status().code() != ::grpc::OK) failed_count++;
  }
  std::cerr << "Exiting with failed_count: " << failed_count << std::endl;
  return failed_count == 0 ? EXIT_SUCCESS : EXIT_FAILURE;
}
