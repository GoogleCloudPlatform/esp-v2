// Copyright 2019 Google LLC
//
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

#include "src/envoy/http/service_control/handler_utils.h"

#include "api/envoy/v10/http/service_control/config.pb.h"
#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "source/common/common/empty_string.h"
#include "src/api_proxy/service_control/request_builder.h"
#include "test/mocks/server/mocks.h"
#include "test/test_common/utility.h"

using ::espv2::api::envoy::v10::http::service_control::ApiKeyRequirement;
using ::espv2::api::envoy::v10::http::service_control::FilterConfig;
using ::espv2::api::envoy::v10::http::service_control::Service;
using ::espv2::api_proxy::service_control::LatencyInfo;
using ::espv2::api_proxy::service_control::ReportRequestInfo;
using ::espv2::api_proxy::service_control::protocol::Protocol;
using ::google::protobuf::TextFormat;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace {

TEST(ServiceControlUtils, FillGCPInfo) {
  struct TestCase {
    std::string filter_proto;
    std::string expected_zone;
    std::string expected_platform;
  };

  const TestCase test_cases[] = {
      // Test: No gcp_attributes found
      {Envoy::EMPTY_STRING, Envoy::EMPTY_STRING, "UNKNOWN(ESPv2)"},

      // Test: gcp_attributes found but empty
      {R"(gcp_attributes {})", Envoy::EMPTY_STRING, "UNKNOWN(ESPv2)"},

      // Test: bad platform provided should default to unknown
      {R"(gcp_attributes { platform: "bad-platform"})", Envoy::EMPTY_STRING,
       "bad-platform"},

      // Test: GAE_FLEX platform is passed through
      {R"(gcp_attributes { platform: "GAE_FLEX"})", Envoy::EMPTY_STRING,
       "GAE_FLEX"},

      // Test: GCE platform is set
      {R"(gcp_attributes { platform: "GCE"})", Envoy::EMPTY_STRING, "GCE"},

      // Test: GKE platform is set
      {R"(gcp_attributes { platform: "GKE"})", Envoy::EMPTY_STRING, "GKE"},

      // Test: Provided zone is set
      {R"(gcp_attributes { zone: "test-zone"})", "test-zone", "UNKNOWN(ESPv2)"},

      // Test: Provided platform and zone can both be set
      {R"(gcp_attributes { zone: "test-zone" platform: "GKE"})", "test-zone",
       "GKE"}};

  for (const auto& test : test_cases) {
    FilterConfig filter_config;
    ASSERT_TRUE(TextFormat::ParseFromString(test.filter_proto, &filter_config));
    ReportRequestInfo info;
    fillGCPInfo(filter_config, info);
    EXPECT_EQ(test.expected_zone, info.location);
    EXPECT_EQ(test.expected_platform, info.compute_platform);
  }
}

TEST(ServiceControlUtils, FillLoggedHeader) {
  // First test case: the function can accept null headers
  Service service;
  std::string output;
  fillLoggedHeader(nullptr, service.log_request_headers(), output);
  EXPECT_TRUE(output.empty());

  struct TestCase {
    Envoy::Http::TestRequestHeaderMapImpl headers;
    std::string service_proto;
    std::string expected_output;
  };
  const TestCase test_cases[] = {
      // Test: Search for the only header
      {
          {{"log-this", "foo"}},
          R"(log_request_headers: "log-this")",
          "log-this=foo;",
      },

      // Test: Desired header is not provided when other ones are
      {
          {{"ignore-this", "foo"}},
          R"(log_request_headers: "log-this")",
          Envoy::EMPTY_STRING,
      },

      // Test: Search for one header when there are others
      {
          {{"log-this", "foo"}, {"ignore-this", "bar"}},
          R"(log_request_headers: "log-this")",
          "log-this=foo;",
      },

      // Test: Multiple desired headers are logged in the order in the proto
      {
          {{"log-this", "foo"}, {"and-this", "bar"}},
          R"(log_request_headers: "log-this" log_request_headers: "and-this")",
          "log-this=foo;and-this=bar;",
      },

      // Test: Multiple desired headers are logged in the order in the proto
      // when the headers are provided in the opposite order
      {
          {{"log-this", "foo"}, {"and-this", "bar"}},
          R"(log_request_headers: "and-this" log_request_headers: "log-this")",
          "and-this=bar;log-this=foo;",
      },

      // Test: Intermix multiple desired headers with undesired ones
      {
          {{"log-this", "foo"}, {"and-this", "bar"}, {"ignore-this", "biz"}},
          R"(log_request_headers: "log-this" log_request_headers: "and-this")",
          "log-this=foo;and-this=bar;",
      },
  };

  for (const auto& test : test_cases) {
    Service service_tc;
    ASSERT_TRUE(TextFormat::ParseFromString(test.service_proto, &service_tc));

    std::string output_tc;

    fillLoggedHeader(&test.headers, service_tc.log_request_headers(),
                     output_tc);
    EXPECT_EQ(test.expected_output, output_tc);
  }

  // Test: The headers contain the logged header twice.
  // Both should be logged, but order is not consistent. Expect either.
  std::string service_proto = R"(log_request_headers: "log-this")";
  ASSERT_TRUE(TextFormat::ParseFromString(service_proto, &service));

  Envoy::Http::TestRequestHeaderMapImpl headers{{"log-this", "foo"},
                                                {"log-this", "bar"}};
  fillLoggedHeader(&headers, service.log_request_headers(), output);
  EXPECT_TRUE(output == "log-this=bar,foo;" || output == "log-this=foo,bar;");
}

TEST(ServiceControlUtils, ExtractApiKey) {
  struct TestCase {
    std::string requirement_proto;
    Envoy::Http::TestRequestHeaderMapImpl headers;
    std::string expected_api_key;
  };

  const TestCase test_cases[] = {
      // Test: No locations provided does nothing
      {Envoy::EMPTY_STRING, {}, Envoy::EMPTY_STRING},

      // Test: cookie location expected but not provided
      {
          R"(locations: { cookie: "apikey" } )",
          {},
          Envoy::EMPTY_STRING,
      },

      // Test: find apikey in cookie location
      {
          R"(locations: { cookie: "apikey" } )",
          {{"cookie", "apikey=foobar"}},
          "foobar",
      },

      // Test: find apikey in one of multiple cookie locations
      {
          R"(
            locations: { cookie: "apikey" }
            locations: { cookie: "apikey2" } )",
          {{"cookie", "apikey2=foobar"}},
          "foobar",
      },

      // Test: header location expected but not provided
      {
          R"(locations: { header: "apikey" } )",
          {},
          Envoy::EMPTY_STRING,
      },

      // Test: find apikey in header location
      {
          R"(locations: { header: "apikey" } )",
          {{"apikey", "foobar"}},
          "foobar",
      },

      // Test: find apikey in one of multiple header locations
      {
          R"(
            locations: { header: "apikey" }
            locations: { header: "apikey2" } )",
          {{"apikey2", "foobar"}},
          "foobar",
      },

      // Test: query location expected but not provided
      {
          R"(locations: { query: "apikey" } )",
          {{":path", "/echo"}},
          Envoy::EMPTY_STRING,
      },

      // Test: find apikey in query location
      {
          R"(locations: { query: "apikey" } )",
          {{":path", "/echo?apikey=foobar"}},
          "foobar",
      },

      // Test: find apikey in one of multiple query locations
      {
          R"(
            locations: { query: "apikey" }
            locations: { query: "apikey2" } )",
          {{":path", "/echo?apikey2=foobar"}},
          "foobar",
      },

      // Test: apikey is in cookie but cookie location is not expected
      {
          R"(
            locations: { header: "apikey" }
            locations: { query: "apikey" } )",
          {{"cookie", "apikey=foobar"}, {":path", "/echo"}},
          Envoy::EMPTY_STRING},

      // Test: apikey is in header but header location is not expected
      {
          R"(
            locations: { cookie: "apikey" }
            locations: { query: "apikey" } )",
          {{"apikey", "foobar"}, {":path", "/echo"}},
          Envoy::EMPTY_STRING},

      // Test: apikey is in query but query location is not expected
      {
          R"(
            locations: { cookie: "apikey" }
            locations: { header: "apikey" } )",
          {{":path", "/echo?apikey=foobar"}},
          Envoy::EMPTY_STRING,
      }};

  for (const auto& test : test_cases) {
    ApiKeyRequirement requirement;
    ASSERT_TRUE(
        TextFormat::ParseFromString(test.requirement_proto, &requirement));

    std::string api_key;

    EXPECT_EQ(!test.expected_api_key.empty(),
              extractAPIKey(test.headers, requirement.locations(), api_key));

    EXPECT_EQ(test.expected_api_key, api_key);
  }
}

