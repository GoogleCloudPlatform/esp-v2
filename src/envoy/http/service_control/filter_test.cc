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

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/stats/mocks.h"
#include "test/mocks/tracing/mocks.h"
#include "test/test_common/utility.h"

#include "src/envoy/http/service_control/filter.h"
#include "src/envoy/http/service_control/handler.h"
#include "src/envoy/http/service_control/mocks.h"

using Envoy::Http::MockStreamDecoderFilterCallbacks;
using Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
using ::testing::_;
using ::testing::Invoke;
using ::testing::Return;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

const Status kBadStatus(Code::UNAUTHENTICATED, "test");

class ServiceControlFilterTest : public ::testing::Test {
 protected:
  ServiceControlFilterTest() : stats_base_("", mock_stats_scope_) {}

  void SetUp() override {
    filter_ = std::make_unique<ServiceControlFilter>(stats_base_.stats(),
                                                     mock_handler_factory_);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);

    mock_span_ = std::make_unique<Envoy::Tracing::MockSpan>();
  }

  std::unique_ptr<ServiceControlFilter> filter_;
  testing::NiceMock<MockStreamDecoderFilterCallbacks> mock_decoder_callbacks_;
  testing::NiceMock<MockFactoryContext> mock_factory_context_;
  testing::NiceMock<MockServiceControlHandlerFactory> mock_handler_factory_;
  testing::NiceMock<MockBuffer> mock_buffer_;
  testing::NiceMock<Stats::MockStore> mock_stats_scope_;
  ServiceControlFilterStatBase stats_base_;
  Http::TestHeaderMapImpl headers_;

  // Tracing mocks
  std::unique_ptr<Envoy::Tracing::MockSpan> mock_span_;
};

TEST_F(ServiceControlFilterTest, DecodeHeadersSyncOKStatus) {
  // Test: If onCall is called with OK status, return Continue
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));

  // Call onCheckDone synchronously
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(Invoke([](Http::HeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK);
      }));
  EXPECT_EQ(Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, true));

  // Verify handler->onDestroy is called when filter::onDestroy() is called
  EXPECT_CALL(*mock_handler, onDestroy()).Times(1);
  filter_->onDestroy();
}

TEST_F(ServiceControlFilterTest, OnDestoryWithoutHandler) {
  // Test: calling filter::onDestroy() without handler
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _)).Times(0);
  filter_->onDestroy();
}

TEST_F(ServiceControlFilterTest, DecodeHeadersSyncBadStatus) {
  // Test: If onCall is called with a bad status, reject and return stop
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));

  // Call onCheckDone synchronously
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(Invoke([](Http::HeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(kBadStatus);
      }));

  // TODO(toddbeckman) Figure out how to EXPECT_CALL sendLocalReply directly
  EXPECT_CALL(
      mock_decoder_callbacks_.stream_info_,
      setResponseFlag(StreamInfo::ResponseFlag::UnauthorizedExternalService));

  EXPECT_EQ(Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers_, true));
}

TEST_F(ServiceControlFilterTest, DecodeHeadersAsyncGoodStatus) {
  // Test: While Filter is Calling/stopped, onCheckDone calls
  // continueDecoding
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));

  ServiceControlHandler::CheckDoneCallback* stored_check_done_callback;

  // Store CheckDoneCallback but do not call it yet
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      // This invocation works around SaveArg storing the value to
      // the pointer in a way that does not work with the interface
      .WillOnce(Invoke([&stored_check_done_callback](
                           Http::HeaderMap&, Envoy::Tracing::Span&,
                           ServiceControlHandler::CheckDoneCallback& callback) {
        stored_check_done_callback = &callback;
      }));
  EXPECT_EQ(Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers_, true));

  EXPECT_CALL(mock_decoder_callbacks_, continueDecoding());
  stored_check_done_callback->onCheckDone(Status::OK);
}

