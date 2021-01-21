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

#include "src/envoy/http/service_control/filter.h"

#include "common/common/empty_string.h"
#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/handler.h"
#include "src/envoy/http/service_control/mocks.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/stats/mocks.h"
#include "test/mocks/tracing/mocks.h"
#include "test/test_common/utility.h"

using Envoy::Http::MockStreamDecoderFilterCallbacks;
using Envoy::Server::Configuration::MockFactoryContext;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
using ::testing::_;
using ::testing::ByMove;
using ::testing::Invoke;
using ::testing::Return;

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {
namespace {

const Status kBadStatus(Code::UNAUTHENTICATED, "test");

class ServiceControlFilterTest : public ::testing::Test {
 protected:
  ServiceControlFilterTest()
      : stats_(ServiceControlFilterStats::create(Envoy::EMPTY_STRING,
                                                 mock_stats_scope_)),
        req_headers_{{":method", "GET"}, {":path", "/bar"}} {}

  void SetUp() override {
    filter_ =
        std::make_unique<ServiceControlFilter>(stats_, mock_handler_factory_);
    filter_->setDecoderFilterCallbacks(mock_decoder_callbacks_);

    mock_handler_ = new testing::NiceMock<MockServiceControlHandler>();
    mock_handler_ptr_.reset(mock_handler_);
    mock_route_ = std::make_shared<NiceMock<Envoy::Router::MockRoute>>();
    ON_CALL(mock_handler_factory_, createHandler(_, _, _))
        .WillByDefault(Return(ByMove(std::move(mock_handler_ptr_))));

    mock_span_ = std::make_unique<Envoy::Tracing::MockSpan>();
  }

  std::unique_ptr<ServiceControlFilter> filter_;
  testing::NiceMock<MockStreamDecoderFilterCallbacks> mock_decoder_callbacks_;
  testing::NiceMock<MockFactoryContext> mock_factory_context_;
  testing::NiceMock<MockServiceControlHandlerFactory> mock_handler_factory_;
  testing::NiceMock<MockServiceControlHandler>* mock_handler_;
  ServiceControlHandlerPtr mock_handler_ptr_;
  testing::NiceMock<Envoy::MockBuffer> mock_buffer_;
  testing::NiceMock<Envoy::Stats::MockIsolatedStatsStore> mock_stats_scope_;
  ServiceControlFilterStats stats_;
  std::shared_ptr<NiceMock<Envoy::Router::MockRoute>> mock_route_;
  Envoy::Http::TestRequestHeaderMapImpl req_headers_;
  Envoy::Http::TestRequestTrailerMapImpl req_trailer_;
  Envoy::Http::TestResponseHeaderMapImpl resp_headers_;
  Envoy::Http::TestResponseTrailerMapImpl resp_trailer_;

  // Tracing mocks
  std::unique_ptr<Envoy::Tracing::MockSpan> mock_span_;
};

TEST_F(ServiceControlFilterTest, DecodeHeadersMissingHeaders) {
  Envoy::Http::TestRequestHeaderMapImpl missingPath{{":method", "POST"}};
  Envoy::Http::TestRequestHeaderMapImpl missingMethod{{":path", "/bar"}};

  // Filter should reject this request
  EXPECT_CALL(mock_decoder_callbacks_.stream_info_,
              setResponseFlag(
                  Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService))
      .Times(2);

  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(Envoy::Http::Code::BadRequest,
                             "No path in request headers.", _, _,
                             "service_control_bad_request{MISSING_PATH}"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(missingPath, true));
  EXPECT_CALL(mock_decoder_callbacks_,
              sendLocalReply(Envoy::Http::Code::BadRequest,
                             "No method in request headers.", _, _,
                             "service_control_bad_request{MISSING_METHOD}"))
      .Times(1);
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(missingMethod, true));
}

TEST_F(ServiceControlFilterTest, DecodeHeadersSyncOKStatus) {
  // Test: If onCall is called with OK status, return Continue

  // Call onCheckDone synchronously
  EXPECT_CALL(*mock_handler_, callCheck(_, _, _))
      .WillOnce(Invoke([](Envoy::Http::RequestHeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK, "");
      }));
  EXPECT_CALL(*mock_handler_, fillFilterState(_));
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(req_headers_, true));

  // Verify handler->onDestroy is called when filter::onDestroy() is called
  EXPECT_CALL(*mock_handler_, onDestroy()).Times(1);
  filter_->onDestroy();
}

TEST_F(ServiceControlFilterTest, OnDestoryWithoutHandler) {
  // Test: calling filter::onDestroy() without handler
  EXPECT_CALL(mock_handler_factory_, createHandler(_, _, _)).Times(0);
  filter_->onDestroy();
}

TEST_F(ServiceControlFilterTest, DecodeHeadersSyncBadStatus) {
  // Test: If onCall is called with a bad status, reject and return stop

  // Call onCheckDone synchronously
  EXPECT_CALL(*mock_handler_, callCheck(_, _, _))
      .WillOnce(Invoke([](Envoy::Http::RequestHeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(kBadStatus,
                             "service_control_check_error{API_KEY_INVALID}");
      }));

  EXPECT_CALL(
      mock_decoder_callbacks_.stream_info_,
      setResponseFlag(
          Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService));
  EXPECT_CALL(
      mock_decoder_callbacks_,
      sendLocalReply(Envoy::Http::Code::Unauthorized, "UNAUTHENTICATED:test", _,
                     _, "service_control_check_error{API_KEY_INVALID}"));

  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(req_headers_, true));
}

