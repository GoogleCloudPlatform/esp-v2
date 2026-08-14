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
    config_ = std::make_shared<config_pb::TraceContextForwardedConfig>();
    filter_ = std::make_unique<Filter>(config_);
    filter_->setDecoderFilterCallbacks(decoder_callbacks_);
    
    ON_CALL(decoder_callbacks_, activeSpan()).WillByDefault(ReturnRef(active_span_));
  }

  void setupCloudTraceFormat() {
    config_->add_trace_context_format(config_pb::TraceContextConfig::CLOUD_TRACE_CONTEXT);
  }

  void setupGrpcTraceBinFormat() {
    config_->add_trace_context_format(config_pb::TraceContextConfig::GRPC_TRACE_BIN);
  }

  std::shared_ptr<config_pb::TraceContextForwardedConfig> config_;
  std::unique_ptr<Filter> filter_;
  NiceMock<Envoy::Http::MockStreamDecoderFilterCallbacks> decoder_callbacks_;
  NiceMock<Envoy::Tracing::MockSpan> active_span_;
  Envoy::Http::TestRequestHeaderMapImpl headers_;
};

TEST_F(TraceContextFilterTest, CloudTraceContextFormat) {
  setupCloudTraceFormat();
  
  EXPECT_CALL(active_span_, injectContext(_, _)).WillOnce(Invoke([](Envoy::Tracing::TraceContext& trace_context, const Envoy::Tracing::UpstreamContext&) {
    trace_context.setCopy(TraceContextHeadersSingleton::get().Traceparent, "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");
  }));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue, filter_->decodeHeaders(headers_, false));

  EXPECT_EQ("0af7651916cd43dd8448eb211c80319c/13386927375254264033;o=1", std::string(headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get())));
}

TEST_F(TraceContextFilterTest, GrpcTraceBinFormat) {
  setupGrpcTraceBinFormat();

  EXPECT_CALL(active_span_, injectContext(_, _)).WillOnce(Invoke([](Envoy::Tracing::TraceContext& trace_context, const Envoy::Tracing::UpstreamContext&) {
    trace_context.setCopy(TraceContextHeadersSingleton::get().Traceparent, "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");
  }));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue, filter_->decodeHeaders(headers_, false));

  const char kExpectedGrpcTraceBin[] = {
      '\x00', '\x00', '\x0a', '\xf7', '\x65', '\x19', '\x16', '\xcd',
      '\x43', '\xdd', '\x84', '\x48', '\xeb', '\x21', '\x1c', '\x80',
      '\x31', '\x9c', '\x01', '\xb9', '\xc7', '\xc9', '\x89', '\xf9',
      '\x79', '\x18', '\xe1', '\x02', '\x01'
  };
  absl::string_view expected_grpc_trace_bin(kExpectedGrpcTraceBin, 29);
  
  EXPECT_EQ(expected_grpc_trace_bin, headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()));
}

TEST_F(TraceContextFilterTest, FormatUnspecified) {
  config_->add_trace_context_format(config_pb::TraceContextConfig::TRACE_CONTEXT_FORMAT_UNSPECIFIED);

  EXPECT_CALL(active_span_, injectContext(_, _)).WillOnce(Invoke([](Envoy::Tracing::TraceContext& trace_context, const Envoy::Tracing::UpstreamContext&) {
    trace_context.setCopy(TraceContextHeadersSingleton::get().Traceparent, "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");
  }));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue, filter_->decodeHeaders(headers_, false));

  EXPECT_TRUE(headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()).empty());
  EXPECT_TRUE(headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()).empty());
}

TEST_F(TraceContextFilterTest, FormatEmpty) {
  // Empty configuration should just no-op
  EXPECT_CALL(active_span_, injectContext(_, _)).WillOnce(Invoke([](Envoy::Tracing::TraceContext& trace_context, const Envoy::Tracing::UpstreamContext&) {
    trace_context.setCopy(TraceContextHeadersSingleton::get().Traceparent, "00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01");
  }));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue, filter_->decodeHeaders(headers_, false));

  EXPECT_TRUE(headers_.get_(TraceContextHeadersSingleton::get().CloudTraceContext.get()).empty());
  EXPECT_TRUE(headers_.get_(TraceContextHeadersSingleton::get().GrpcTraceBin.get()).empty());
}

}  // namespace
}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
