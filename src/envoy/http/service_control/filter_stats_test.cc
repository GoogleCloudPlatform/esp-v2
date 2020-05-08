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

  void runTest(const std::vector<CodeToCounter>& mappings,
               const std::function<void(ServiceControlFilterStats&, Code&)>&
               collectStatus) {
    for (auto i : mappings) {
      // All counters are 0.
      for (auto j : mappings) {
        EXPECT_EQ(j.counter.value(), 0);
      }

      collectStatus(stats_, i.code);

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

TEST_F(FilterStatsTest, CollectCheckStatus
) {
std::vector<CodeToCounter> mappings = {
    {Code::OK, stats_.check_count_0_},
    {Code::CANCELLED, stats_.check_count_1_},
    {Code::UNKNOWN, stats_.check_count_2_},
    {Code::INVALID_ARGUMENT, stats_.check_count_3_},
    {Code::DEADLINE_EXCEEDED, stats_.check_count_4_},
    {Code::NOT_FOUND, stats_.check_count_5_},
    {Code::ALREADY_EXISTS, stats_.check_count_6_},
    {Code::PERMISSION_DENIED, stats_.check_count_7_},
    {Code::RESOURCE_EXHAUSTED, stats_.check_count_8_},
    {Code::FAILED_PRECONDITION, stats_.check_count_9_},
    {Code::ABORTED, stats_.check_count_10_},
    {Code::OUT_OF_RANGE, stats_.check_count_11_},
    {Code::UNIMPLEMENTED, stats_.check_count_12_},
    {Code::INTERNAL, stats_.check_count_13_},
    {Code::UNAVAILABLE, stats_.check_count_14_},
    {Code::DATA_LOSS, stats_.check_count_15_},
    {Code::UNAUTHENTICATED, stats_.check_count_16_}};

runTest(mappings, ServiceControlFilterStats::collectCheckStatus
);
}

TEST_F(FilterStatsTest, CollecReportStatus
) {
std::vector<CodeToCounter> mappings = {
    {Code::OK, stats_.report_count_0_},
    {Code::CANCELLED, stats_.report_count_1_},
    {Code::UNKNOWN, stats_.report_count_2_},
    {Code::INVALID_ARGUMENT, stats_.report_count_3_},
    {Code::DEADLINE_EXCEEDED, stats_.report_count_4_},
    {Code::NOT_FOUND, stats_.report_count_5_},
    {Code::ALREADY_EXISTS, stats_.report_count_6_},
    {Code::PERMISSION_DENIED, stats_.report_count_7_},
    {Code::RESOURCE_EXHAUSTED, stats_.report_count_8_},
    {Code::FAILED_PRECONDITION, stats_.report_count_9_},
    {Code::ABORTED, stats_.report_count_10_},
    {Code::OUT_OF_RANGE, stats_.report_count_11_},
    {Code::UNIMPLEMENTED, stats_.report_count_12_},
    {Code::INTERNAL, stats_.report_count_13_},
    {Code::UNAVAILABLE, stats_.report_count_14_},
    {Code::DATA_LOSS, stats_.report_count_15_},
    {Code::UNAUTHENTICATED, stats_.report_count_16_}};

runTest(mappings, ServiceControlFilterStats::collectReportStatus
);
}

}  // namespace
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2