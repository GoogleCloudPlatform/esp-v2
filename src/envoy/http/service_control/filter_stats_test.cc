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

#include "src/envoy/http/service_control/filter_stats.h"

#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

using ::Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::util::OkStatus;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::StatusCode;
using ::testing::_;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace {
struct CodeToCounter {
  Code code;
  Envoy::Stats::Counter& counter;
};

class FilterStatsTest : public ::testing::Test {
 public:
  FilterStatsTest()
      : context_(),
        stats_(ServiceControlFilterStats::create("", context_.scope_)) {}

  NiceMock<MockFactoryContext> context_;
  ServiceControlFilterStats stats_;

  void runTest(
      const std::vector<CodeToCounter>& mappings,
      const std::function<void(CallStatusStats&, Code&)>& collectStatus) {
    for (auto i : mappings) {
      // All counters are 0.
      for (auto j : mappings) {
        EXPECT_EQ(j.counter.value(), 0);
      }

      collectStatus(stats_.check_, i.code);

      // Counter in i is 1 and all other counters are 0.
      for (auto j : mappings) {
        if (j.code == i.code) {
          EXPECT_EQ(j.counter.value(), 1);
          j.counter.reset();
        } else {
          EXPECT_EQ(j.counter.value(), 0);
        }
      }
    }
  }
};

TEST_F(FilterStatsTest, CollectCallStatus) {
  std::vector<CodeToCounter> mappings = {
      {StatusCode::kOk, stats_.check_.OK_},
      {StatusCode::kCancelled, stats_.check_.CANCELLED_},
      {StatusCode::kUnknown, stats_.check_.UNKNOWN_},
      {StatusCode::kInvalidArgument, stats_.check_.INVALID_ARGUMENT_},
      {StatusCode::kDeadlineExceeded, stats_.check_.DEADLINE_EXCEEDED_},
      {StatusCode::kNotFound, stats_.check_.NOT_FOUND_},
      {StatusCode::kAlreadyExists, stats_.check_.ALREADY_EXISTS_},
      {StatusCode::kPermissionDenied, stats_.check_.PERMISSION_DENIED_},
      {StatusCode::kResourceExhausted, stats_.check_.RESOURCE_EXHAUSTED_},
      {StatusCode::kFailedPrecondition, stats_.check_.FAILED_PRECONDITION_},
      {StatusCode::kAborted, stats_.check_.ABORTED_},
      {StatusCode::kOutOfRange, stats_.check_.OUT_OF_RANGE_},
      {StatusCode::kUnimplemented, stats_.check_.UNIMPLEMENTED_},
      {StatusCode::kInternal, stats_.check_.INTERNAL_},
      {StatusCode::kUnavailable, stats_.check_.UNAVAILABLE_},
      {StatusCode::kDataLoss, stats_.check_.DATA_LOSS_},
      {StatusCode::kUnauthenticated, stats_.check_.UNAUTHENTICATED_}};

  runTest(mappings, ServiceControlFilterStats::collectCallStatus);
}

}  // namespace
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