TEST_F(ServiceControlFilterTest, DecodeHeadersAsyncBadStatus) {
  // Test: When status is not ok, the request is rejected and
  // continueDecoding is not called
  EXPECT_CALL(mock_decoder_callbacks_, continueDecoding()).Times(0);

  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));

  ServiceControlHandler::CheckDoneCallback* stored_check_done_callback;

  // Store CheckDoneCallback but do not call it yet
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(
          // This invocation works around SaveArg storing the value to
          // the pointer in a way that does not work with the interface
          Invoke([&stored_check_done_callback](
                     Http::HeaderMap&, Envoy::Tracing::Span&,
                     ServiceControlHandler::CheckDoneCallback& callback) {
            stored_check_done_callback = &callback;
          }));
  EXPECT_EQ(Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers_, true));

  // Filter should reject this request
  // TODO(toddbeckman) Figure out how to EXPECT_CALL sendLocalReply directly
  EXPECT_CALL(
      mock_decoder_callbacks_.stream_info_,
      setResponseFlag(StreamInfo::ResponseFlag::UnauthorizedExternalService));
  stored_check_done_callback->onCheckDone(kBadStatus);
}

TEST_F(ServiceControlFilterTest, LogWithoutHandlerOrHeaders) {
  // Test: If no handler and no headers, a handler is not created
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _)).Times(0);

  // Filter has no handler. If it tries to callReport, it will seg fault
  filter_->log(nullptr, &headers_, &headers_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, LogWithoutHandler) {
  // Test: When a Filter has no Handler yet, another is created for log()
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  EXPECT_CALL(*mock_handler, callReport(_, _, _, _));
  filter_->log(&headers_, &headers_, &headers_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, LogWithHandler) {
  // Test: When a Filter has a Handler from decodeHeaders,
  // that one is used for log() and another is not created
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  filter_->decodeHeaders(headers_, true);

  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _)).Times(0);
  EXPECT_CALL(*mock_handler, callReport(_, _, _, _));
  filter_->log(&headers_, &headers_, &headers_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, DecodeHelpersWhileStopped) {
  // This puts the Filter into a stopped state
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  EXPECT_EQ(Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(headers_, true));

  // Test: While Filter is Calling/stopped, decodeData returns Stop
  EXPECT_EQ(Http::FilterDataStatus::StopIterationAndWatermark,
            filter_->decodeData(mock_buffer_, true));

  // Test: While Filter is Calling/stopped, decodeTrailers returns Stop
  EXPECT_EQ(Http::FilterTrailersStatus::StopIteration,
            filter_->decodeTrailers(headers_));
}

TEST_F(ServiceControlFilterTest, DecodeHelpersWhileContinuing) {
  // This puts the Filter into a continue state
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(Invoke([](Http::HeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK);
      }));
  EXPECT_EQ(Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, true));

  // Test: When Filter is Complete, decodeData returns Continue
  EXPECT_EQ(Http::FilterDataStatus::Continue,
            filter_->decodeData(mock_buffer_, true));

  // Test: When Filter is Complete, decodeTrailers returns Continue
  EXPECT_EQ(Http::FilterTrailersStatus::Continue,
            filter_->decodeTrailers(headers_));
}

TEST_F(ServiceControlFilterTest, DecodeDataSendStreamReport) {
  // This puts the Filter into a continue state
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(Invoke([](Http::HeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK);
      }));
  EXPECT_EQ(Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, /*end_stream=*/true));

  mock_buffer_.add("filler");

  EXPECT_CALL(*mock_handler, tryIntermediateReport(_));
  filter_->decodeData(mock_buffer_, /*end_stream=*/false);
}

TEST_F(ServiceControlFilterTest, EncodeDataSendStreamReport) {
  // This puts the Filter into a continue state
  auto* mock_handler = new testing::NiceMock<MockServiceControlHandler>();
  EXPECT_CALL(mock_handler_factory_, createHandler_(_, _))
      .WillOnce(Return(mock_handler));
  EXPECT_CALL(*mock_handler, callCheck(_, _, _))
      .WillOnce(Invoke([](Http::HeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK);
      }));
  EXPECT_EQ(Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(headers_, /*end_stream=*/true));

  mock_buffer_.add("filler");

  EXPECT_CALL(*mock_handler, tryIntermediateReport(_));
  filter_->encodeData(mock_buffer_, /*end_stream=*/false);
}

}  // namespace

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
