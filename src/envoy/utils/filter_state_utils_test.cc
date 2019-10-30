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

#include "src/envoy/utils/filter_state_utils.h"

#include "common/stream_info/filter_state_impl.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "test/test_common/utility.h"

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {

TEST(FilterStateUtilsTest, SetAndGetStringValueFromFilterState) {
  Envoy::StreamInfo::FilterStateImpl filter_state;

  setStringFilterState(filter_state, "data_name_foo", "foo");
  setStringFilterState(filter_state, "data_name_bar", "bar");

  EXPECT_EQ(getStringFilterState(filter_state, "data_name_foo"), "foo");
  EXPECT_EQ(getStringFilterState(filter_state, "data_name_bar"), "bar");
}

TEST(FilterStateUtilsTest, ReturnEmptyStringViewForNonExistingDataName) {
  Envoy::StreamInfo::FilterStateImpl filter_state;
  EXPECT_EQ(getStringFilterState(filter_state, "non_existing_data_name"), "");
}

}  // namespace
}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
