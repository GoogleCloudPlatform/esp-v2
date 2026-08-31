#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation.h"
#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation_factory.cc"
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

TEST_F(TraceContextEarlyHeaderMutationTest, TranslatesXCloudTraceContext) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context")).empty());
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-105445aa7843bc8bf206b12000100000-0000000000000001-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, TranslatesGrpcTraceBin) {
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

  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("grpc-trace-bin")).empty());
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-41414141414141414141414141414141-4242424242424242-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, PrecedenceW3C) {
  setupConfig({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // W3C traceparent exists BEFORE
  headers_.addCopy(Envoy::Http::LowerCaseString("traceparent"),
                   "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));

  // Legacy deleted
  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context")).empty());
  // W3C preserved and unmutated
  auto traceparent = headers_.get(Envoy::Http::LowerCaseString("traceparent"));
  ASSERT_FALSE(traceparent.empty());
  EXPECT_EQ(traceparent[0]->value().getStringView(),
            "00-999999aa7843bc8bf206b12000100000-0000000000000002-01");
}

TEST_F(TraceContextEarlyHeaderMutationTest, NoConfigDoesNothingDeleteLegacy) {
  setupConfig({});

  headers_.addCopy(Envoy::Http::LowerCaseString("x-cloud-trace-context"),
                   "105445aa7843bc8bf206b12000100000/1;o=1");

  EXPECT_TRUE(mutation_->mutate(headers_, stream_info_));
  
  // W3C is empty because nothing was translated
  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("traceparent")).empty());
  
  // Notice: The deletion still occurs because we ALWAYS strip legacy headers!
  EXPECT_TRUE(headers_.get(Envoy::Http::LowerCaseString("x-cloud-trace-context")).empty());
}

}  // namespace
}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