TEST(ServiceControlUtils, FillLatency) {
  struct TestCase {
    std::chrono::nanoseconds end_time;
    std::chrono::nanoseconds first_upstream_tx_byte_sent;
    std::chrono::nanoseconds last_upstream_rx_byte_received;
    int expect_request_time_ms;
    int expect_backend_time_ms;
    int expect_overhead_time_ms;
  };

  const std::chrono::nanoseconds zero = std::chrono::nanoseconds(0);
  testing::NiceMock<Envoy::Stats::MockIsolatedStatsStore> mock_stats_scope;
  ServiceControlFilterStats stats(
      ServiceControlFilterStats::create(Envoy::EMPTY_STRING, mock_stats_scope));

  const std::vector<TestCase> test_cases = {
      // Test: If the stream has not ended, all stay their defaults.
      {zero, zero, zero, -1, 0, -1},

      // Test: If the stream has ended, request_time_ms should be set.
      {
          std::chrono::nanoseconds(1000000),  // end_time
          zero, zero,                         // first and last are not set
          1,                                  // request_time_ms set by end time
          0,                                  // default if start/end not set
          1,                                  // overhead time=request_time_ms
      },

      // Test: First and last bytes are provided, so backend_time is set
      {
          zero,                               // end is not set
          std::chrono::nanoseconds(2000000),  // first
          std::chrono::nanoseconds(5000000),  // last
          -1,                                 // default if end_time is not set
          3,                                  // backend = last - first
          -1,                                 // default if request_time not set
      },

      // Test: All three provided. request > backend
      {
          std::chrono::nanoseconds(5000000),  // end > last - first
          std::chrono::nanoseconds(2000000),  // first
          std::chrono::nanoseconds(5000000),  // last
          5,                                  // request = end time
          3,                                  // backend = last - first
          2,                                  // overhead = request - backend
      },

      // Test: All three provided. request = backend
      {
          std::chrono::nanoseconds(3000000),  // end = last - first
          std::chrono::nanoseconds(2000000),  // first
          std::chrono::nanoseconds(5000000),  // last
          3, 3,                               // request == backend
          0,                                  // overhead = request - backend
      },

      // Test: All three provided. request < backend
      {
          std::chrono::nanoseconds(2000000),  // end < last - first
          std::chrono::nanoseconds(2000000),  // first
          std::chrono::nanoseconds(5000000),  // last
          2,                                  // request = end time
          3,                                  // backend = last - first
          -1,                                 // default if request<backend
      },

      // Test: Realistic example of successful request.
      {
          std::chrono::nanoseconds(8000000),
          std::chrono::nanoseconds(2000000),
          std::chrono::nanoseconds(7000000),
          8,
          5,
          3,
      },

      // Test: Request times out on backend.
      // Note the overhead time is a little less, since we don't account
      // for the time to reject the request after backend timeout. Should
      // be minimal.
      {
          std::chrono::nanoseconds(8000000),
          std::chrono::nanoseconds(2000000),
          zero,
          8,
          6,
          2,
      },

      // Test: Filter rejects the request.
      {
          std::chrono::nanoseconds(8000000),
          zero,
          zero,
          8,
          0,
          8,
      },

  };

  for (unsigned long i = 0; i < test_cases.size(); i++) {
    const auto& test = test_cases[i];
    testing::NiceMock<Envoy::StreamInfo::MockStreamInfo> mock_stream_info;
    if (test.end_time > zero) {
      mock_stream_info.end_time_ = test.end_time;
    }
    if (test.first_upstream_tx_byte_sent > zero) {
      mock_stream_info.first_upstream_tx_byte_sent_ =
          test.first_upstream_tx_byte_sent;
    }
    if (test.last_upstream_rx_byte_received > zero) {
      mock_stream_info.last_upstream_rx_byte_received_ =
          test.last_upstream_rx_byte_received;
    }

    LatencyInfo info;
    fillLatency(mock_stream_info, info, stats);
    EXPECT_EQ(test.expect_request_time_ms, info.request_time_ms)
        << "Test case " << i;
    EXPECT_EQ(test.expect_backend_time_ms, info.backend_time_ms)
        << "Test case " << i;
    EXPECT_EQ(test.expect_overhead_time_ms, info.overhead_time_ms)
        << "Test case " << i;
  }
}

