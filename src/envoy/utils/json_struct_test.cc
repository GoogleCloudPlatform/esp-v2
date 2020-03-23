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

#include "src/envoy/utils/json_struct.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

using ::google::protobuf::util::error::Code;

namespace espv2 {
namespace envoy {
namespace utils {
namespace {

TEST(JsonStructTest, GetString) {
  ::google::protobuf::util::JsonParseOptions options;
  ::google::protobuf::Struct struct_pb;

  const std::string strings_struct = R"(
  {
    "good_string": "good",
    "empty_string": "",
    "bad_string": 28657
  }
  )";
  ASSERT_TRUE(::google::protobuf::util::JsonStringToMessage(strings_struct,
                                                            &struct_pb, options)
                  .ok());
  JsonStruct json_struct(struct_pb);

  // Test: Getting a string works
  std::string good_string;
  EXPECT_OK(json_struct.getString("good_string", &good_string));
  EXPECT_EQ(good_string, "good");

  // Test: Getting empty string works
  std::string empty_string;
  EXPECT_OK(json_struct.getString("empty_string", &empty_string));
  EXPECT_TRUE(empty_string.empty());

  // Test: Getting a string that is not a string type fails
  std::string bad_string;
  EXPECT_EQ(json_struct.getString("bad_string", &bad_string).code(),
            Code::INVALID_ARGUMENT);

  // Test: Getting a missing string fails
  std::string missing_string;
  EXPECT_EQ(json_struct.getString("missing_string", &missing_string).code(),
            Code::NOT_FOUND);
}

TEST(JsonStructTest, GetInt) {
  ::google::protobuf::util::JsonParseOptions options;
  ::google::protobuf::Struct struct_pb;

  const std::string strings_struct = R"(
  {
    "good_int": 377,
    "float_number": 1.57,
    "bad_int": "actually a string"
  }
  )";
  ASSERT_TRUE(::google::protobuf::util::JsonStringToMessage(strings_struct,
                                                            &struct_pb, options)
                  .ok());
  JsonStruct json_struct(struct_pb);

  // Test: Getting an integer works
  int good_int;
  EXPECT_OK(json_struct.getInteger("good_int", &good_int));
  EXPECT_EQ(good_int, 377);

  // Test: Getting an integer that is actually a float passes
  int float_to_int;
  EXPECT_OK(json_struct.getInteger("float_number", &float_to_int));
  EXPECT_EQ(float_to_int, 1);

  // Test: Getting an integer that is not a number type fails
  int bad_int;
  EXPECT_EQ(json_struct.getInteger("bad_int", &bad_int).code(),
            Code::INVALID_ARGUMENT);

  // Test: Getting a missing integer fails
  int missing_int;
  EXPECT_EQ(json_struct.getInteger("missing_int", &missing_int).code(),
            Code::NOT_FOUND);
}

}  // namespace
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