TEST_F(ServiceControlFilterTest, DecodeHeadersAsyncGoodStatus) {
  // Test: While Filter is Calling/stopped, onCheckDone calls
  // continueDecoding
  ServiceControlHandler::CheckDoneCallback* stored_check_done_callback;

  // Store CheckDoneCallback but do not call it yet
  EXPECT_CALL(*mock_handler_, callCheck(_, _, _))
      // This invocation works around SaveArg storing the value to
      // the pointer in a way that does not work with the interface
      .WillOnce(
          Invoke([&stored_check_done_callback](
                     Envoy::Http::RequestHeaderMap&, Envoy::Tracing::Span&,
                     ServiceControlHandler::CheckDoneCallback& callback) {
            stored_check_done_callback = &callback;
          }));
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(req_headers_, true));

  EXPECT_CALL(mock_decoder_callbacks_, continueDecoding());
  stored_check_done_callback->onCheckDone(Status::OK, "");
}

TEST_F(ServiceControlFilterTest, DecodeHeadersAsyncBadStatus) {
  // Test: When status is not ok, the request is rejected and
  // continueDecoding is not called
  EXPECT_CALL(mock_decoder_callbacks_, continueDecoding()).Times(0);

  ServiceControlHandler::CheckDoneCallback* stored_check_done_callback;

  // Store CheckDoneCallback but do not call it yet
  EXPECT_CALL(*mock_handler_, callCheck(_, _, _))
      .WillOnce(
          // This invocation works around SaveArg storing the value to
          // the pointer in a way that does not work with the interface
          Invoke([&stored_check_done_callback](
                     Envoy::Http::RequestHeaderMap&, Envoy::Tracing::Span&,
                     ServiceControlHandler::CheckDoneCallback& callback) {
            stored_check_done_callback = &callback;
          }));
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(req_headers_, true));

  // Filter should reject this request
  EXPECT_CALL(
      mock_decoder_callbacks_.stream_info_,
      setResponseFlag(
          Envoy::StreamInfo::ResponseFlag::UnauthorizedExternalService));

  EXPECT_CALL(
      mock_decoder_callbacks_,
      sendLocalReply(Envoy::Http::Code::Unauthorized, "UNAUTHENTICATED:test", _,
                     _, "service_control_check_error{API_KEY_INVALID}"));
  stored_check_done_callback->onCheckDone(
      kBadStatus, "service_control_check_error{API_KEY_INVALID}");
}

TEST_F(ServiceControlFilterTest, LogWithoutHandlerOrHeaders) {
  // Test: If no handler and no headers, a handler is not created
  EXPECT_CALL(mock_handler_factory_, createHandler(_, _, _)).Times(0);

  // Filter has no handler. If it tries to callReport, it will seg fault
  filter_->log(nullptr, &resp_headers_, &resp_trailer_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, LogWithoutHandler) {
  // Test: When a Filter has no Handler yet, another is created for log()
  EXPECT_CALL(*mock_handler_, callReport(_, _, _, _));
  filter_->log(&req_headers_, &resp_headers_, &resp_trailer_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, LogWithHandler) {
  // Test: When a Filter has a Handler from decodeHeaders,
  // that one is used for log() and another is not created
  filter_->decodeHeaders(req_headers_, true);

  EXPECT_CALL(mock_handler_factory_, createHandler(_, _, _)).Times(0);
  EXPECT_CALL(*mock_handler_, callReport(_, _, _, _));
  filter_->log(&req_headers_, &resp_headers_, &resp_trailer_,
               mock_decoder_callbacks_.stream_info_);
}

TEST_F(ServiceControlFilterTest, DecodeHelpersWhileStopped) {
  // This puts the Filter into a stopped state
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::StopIteration,
            filter_->decodeHeaders(req_headers_, true));

  // Test: While Filter is Calling/stopped, decodeData returns Stop
  EXPECT_EQ(Envoy::Http::FilterDataStatus::StopIterationAndWatermark,
            filter_->decodeData(mock_buffer_, true));

  // Test: While Filter is Calling/stopped, decodeTrailers returns Stop
  EXPECT_EQ(Envoy::Http::FilterTrailersStatus::StopIteration,
            filter_->decodeTrailers(req_trailer_));
}

TEST_F(ServiceControlFilterTest, DecodeHelpersWhileContinuing) {
  // This puts the Filter into a continue state
  EXPECT_CALL(*mock_handler_, callCheck(_, _, _))
      .WillOnce(Invoke([](Envoy::Http::RequestHeaderMap&, Envoy::Tracing::Span&,
                          ServiceControlHandler::CheckDoneCallback& callback) {
        callback.onCheckDone(Status::OK, "");
      }));
  EXPECT_EQ(Envoy::Http::FilterHeadersStatus::Continue,
            filter_->decodeHeaders(req_headers_, true));

  // Test: When Filter is Complete, decodeData returns Continue
  EXPECT_EQ(Envoy::Http::FilterDataStatus::Continue,
            filter_->decodeData(mock_buffer_, true));

  // Test: When Filter is Complete, decodeTrailers returns Continue
  EXPECT_EQ(Envoy::Http::FilterTrailersStatus::Continue,
            filter_->decodeTrailers(req_trailer_));
}

}  // namespace

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