TEST(ServiceControlUtils, GetBackendProtocol) {
  Service service;

  // Test: no backend protocol defaults to UNKNOWN
  EXPECT_EQ(Protocol::UNKNOWN, getBackendProtocol(service));

  // Test: unidentified protocol defaults to UNKNOWN
  service.set_backend_protocol("bad-protocol");
  EXPECT_EQ(Protocol::UNKNOWN, getBackendProtocol(service));

  // Test: http1 protocol returns HTTP
  service.set_backend_protocol("http1");
  EXPECT_EQ(Protocol::HTTP, getBackendProtocol(service));

  // Test: http2 protocol returns HTTP
  service.set_backend_protocol("http2");
  EXPECT_EQ(Protocol::HTTP, getBackendProtocol(service));

  // Test: grpc protocol returns HTTP
  service.set_backend_protocol("grpc");
  EXPECT_EQ(Protocol::GRPC, getBackendProtocol(service));
}

TEST(ServiceControlUtils, GetFrontendProtocol) {
  Envoy::Http::TestResponseHeaderMapImpl headers;
  testing::NiceMock<Envoy::StreamInfo::MockStreamInfo> mock_stream_info;

  // Test: header is nullptr and stream_info has no protocol
  EXPECT_EQ(Protocol::UNKNOWN, getFrontendProtocol(nullptr, mock_stream_info));

  // Test: header has no content-type and stream_info has no protocol
  EXPECT_EQ(Protocol::UNKNOWN, getFrontendProtocol(&headers, mock_stream_info));

  // Test: header has a non-grpc content-type and stream_info has no protocol
  headers = {{"content-type", "application/json"}};
  EXPECT_EQ(Protocol::UNKNOWN, getFrontendProtocol(&headers, mock_stream_info));

  // Test: header has a grpc content-type
  headers = {{"content-type", "application/grpc"}};
  EXPECT_EQ(Protocol::GRPC, getFrontendProtocol(&headers, mock_stream_info));

  // Test: header has a grpc-web content-type
  // This tests all grpc types until the spec changes
  headers = {{"content-type", "application/grpc-web"}};
  EXPECT_EQ(Protocol::GRPC, getFrontendProtocol(&headers, mock_stream_info));

  // Test: header does not have a grpc content-type and stream_info has HTTP
  // protocol This tests all stream info protocols as they are all HTTP
  mock_stream_info.protocol_ = Envoy::Http::Protocol::Http10;
  EXPECT_EQ(Protocol::HTTP, getFrontendProtocol(nullptr, mock_stream_info));
}

}  // namespace
}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
