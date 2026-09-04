// Copyright 2026 Google LLC
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

#include "src/envoy/http/trace_context/filter.h"

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/http/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/tracing/mocks.h"
#include "test/test_common/utility.h"

using ::testing::_;
using ::testing::NiceMock;
using ::testing::Return;
using ::testing::ReturnRef;
using ::testing::Invoke;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {
namespace {

class TraceContextFilterTest : public ::testing::Test {
 protected:
  void SetUp() override {
    ON_CALL(decoder_callbacks_, activeSpan())
        .WillByDefault(ReturnRef(active_span_));
    ON_CALL(active_span_, getTraceId())
        .WillByDefault(Return("0af7651916cd43dd8448eb211c80319c"));
    ON_CALL(active_span_, getSpanId())
        .WillByDefault(Return("b9c7c989f97918e1"));
  }

  void setupFilter(std::vector<::envoy::v12::http::trace_context::TraceContextFormat> formats) {
    auto config = std::make_shared<::envoy::v12::http::trace_context::TraceContextForwardedConfig>();
    for (auto format : formats) {
      config->add_outgoing_contexts(format);
    }
    filter_ = std::make_unique<Filter>(config);
    filter_->setDecoderFilterCallbacks(decoder_callbacks_);
  }

  std::unique_ptr<Filter> filter_;
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks> decoder_callbacks_;
  NiceMock<Envoy::Tracing::MockSpan> active_span_;
  Envoy::Http::TestRequestHeaderMapImpl headers_;
};

TEST_F(TraceContextFilterTest, CloudTraceContextFormat) {
  setupFilter({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  EXPECT_EQ(
      "0af7651916cd43dd8448eb211c80319c/13386890011815254241;o=1",
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()));
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get())
          .empty());
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get())
          .empty());
}

TEST_F(TraceContextFilterTest, GrpcTraceBinFormat) {
  setupFilter({::envoy::v12::http::trace_context::GRPC_TRACE_BIN});

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  const char kExpectedGrpcTraceBin[] = {
      '\x00', '\x00', '\x0a', '\xf7', '\x65', '\x19', '\x16', '\xcd',
      '\x43', '\xdd', '\x84', '\x48', '\xeb', '\x21', '\x1c', '\x80',
      '\x31', '\x9c', '\x01', '\xb9', '\xc7', '\xc9', '\x89', '\xf9',
      '\x79', '\x18', '\xe1', '\x02', '\x01'};
  absl::string_view expected_grpc_trace_bin(kExpectedGrpcTraceBin, 29);

  // Wire format is base64-encoded
  EXPECT_EQ(
      "AAAK92UZFs1D3YRI6yEcgDGcAbnHyYn5eRjhAgE=",
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()));
  std::string decoded_header;
  ASSERT_TRUE(absl::Base64Unescape(
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()),
      &decoded_header));
  EXPECT_EQ(expected_grpc_trace_bin, decoded_header);

  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get())
          .empty());
}

