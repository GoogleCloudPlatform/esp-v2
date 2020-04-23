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

#include "common/common/empty_string.h"
#include "common/protobuf/utility.h"
#include "common/stream_info/filter_state_impl.h"
#include "gmock/gmock.h"
#include "google/rpc/status.pb.h"
#include "gtest/gtest.h"
#include "test/test_common/utility.h"

namespace espv2 {
namespace envoy {
namespace utils {
namespace {

TEST(FilterStateUtilsTest, SetAndGetStringValueFromFilterState) {
  Envoy::StreamInfo::FilterStateImpl filter_state(
      Envoy::StreamInfo::FilterState::LifeSpan::FilterChain);

  setStringFilterState(filter_state, "data_name_foo", "foo");
  setStringFilterState(filter_state, "data_name_bar", "bar");

  EXPECT_EQ(getStringFilterState(filter_state, "data_name_foo"), "foo");
  EXPECT_EQ(getStringFilterState(filter_state, "data_name_bar"), "bar");
}

TEST(FilterStateUtilsTest, ReturnEmptyStringViewForNonExistingDataName) {
  Envoy::StreamInfo::FilterStateImpl filter_state(
      Envoy::StreamInfo::FilterState::LifeSpan::FilterChain);
  EXPECT_EQ(getStringFilterState(filter_state, "non_existing_data_name"),
            Envoy::EMPTY_STRING);
}

TEST(FilterStateUtilsTest, SetAndGetErrorFilterState) {
  Envoy::StreamInfo::FilterStateImpl filter_state(
      Envoy::StreamInfo::FilterState::LifeSpan::FilterChain);

  google::rpc::Status error;
  error.set_code(3);
  error.set_message("test-error-message");

  EXPECT_FALSE(hasErrorFilterState(filter_state));
  setErrorFilterState(filter_state, error);
  EXPECT_TRUE(hasErrorFilterState(filter_state));

  const google::rpc::Status& got = getErrorFilterState(filter_state);
  EXPECT_TRUE(Envoy::Protobuf::util::MessageDifferencer::Equals(got, error));
}

TEST(FilterStateUtilsTest, ErrorFilterStateIsCopiedWhenSet) {
  Envoy::StreamInfo::FilterStateImpl filter_state(
      Envoy::StreamInfo::FilterState::LifeSpan::FilterChain);

  google::rpc::Status error;
  error.set_code(3);

  setErrorFilterState(filter_state, error);

  error.set_code(0);

  const google::rpc::Status& got = getErrorFilterState(filter_state);
  EXPECT_NE(got.code(), error.code());
  EXPECT_EQ(got.code(), 3);
}

}  // namespace
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
