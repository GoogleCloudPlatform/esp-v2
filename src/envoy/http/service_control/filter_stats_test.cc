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

#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

#include "src/envoy/http/service_control/filter_stats.h"

using ::Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
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
        statBase_(ServiceControlFilterStatBase("", context_.scope_)),
        stats_(statBase_.stats()) {}

  NiceMock<MockFactoryContext> context_;
  ServiceControlFilterStatBase statBase_;
  ServiceControlFilterStats stats_;

  $ void runTest(
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
      {Code::OK, stats_.check_.OK_},
      {Code::CANCELLED, stats_.check_.CANCELLED_},
      {Code::UNKNOWN, stats_.check_.UNKNOWN_},
      {Code::INVALID_ARGUMENT, stats_.check_.INVALID_ARGUMENT_},
      {Code::DEADLINE_EXCEEDED, stats_.check_.DEADLINE_EXCEEDED_},
      {Code::NOT_FOUND, stats_.check_.NOT_FOUND_},
      {Code::ALREADY_EXISTS, stats_.check_.ALREADY_EXISTS_},
      {Code::PERMISSION_DENIED, stats_.check_.PERMISSION_DENIED_},
      {Code::RESOURCE_EXHAUSTED, stats_.check_.RESOURCE_EXHAUSTED_},
      {Code::FAILED_PRECONDITION, stats_.check_.FAILED_PRECONDITION_},
      {Code::ABORTED, stats_.check_.ABORTED_},
      {Code::OUT_OF_RANGE, stats_.check_.OUT_OF_RANGE_},
      {Code::UNIMPLEMENTED, stats_.check_.UNIMPLEMENTED_},
      {Code::INTERNAL, stats_.check_.INTERNAL_},
      {Code::UNAVAILABLE, stats_.check_.UNAVAILABLE_},
      {Code::DATA_LOSS, stats_.check_.DATA_LOSS_},
      {Code::UNAUTHENTICATED, stats_.check_.UNAUTHENTICATED_}};

  runTest(mappings, ServiceControlFilterStats::collectCallStatus);
}

}  // namespace
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2