TEST_F(TraceContextFilterTest, TraceContextFormatPreservesEnvoyChildSpan) {
  setupFilter({::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // Envoy tracer wrote its child span
  headers_.addCopy(
      TraceContextHeadersSingleton::get().Traceparent,
      "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");
  // Stashed original client header
  headers_.addCopy(
      TraceContextHeadersSingleton::get().OriginalTraceparent,
      "00-clientraw123456789012345678901234-0000000000000002-01");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  // Child span preserved because TRACE_CONTEXT was configured
  EXPECT_EQ(
      "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01",
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get()));
  // Stash header scrubbed
  EXPECT_TRUE(headers_
                  .get_(TraceContextHeadersSingleton::get()
                            .OriginalTraceparent.get())
                  .empty());
}

TEST_F(TraceContextFilterTest,
       TransparentPassthroughOfStashedTraceparentWhenOutgoingDisabled) {
  // Outgoing tracing does not include TRACE_CONTEXT
  setupFilter({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // Envoy tracer added an unprompted child span
  headers_.addCopy(
      TraceContextHeadersSingleton::get().Traceparent,
      "00-envoytraceid999999999999999999999-envoychildspan1-01");
  // Ingress filter stashed the client's unconfigured raw traceparent
  headers_.addCopy(
      TraceContextHeadersSingleton::get().OriginalTraceparent,
      "00-clientraw123456789012345678901234-0000000000000002-01");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  // Unprompted Envoy child span is stripped and client's original traceparent is restored
  EXPECT_EQ(
      "00-clientraw123456789012345678901234-0000000000000002-01",
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get()));
  // Stash header scrubbed
  EXPECT_TRUE(headers_
                  .get_(TraceContextHeadersSingleton::get()
                            .OriginalTraceparent.get())
                  .empty());
  // Cloud trace header is generated
  EXPECT_EQ(
      "0af7651916cd43dd8448eb211c80319c/13386890011815254241;o=1",
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()));
}

TEST_F(TraceContextFilterTest, StripsTraceparentWhenOutgoingDisabledAndNoStashed) {
  setupFilter({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  // Envoy tracer added child span, but there was no stashed traceparent
  headers_.addCopy(
      TraceContextHeadersSingleton::get().Traceparent,
      "00-envoytraceid999999999999999999999-envoychildspan1-01");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  // Traceparent stripped completely
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get())
          .empty());
  EXPECT_TRUE(headers_
                  .get_(TraceContextHeadersSingleton::get()
                            .OriginalTraceparent.get())
                  .empty());
}

TEST_F(TraceContextFilterTest, PreservesLegacyHeadersWhenOutgoingDisabled) {
  // Outgoing configured only for TRACE_CONTEXT
  setupFilter({::envoy::v12::http::trace_context::TRACE_CONTEXT});

  // Client legacy headers present on the stream
  headers_.addCopy(TraceContextHeadersSingleton::get().CloudTraceContext,
                   "105445aa7843bc8bf206b12000100000/1;o=1");
  headers_.addCopy(TraceContextHeadersSingleton::get().GrpcTraceBin,
                   "dummy-grpc-trace-bin");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  // Legacy headers not touched by filter
  EXPECT_EQ(
      "105445aa7843bc8bf206b12000100000/1;o=1",
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()));
  EXPECT_EQ(
      "dummy-grpc-trace-bin",
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()));
}

TEST_F(TraceContextFilterTest, MultipleOutgoingContexts) {
  setupFilter({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT,
               ::envoy::v12::http::trace_context::GRPC_TRACE_BIN,
               ::envoy::v12::http::trace_context::TRACE_CONTEXT});

  headers_.addCopy(
      TraceContextHeadersSingleton::get().Traceparent,
      "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  EXPECT_EQ(
      "0af7651916cd43dd8448eb211c80319c/13386890011815254241;o=1",
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()));
  EXPECT_EQ(
      "AAAK92UZFs1D3YRI6yEcgDGcAbnHyYn5eRjhAgE=",
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()));
  EXPECT_EQ(
      "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01",
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get()));
}

TEST_F(TraceContextFilterTest, FormatEmpty) {
  setupFilter({});

  headers_.addCopy(
      TraceContextHeadersSingleton::get().Traceparent,
      "00-envoytraceid999999999999999999999-envoychildspan1-01");
  headers_.addCopy(
      TraceContextHeadersSingleton::get().OriginalTraceparent,
      "00-clientraw123456789012345678901234-0000000000000002-01");

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  // Restores stashed traceparent, does not create any legacy headers
  EXPECT_EQ(
      "00-clientraw123456789012345678901234-0000000000000002-01",
      headers_.get_(TraceContextHeadersSingleton::get().Traceparent.get()));
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get())
          .empty());
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get())
          .empty());
  EXPECT_TRUE(headers_
                  .get_(TraceContextHeadersSingleton::get()
                            .OriginalTraceparent.get())
                  .empty());
}

TEST_F(TraceContextFilterTest, FormatUnspecified) {
  setupFilter({::envoy::v12::http::trace_context::TRACE_CONTEXT_FORMAT_UNSPECIFIED});

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get())
          .empty());
  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get())
          .empty());
}

TEST_F(TraceContextFilterTest, ActiveSpanEmptyDoesNotCrash) {
  setupFilter({::envoy::v12::http::trace_context::CLOUD_TRACE_CONTEXT});

  EXPECT_CALL(active_span_, getTraceId()).WillRepeatedly(Return(""));
  EXPECT_CALL(active_span_, getSpanId()).WillRepeatedly(Return(""));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, false));

  EXPECT_TRUE(
      headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get())
          .empty());
}

TEST_F(TraceContextFilterTest, EncodeHeadersContinues) {
  setupFilter({});
  Envoy::Http::TestResponseHeaderMapImpl resp_headers;
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->encodeHeaders(resp_headers, false));
}

}  // namespace
}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
