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

#include "src/api_proxy/tracing/trace_context_utils.h"

#include "gtest/gtest.h"

namespace espv2 {
namespace api_proxy {
namespace tracing {
namespace {

TEST(TraceContextUtilsTest, XCloudTraceContextToTraceParent) {
    // Valid conversion (sampled)
    auto tp = TraceContextUtils::XCloudTraceContextToTraceParent("105445aa7843bc8bf206b12000100000/1;o=1");
    ASSERT_TRUE(tp.has_value());
    EXPECT_EQ(tp.value(), "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");

    // Valid conversion (unsampled)
    tp = TraceContextUtils::XCloudTraceContextToTraceParent("105445aa7843bc8bf206b12000100000/123456789;o=0");
    ASSERT_TRUE(tp.has_value());
    EXPECT_EQ(tp.value(), "00-105445aa7843bc8bf206b12000100000-00000000075bcd15-00");

    // Invalid format
    tp = TraceContextUtils::XCloudTraceContextToTraceParent("invalid");
    EXPECT_FALSE(tp.has_value());
}

TEST(TraceContextUtilsTest, TraceParentToXCloudTraceContext) {
    // Valid conversion (sampled)
    auto xc = TraceContextUtils::TraceParentToXCloudTraceContext("00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
    ASSERT_TRUE(xc.has_value());
    EXPECT_EQ(xc.value(), "105445aa7843bc8bf206b12000100000/1;o=1");

    // Valid conversion (sampled via bitwise mask - e.g. trace flag '03' or 'FF')
    xc = TraceContextUtils::TraceParentToXCloudTraceContext("00-105445aa7843bc8bf206b12000100000-0000000000000001-03");
    ASSERT_TRUE(xc.has_value());
    EXPECT_EQ(xc.value(), "105445aa7843bc8bf206b12000100000/1;o=1");
    
    xc = TraceContextUtils::TraceParentToXCloudTraceContext("00-105445aa7843bc8bf206b12000100000-0000000000000001-ff");
    ASSERT_TRUE(xc.has_value());
    EXPECT_EQ(xc.value(), "105445aa7843bc8bf206b12000100000/1;o=1");

    // Valid conversion (unsampled)
    xc = TraceContextUtils::TraceParentToXCloudTraceContext("00-105445aa7843bc8bf206b12000100000-00000000075bcd15-00");
    ASSERT_TRUE(xc.has_value());
    EXPECT_EQ(xc.value(), "105445aa7843bc8bf206b12000100000/123456789;o=0");
    
    // Invalid format
    xc = TraceContextUtils::TraceParentToXCloudTraceContext("invalid");
    EXPECT_FALSE(xc.has_value());
}

TEST(TraceContextUtilsTest, GrpcTraceBinToTraceParent) {
    // Valid conversion (sampled)
    auto tp = TraceContextUtils::GrpcTraceBinToTraceParent("AAAQVEWqeEO8i/IGsSAAEAAAAQAAAAAAAAABAgE=");
    ASSERT_TRUE(tp.has_value());
    EXPECT_EQ(tp.value(), "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");

    // Valid conversion (unsampled)
    tp = TraceContextUtils::GrpcTraceBinToTraceParent("AAAQVEWqeEO8i/IGsSAAEAAAAQAAAAAHW80VAgA=");
    ASSERT_TRUE(tp.has_value());
    EXPECT_EQ(tp.value(), "00-105445aa7843bc8bf206b12000100000-00000000075bcd15-00");

    // Invalid format
    tp = TraceContextUtils::GrpcTraceBinToTraceParent("invalid!!base64");
    EXPECT_FALSE(tp.has_value());
}

TEST(TraceContextUtilsTest, TraceParentToGrpcTraceBin) {
    // Valid conversion (sampled)
    auto gb = TraceContextUtils::TraceParentToGrpcTraceBin("00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
    ASSERT_TRUE(gb.has_value());
    EXPECT_EQ(gb.value(), "AAAQVEWqeEO8i/IGsSAAEAAAAQAAAAAAAAABAgE=");

    // Valid conversion (sampled via bitwise mask - trace flag '03')
    gb = TraceContextUtils::TraceParentToGrpcTraceBin("00-105445aa7843bc8bf206b12000100000-0000000000000001-03");
    ASSERT_TRUE(gb.has_value());
    EXPECT_EQ(gb.value(), "AAAQVEWqeEO8i/IGsSAAEAAAAQAAAAAAAAABAgE="); // Will evaluate as sampled

    // Valid conversion (unsampled)
    gb = TraceContextUtils::TraceParentToGrpcTraceBin("00-105445aa7843bc8bf206b12000100000-00000000075bcd15-00");
    ASSERT_TRUE(gb.has_value());
    EXPECT_EQ(gb.value(), "AAAQVEWqeEO8i/IGsSAAEAAAAQAAAAAHW80VAgA=");
    
    // Invalid format
    gb = TraceContextUtils::TraceParentToGrpcTraceBin("invalid");
    EXPECT_FALSE(gb.has_value());
}

} // namespace
} // namespace tracing
} // namespace api_proxy
} // namespace espv2
