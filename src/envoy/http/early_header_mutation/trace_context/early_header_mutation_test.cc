#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/factory_context.h"
#include "test/mocks/stream_info/mocks.h"
#include "gtest/gtest.h"
#include "gmock/gmock.h"
#include "absl/strings/escaping.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {
namespace {

using testing::Return;
using testing::_;
using testing::Invoke;

class TraceContextEarlyHeaderMutationTest : public testing::Test {
 protected:
  void setupConfig(std::vector<::envoy::v12::http::trace_context::TraceContextFormat> formats) {
    ::envoy::v12::http::trace_context::TraceContextTranslatorConfig config;
    for (auto format : formats) {
      config.add_incoming_contexts(format);
    }
    mutation_ = std::make_unique<TraceContextEarlyHeaderMutation>(
        std::make_shared<::envoy::v12::http::trace_context::TraceContextTranslatorConfig>(config));
  }

  std::unique_ptr<TraceContextEarlyHeaderMutation> mutation_;
  Envoy::Http::TestRequestHeaderMapImpl headers_;
  Envoy::StreamInfo::MockStreamInfo stream_info_;
};

TEST_F(TraceContextEarlyHeaderMutationTest, TranslatesXCloudTraceContextAndPreservesOriginal) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy header is preserved on ingress (not scrubbed)
  auto cloud_trace = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace.empty());
  EXPECT_EQ(cloud_trace[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, TranslatesGrpcTraceBinAndPreservesOriginal) {
  setupConfig({::envoy::v12::http::trace_context::GRPC_TRACE_BIN});

  std::string binary_header;
  binary_header.push_back(0); // version
  binary_header.push_back(0); // trace_id field
  for (int i = 0; i < 16; i++) binary_header.push_back('A');
  binary_header.push_back(1); // span_id field
  for (int i = 0; i < 8; i++) binary_header.push_back('B');
  binary_header.push_back(2); // trace_options field
  binary_header.push_back(1);
  
  std::string base64_header = absl::Base64Escape(binary_header);

  headers_.addCopy(Envoy::Http::LowerCaseString("grpc-trace-bin"), base64_header);

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy header is preserved on ingress (not scrubbed)
  auto grpc_trace = headers_.get(Envoy::Http::LowerCaseString("grpc-trace-bin"));
  ASSERT_FALSE(grpc_trace.empty());
  EXPECT_EQ(grpc_trace[0]->value().getStringView(), base64_header);
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-41414141414141414141414141414141-4242424242424242-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, PrecedenceW3C) {
  setupConfig({::envoy::v12::http::trace_context::TRACE_CONTEXT,
               ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // W3C traceparent exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy preserved
  auto cloud_trace = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace.empty());
  EXPECT_EQ(cloud_trace[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");
  // Original W3C is stashed
  auto stashed = headers_.get(Envoy::Http::LowerCaseString("x-espv2-original-traceparent"));
  ASSERT_FALSE(stashed.empty());
  EXPECT_EQ(stashed[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // W3C preserved and unmutated
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, PrecedenceOrderHonoredCloudTraceFirst) {
  // Simulates --tracing_incoming_context="x-cloud-trace-context,traceparent"
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT,
               ::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // W3C traceparent exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // Legacy exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy preserved
  auto cloud_trace = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace.empty());
  EXPECT_EQ(cloud_trace[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");
  // Original W3C is stashed
  auto stashed = headers_.get(Envoy::Http::LowerCaseString("x-espv2-original-traceparent"));
  ASSERT_FALSE(stashed.empty());
  EXPECT_EQ(stashed[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // CLOUD_TRACE_CONTEXT was listed first, so it won and was translated to traceparent
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, PrecedenceOrderHonoredGrpcTraceBinFirst) {
  // Simulates --tracing_incoming_context="grpc-trace-bin,traceparent"
  setupConfig({::envoy::v12::http::trace_context::GRPC_TRACE_BIN,
               ::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // W3C traceparent exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // gRPC trace bin exists BEFORE
  std::string binary_header;
  binary_header.push_back(0); // version
  binary_header.push_back(0); // trace_id field
  for (int i = 0; i < 16; i++) binary_header.push_back('A');
  binary_header.push_back(1); // span_id field
  for (int i = 0; i < 8; i++) binary_header.push_back('B');
  binary_header.push_back(2); // trace_options field
  binary_header.push_back(1);
  std::string base64_header = absl::Base64Escape(binary_header);
  headers_.addCopy(Envoy::Http::LowerCaseString("grpc-trace-bin"), base64_header);

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy preserved
  auto grpc_trace = headers_.get(Envoy::Http::LowerCaseString("grpc-trace-bin"));
  ASSERT_FALSE(grpc_trace.empty());
  EXPECT_EQ(grpc_trace[0]->value().getStringView(), base64_header);
  // Original W3C is stashed
  auto stashed = headers_.get(Envoy::Http::LowerCaseString("x-espv2-original-traceparent"));
  ASSERT_FALSE(stashed.empty());
  EXPECT_EQ(stashed[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // GRPC_TRACE_BIN was listed first, so it won and was translated to traceparent
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-41414141414141414141414141414141-4242424242424242-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, FallbackToSecondWhenFirstMissing) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT,
               ::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // Only W3C traceparent is provided
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, FallbackToSecondWhenFirstMalformed) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT,
               ::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // Malformed legacy header
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "invalid-trace-format");
  // Valid W3C traceparent
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Because x-cloud-trace-context was malformed, evaluation fell back to traceparent
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  // Malformed legacy header is also preserved
  auto cloud_trace = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace.empty());
  EXPECT_EQ(cloud_trace[0]->value().getStringView(), "invalid-trace-format");
}

TEST_F(TraceContextEarlyHeaderMutationTest, PrecedenceOrderHonoredBetweenLegacyFormats) {
  // Simulates --tracing_incoming_context="grpc-trace-bin,x-cloud-trace-context"
  setupConfig({::envoy::v12::http::trace_context::GRPC_TRACE_BIN,
               ::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // x-cloud-trace-context exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  // gRPC trace bin exists BEFORE
  std::string binary_header;
  binary_header.push_back(0); // version
  binary_header.push_back(0); // trace_id field
  for (int i = 0; i < 16; i++) binary_header.push_back('A');
  binary_header.push_back(1); // span_id field
  for (int i = 0; i < 8; i++) binary_header.push_back('B');
  binary_header.push_back(2); // trace_options field
  binary_header.push_back(1);
  std::string base64_header = absl::Base64Escape(binary_header);
  headers_.addCopy(Envoy::Http::LowerCaseString("grpc-trace-bin"), base64_header);

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Both legacy headers are preserved on ingress
  auto grpc_trace = headers_.get(Envoy::Http::LowerCaseString("grpc-trace-bin"));
  ASSERT_FALSE(grpc_trace.empty());
  EXPECT_EQ(grpc_trace[0]->value().getStringView(), base64_header);
  auto cloud_trace_header = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace_header.empty());
  EXPECT_EQ(cloud_trace_header[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");

  // GRPC_TRACE_BIN was listed first, so it won and was translated to traceparent
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-41414141414141414141414141414141-4242424242424242-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, ScrubsSpoofedStashHeaderWithoutIncomingTraceparent) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // Attacker supplies spoofed stash header without a traceparent
  headers_.addCopy(Envoy::Http::LowerCaseString("x-espv2-original-traceparent"),
                   "00-spoofed-traceparent");
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Spoofed stash header must be removed
  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("x-espv2-original-traceparent")).empty());
  // x-cloud-trace-context is preserved and translated to traceparent
  auto cloud_trace = headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"));
  ASSERT_FALSE(cloud_trace.empty());
  EXPECT_EQ(cloud_trace[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, NoConfigLeavesUnconfiguredHeadersAndStripsTraceparent) {
  setupConfig({});

  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // W3C traceparent is stripped because it was not configured as incoming format
  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("traceparent")).empty());

  // Unconfigured legacy header remains on the stream
  EXPECT_FALSE(headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context")).empty());
  EXPECT_EQ(headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context"))[0]->value().getStringView(),
            "105445aa7843bc8bf206b12000100000/1;o=1");
}

}  // namespace
}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
