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

#include "src/envoy/utils/http_header_utils.h"

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace utils {
namespace {

const Envoy::Http::LowerCaseString kHttpMethodOverrideHeader{
    "x-http-method-override"};

TEST(HttpHeaderUtilsTest, HttpMethodOverride) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "POST"}};
  headers.addCopy(kHttpMethodOverrideHeader, "GET");

  // Call function under test
  bool override = handleHttpMethodOverride(headers);

  // Expect the handler to modify the headers.
  EXPECT_TRUE(override);
  EXPECT_EQ(headers.Method()->value().getStringView(), "GET");
  EXPECT_TRUE(headers.get(kHttpMethodOverrideHeader).empty());
}

TEST(HttpHeaderUtilsTest, NoHttpMethodOverride) {
  Envoy::Http::TestRequestHeaderMapImpl headers{{":method", "POST"}};

  // Call function under test
  bool override = handleHttpMethodOverride(headers);

  // Expect the handler to be a NOOP.
  EXPECT_FALSE(override);
  EXPECT_EQ(headers.Method()->value().getStringView(), "POST");
}

}  // namespace
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
