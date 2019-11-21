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

#include "envoy/http/header_map.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/tracing/mocks.h"

#include "src/envoy/http/service_control/handler_impl.h"
#include "src/envoy/http/service_control/mocks.h"
#include "src/envoy/utils/filter_state_utils.h"

using Envoy::Http::TestHeaderMapImpl;
using Envoy::StreamInfo::MockStreamInfo;
using ::google::api::envoy::http::service_control::FilterConfig;
using ::google::api_proxy::service_control::CheckRequestInfo;
using ::google::api_proxy::service_control::CheckResponseInfo;
using ::google::api_proxy::service_control::QuotaRequestInfo;
using ::google::api_proxy::service_control::ReportRequestInfo;
using ::google::api_proxy::service_control::protocol::Protocol;
using ::google::protobuf::TextFormat;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
using ::testing::_;
using ::testing::MockFunction;
using ::testing::Return;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

const char kFilterConfig[] = R"(
services {
  service_name: "echo"
  backend_protocol: "grpc"
  producer_project_id: "project-id"
  log_request_headers: "x-test-log-request-header"
  log_response_headers: "x-test-log-response-header"
  min_stream_report_interval_ms: 100
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_no_key"
  api_key: {
    allow_without_api_key: true
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_default_location"
  api_key: {
    allow_without_api_key: false
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_query_key"
  api_key: {
    allow_without_api_key: false
    locations: {
      query: "api_key"
    }
    locations: {
      query: "key"
    }
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_header_key"
  api_key: {
    allow_without_api_key: false
    locations: {
      header: "x-api-key"
    }
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_header_key_quota"
  api_key: {
    allow_without_api_key: false
    locations: {
      header: "x-api-key"
    }
  }
  metric_costs: {
    name: "metric_name_1"
    cost: 2
  }
  metric_costs: {
    name: "metric_name_1"
    cost: 2
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "call_quota_without_check"
  api_key: {
    allow_without_api_key: true
  }
  metric_costs: {
    name: "metric_name"
    cost: 1
  }
}
requirements {
  service_name: "echo"
  api_name: "test_api"
  api_version: "test_version"
  operation_name: "get_cookie_key"
  api_key: {
    allow_without_api_key: false
    locations: {
      cookie: "api_key"
    }
  }
})";

class HandlerTest : public ::testing::Test {
 protected:
  HandlerTest() {}

  ~HandlerTest() {}

  void SetUp() override { setUp(kFilterConfig); }

  // Some tests require a different config_proto. They can call this method to
  // override the first setUp with the second one.
  void setUp(const char* filter_config) {
    // Destroy cfg_parser_ before assigning a new one so that the mock_call_
    // it manages can also be destroyed. This is required in order to get
    // mock_call_ expectations on the correct instance.
    cfg_parser_ = nullptr;
    mock_call_ = new testing::NiceMock<MockServiceControlCall>();

    FilterConfig proto_config;
    ASSERT_TRUE(TextFormat::ParseFromString(filter_config, &proto_config));
    EXPECT_CALL(mock_call_factory_, create_(_, _)).WillOnce(Return(mock_call_));
    cfg_parser_ =
        std::make_unique<FilterConfigParser>(proto_config, mock_call_factory_);

    mock_span_ = std::make_unique<Envoy::Tracing::MockSpan>();
  }

  testing::NiceMock<MockCheckDoneCallback> mock_check_done_callback_;
  testing::NiceMock<MockStreamInfo> mock_stream_info_;
  testing::NiceMock<MockServiceControlCallFactory> mock_call_factory_;

  // This pointer is managed by cfg_parser
  testing::NiceMock<MockServiceControlCall>* mock_call_;
  std::unique_ptr<FilterConfigParser> cfg_parser_;

  std::chrono::time_point<std::chrono::system_clock> epoch_{};

  // Tracing mocks
  std::unique_ptr<Envoy::Tracing::MockSpan> mock_span_;
};

MATCHER_P(MatchesCheckInfo, expect, "") {
  // These must match. If not provided in expect, arg should be empty too
  if (arg.api_key != expect.api_key) return false;
  if (arg.ios_bundle_id != expect.ios_bundle_id) return false;
  if (arg.referer != expect.referer) return false;
  if (arg.android_package_name != expect.android_package_name) return false;
  if (arg.android_cert_fingerprint != expect.android_cert_fingerprint)
    return false;

  // These should not change
  if (arg.client_ip != "127.0.0.1") return false;

  if (arg.operation_id != "test-uuid") return false;
  if (arg.operation_name != "get_header_key") return false;
  if (arg.producer_project_id != "project-id") return false;
  return true;
}

MATCHER_P(MatchesQuotaInfo, expect, "") {
  if (arg.method_name != expect.method_name) return false;
  if (arg.metric_cost_vector != expect.metric_cost_vector) return false;

  if (arg.operation_id != "test-uuid") return false;
  if (arg.operation_name != expect.method_name) return false;
  if (arg.api_key != expect.api_key) return false;
  if (arg.producer_project_id != "project-id") return false;
  return true;
}

#define MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name)        \
  if (arg.api_method != operation_name) return false;                 \
  if (arg.operation_name != operation_name) return false;             \
  if (arg.log_message != operation_name + " is called") return false; \
  if (arg.api_key != expect.api_key) return false;                    \
  if (arg.status != expect.status) return false;                      \
  if (arg.request_headers != expect.request_headers) return false;    \
  if (arg.response_headers != expect.response_headers) return false;  \
  if (arg.streaming_request_message_counts !=                         \
      expect.streaming_request_message_counts)                        \
    return false;                                                     \
  if (arg.is_first_report != expect.is_first_report) return false;    \
  if (arg.is_final_report != expect.is_final_report) return false;    \
  if (arg.url != "/echo") return false;                               \
  if (arg.api_name != "test_api") return false;                       \
  if (arg.api_version != "test_version") return false;                \
  if (arg.streaming_durations != expect.streaming_durations) {        \
    return false;                                                     \
  }                                                                   \
  if (arg.streaming_response_message_counts !=                        \
      expect.streaming_response_message_counts)                       \
    return false;                                                     \
  if (arg.method != "GET") return false;

MATCHER_P4(MatchesReportInfo, expect, request_headers, response_headers,
           response_trailers, "") {
  std::string operation_name =
      (expect.operation_name.empty() ? "get_header_key"
                                     : expect.operation_name);
  MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name)

  if (arg.backend_protocol != Protocol::GRPC) return false;
  if (arg.frontend_protocol != Protocol::GRPC) return false;

  int64_t request_size = request_headers.byteSizeInternal();
  if (arg.request_bytes != request_size || arg.request_size != request_size) {
    return false;
  }

  int64_t response_size = response_headers.byteSizeInternal() +
                          response_trailers.byteSizeInternal();
  if (arg.response_bytes != response_size ||
      arg.response_size != response_size) {
    return false;
  }

  return true;
}

MATCHER_P(MatchesDataReportInfo, expect, "") {
  std::string operation_name =
      (expect.operation_name.empty() ? "get_header_key"
                                     : expect.operation_name);

  MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name)

  // the buffer implementation is doing.
  if (arg.request_bytes != expect.request_bytes) return false;
  if (arg.response_bytes != expect.response_bytes) return false;

  return true;
}

TEST_F(HandlerTest, HandlerNoOperationFound) {
  // Test: If no operation is found, check should 404 and report should do
  // nothing

  // Note: The operation is set in mock_stream_info_.filter_state_. This test
  // should not set that value.
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  EXPECT_CALL(mock_check_done_callback_,
              onCheckDone(Status(Code::NOT_FOUND, "Method does not exist.")));
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  EXPECT_CALL(*mock_call_, callReport(_)).Times(0);
  handler.callReport(&headers, &headers, &headers);
}

TEST_F(HandlerTest, HandlerNoRequirementMatched) {
  // Test: If no requirement is matched for the operation, check should 404
  // and report should do nothing
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "bad-operation-name");
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(mock_check_done_callback_,
              onCheckDone(Status(Code::NOT_FOUND, "Method does not exist.")));
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  EXPECT_CALL(*mock_call_, callReport(_)).Times(0);
  handler.callReport(&headers, &headers, &headers);
}

TEST_F(HandlerTest, HandlerCheckNotNeeded) {
  // Test: If the operation does not require check, check should return OK
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_no_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  EXPECT_CALL(*mock_call_, callQuota(_, _)).Times(0);
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // no api key is set on this info
  ReportRequestInfo expected_report_info;
  expected_report_info.status = Status::OK;
  expected_report_info.operation_name = "get_no_key";
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerCheckMissingApiKey) {
  // Test: If the operation requires a check but none is found, check fails
  // and a report is made
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};

  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  Status bad_status =
      Status(Code::UNAUTHENTICATED,
             "Method doesn't allow unregistered callers (callers without "
             "established identity). Please use API Key or other form of "
             "API consumer identity to call this API.");
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  EXPECT_CALL(*mock_call_, callQuota(_, _)).Times(0);
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(bad_status));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // no api key is set on this info
  ReportRequestInfo expected_report_info;
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckSyncWithApiKeyRestrictionFields) {
  // Test: Check is required and succeeds, and api key restriction fields are
  // present on the check request
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"},
                                  {":path", "/echo"},
                                  {"x-api-key", "foobar"},
                                  {"x-ios-bundle-identifier", "ios-bundle-id"},
                                  {"referer", "referer"},
                                  {"x-android-package", "android-package"},
                                  {"x-android-cert", "cert-123"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;

  CheckRequestInfo expected_check_info;
  expected_check_info.api_key = "foobar";
  expected_check_info.android_package_name = "android-package";
  expected_check_info.android_cert_fingerprint = "cert-123";
  expected_check_info.ios_bundle_id = "ios-bundle-id";
  expected_check_info.referer = "referer";
  EXPECT_CALL(*mock_call_,
              callCheck(MatchesCheckInfo(expected_check_info), _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckSyncWithoutApiKeyRestrictionFields) {
  // Test: Check is required and succeeds. The api key restriction fields are
  // left blank if not provided.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;

  CheckRequestInfo expected_check_info;
  expected_check_info.api_key = "foobar";
  EXPECT_CALL(*mock_call_,
              callCheck(MatchesCheckInfo(expected_check_info), _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulQuotaSync) {
  // Test: Quota is required and succeeds.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;

  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  QuotaRequestInfo expected_quota_info;
  expected_quota_info.method_name = "get_header_key_quota";
  expected_quota_info.api_key = "foobar";
  expected_quota_info.metric_cost_vector =
      cfg_parser_->FindRequirement("get_header_key_quota")->metric_costs();

  EXPECT_CALL(*mock_call_, callQuota(MatchesQuotaInfo(expected_quota_info), _))
      .WillOnce(Invoke([](const QuotaRequestInfo&, QuotaDoneFunc on_done) {
        on_done(Status::OK);
      }));

  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  EXPECT_CALL(*mock_call_, callReport(_));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerCallQuotaWithoutCheck) {
  // Test: Quota is required but the Check is not
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "call_quota_without_check");
  Http::TestHeaderMapImpl headers{{":method", "GET"},
                                  {":path", "/echo?key=foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  // Check is not called.
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);

  QuotaRequestInfo expected_quota_info;
  expected_quota_info.method_name = "call_quota_without_check";
  expected_quota_info.api_key = "foobar";
  expected_quota_info.metric_cost_vector =
      cfg_parser_->FindRequirement("call_quota_without_check")->metric_costs();

  EXPECT_CALL(*mock_call_, callQuota(MatchesQuotaInfo(expected_quota_info), _))
      .WillOnce(Invoke([](const QuotaRequestInfo&, QuotaDoneFunc on_done) {
        on_done(Status::OK);
      }));

  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  EXPECT_CALL(*mock_call_, callReport(_));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerFailCheckSync) {
  // Test: Check is required and a request is made, but service control
  // returns a bad status.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  Status bad_status = Status(Code::PERMISSION_DENIED,
                             "test bad status returned from service control");

  CheckResponseInfo response_info;
  response_info.is_api_key_valid = false;
  response_info.service_is_activated = false;
  CheckRequestInfo expected_check_info;
  expected_check_info.api_key = "foobar";
  EXPECT_CALL(*mock_call_,
              callCheck(MatchesCheckInfo(expected_check_info), _, _))
      .WillOnce(Invoke([&response_info, bad_status](const CheckRequestInfo&,
                                                    Envoy::Tracing::Span&,
                                                    CheckDoneFunc on_done) {
        on_done(bad_status, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(bad_status));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // no api key is set on this info
  ReportRequestInfo expected_report_info;
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerFailQuotaSync) {
  // Test: Check is required and a request is made, but service control
  // returns a bad status.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;

  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  QuotaRequestInfo expected_quota_info;
  expected_quota_info.method_name = "get_header_key_quota";
  expected_quota_info.api_key = "foobar";
  expected_quota_info.metric_cost_vector =
      cfg_parser_->FindRequirement("get_header_key_quota")->metric_costs();

  Status bad_status = Status(Code::RESOURCE_EXHAUSTED,
                             "test bad status returned from service control");
  EXPECT_CALL(*mock_call_, callQuota(MatchesQuotaInfo(expected_quota_info), _))
      .WillOnce(
          Invoke([bad_status](const QuotaRequestInfo&, QuotaDoneFunc on_done) {
            on_done(bad_status);
          }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(bad_status));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  EXPECT_CALL(*mock_call_, callReport(_));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckAsync) {
  // Test: Check is required and succeeds, even when the done callback is not
  // called until later.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;

  CheckRequestInfo expected_check_info;
  expected_check_info.api_key = "foobar";

  // Store the done callback
  CheckDoneFunc stored_on_done;
  EXPECT_CALL(*mock_call_,
              callCheck(MatchesCheckInfo(expected_check_info), _, _))
      .WillOnce(Invoke([&stored_on_done](const CheckRequestInfo&,
                                         Envoy::Tracing::Span&,
                                         CheckDoneFunc on_done) {
        stored_on_done = on_done;
        return nullptr;
      }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // Async, later call the done callback
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  stored_on_done(Status::OK, response_info);

  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulQuotaAsync) {
  // Test: Check is required and succeeds, even when the done callback is not
  // called until later.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));

  QuotaRequestInfo expected_quota_info;
  expected_quota_info.method_name = "get_header_key_quota";
  expected_quota_info.api_key = "foobar";
  expected_quota_info.metric_cost_vector =
      cfg_parser_->FindRequirement("get_header_key_quota")->metric_costs();
  // Store the done callback
  QuotaDoneFunc stored_on_done;
  EXPECT_CALL(*mock_call_, callQuota(MatchesQuotaInfo(expected_quota_info), _))
      .WillOnce(Invoke(
          [&stored_on_done](const QuotaRequestInfo&, QuotaDoneFunc on_done) {
            stored_on_done = on_done;
          }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // Async, later call the done callback
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  stored_on_done(Status::OK);

  ReportRequestInfo expected_report_info;
  expected_report_info.operation_name = "get_header_key_quota";
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerFailCheckAsync) {
  // Test: Check is required and a request is made, but later on service
  // control returns a bad status.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  CheckResponseInfo response_info;
  response_info.is_api_key_valid = false;
  response_info.service_is_activated = false;

  CheckRequestInfo expected_check_info;
  expected_check_info.api_key = "foobar";

  // Store the done callback
  CheckDoneFunc stored_on_done;
  EXPECT_CALL(*mock_call_,
              callCheck(MatchesCheckInfo(expected_check_info), _, _))
      .WillOnce(Invoke([&stored_on_done](const CheckRequestInfo&,
                                         Envoy::Tracing::Span&,
                                         CheckDoneFunc on_done) {
        stored_on_done = on_done;
        return nullptr;
      }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);
  EXPECT_CALL(*mock_call_, callQuota(_, _)).Times(0);

  // Async, later call the done callback
  Status bad_status = Status(Code::PERMISSION_DENIED,
                             "test bad status returned from service control");
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(bad_status));
  stored_on_done(bad_status, response_info);

  // no api key is set on this info
  ReportRequestInfo expected_report_info;
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerFailQuotaAsync) {
  // Test: Quota is required and a request is made, but later on service
  // control returns a bad status.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));

  QuotaRequestInfo expected_quota_info;
  expected_quota_info.method_name = "get_header_key_quota";
  expected_quota_info.api_key = "foobar";
  expected_quota_info.metric_cost_vector =
      cfg_parser_->FindRequirement("get_header_key_quota")->metric_costs();
  // Store the done callback
  QuotaDoneFunc stored_on_done;
  EXPECT_CALL(*mock_call_, callQuota(MatchesQuotaInfo(expected_quota_info), _))
      .WillOnce(Invoke(
          [&stored_on_done](const QuotaRequestInfo&, QuotaDoneFunc on_done) {
            stored_on_done = on_done;
          }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // Async, later call the done callback
  Status bad_status = Status(Code::RESOURCE_EXHAUSTED,
                             "test bad status returned from service control");
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(bad_status));
  stored_on_done(bad_status);

  ReportRequestInfo expected_report_info;
  expected_report_info.operation_name = "get_header_key_quota";
  expected_report_info.api_key = "foobar";
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerCancelFuncResetOnDone) {
  // Test: Cancel function will not be called if on_done is called
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  CheckDoneFunc stored_on_done;
  CheckResponseInfo response_info;
  MockFunction<void()> mock_cancel;
  CancelFunc cancel_fn = mock_cancel.AsStdFunction();

  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&stored_on_done, cancel_fn](const CheckRequestInfo&,
                                                    Envoy::Tracing::Span&,
                                                    CheckDoneFunc on_done) {
        stored_on_done = on_done;
        return cancel_fn;
      }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  stored_on_done(Status::OK, response_info);

  // Cancel is reset in the on_done() call. so onDestroy() will not call.
  EXPECT_CALL(mock_cancel, Call()).Times(0);
  handler.onDestroy();
}

TEST_F(HandlerTest, HandlerCancelFuncCalledOnDestroy) {
  // Test: Cancel function will be called if on_done is not called
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  CheckDoneFunc stored_on_done;
  CheckResponseInfo response_info;
  MockFunction<void()> mock_cancel;
  CancelFunc cancel_fn = mock_cancel.AsStdFunction();

  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&stored_on_done, cancel_fn](const CheckRequestInfo&,
                                                    Envoy::Tracing::Span&,
                                                    CheckDoneFunc on_done) {
        stored_on_done = on_done;
        return cancel_fn;
      }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // onDestroy() will call cancel function if on_done is not called.
  EXPECT_CALL(mock_cancel, Call()).Times(1);
  handler.onDestroy();
}

TEST_F(HandlerTest, HandlerReportWithoutCheck) {
  // Test: Test that callReport works when callCheck is not called first.
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  CheckDoneFunc stored_on_done;
  CheckResponseInfo response_info;
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  // The default value of status if a check is not made is OK
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, headers)));
  handler.callReport(&headers, &response_headers, &headers, epoch_);
}

TEST_F(HandlerTest, HandlerCollectDecodeData) {
  // CollectDecodeData test cases after the boilerplate
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"},
                                  {":path", "/echo"},
                                  {"x-api-key", "foobar"},
                                  {"content-type", "application/grpc"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  testing::NiceMock<Envoy::MockBuffer> mock_buffer;
  mock_buffer.writeByte(0);  // gRPC flags
  mock_buffer.writeByte(0);  // gRPC size (32 bit big endian)
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(1);
  mock_buffer.writeByte(128);  // gRPC payload
  ASSERT_EQ(mock_buffer.length(), 6);

  std::chrono::system_clock::time_point start_time =
      std::chrono::system_clock::now();

  handler.processResponseHeaders(response_headers);
  // Test: First call is skipped because start time == start time
  EXPECT_CALL(*mock_call_, callReport(_)).Times(0);
  handler.collectDecodeData(mock_buffer, start_time);

  // Test: Next call is skipped because not enough time has passed
  std::chrono::system_clock::time_point time = start_time;
  time += std::chrono::milliseconds(1);
  handler.collectDecodeData(mock_buffer, time);

  // Test: Next call is sent because enough time has passed
  time += std::chrono::milliseconds(200);
  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  expected_report_info.is_first_report = true;
  expected_report_info.is_final_report = false;
  expected_report_info.status = Status::OK;
  // streaming_request_message_counts and streaming_durations only exist in
  // the final report.
  expected_report_info.streaming_request_message_counts = 0;
  expected_report_info.streaming_durations = 0;
  expected_report_info.request_bytes =
      mock_buffer.length() * 3 + headers.byteSizeInternal();
  expected_report_info.response_bytes = response_headers.byteSizeInternal();
  mock_stream_info_.bytes_received_ = mock_buffer.length() * 3;
  mock_stream_info_.bytes_sent_ = 0;

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.collectDecodeData(mock_buffer, time);

  // Test: Next call is sent. First report is false
  time += std::chrono::milliseconds(200);
  expected_report_info.is_first_report = false;
  expected_report_info.request_bytes =
      mock_buffer.length() * 4 + headers.byteSizeInternal();
  mock_stream_info_.bytes_received_ = mock_buffer.length() * 4;
  mock_stream_info_.bytes_sent_ = 0;

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.collectDecodeData(mock_buffer, time);
}

TEST_F(HandlerTest, HandlerCollectEncodeData) {
  // CollectEncodeData test cases after the boilerplate
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"},
                                  {":path", "/echo"},
                                  {"x-api-key", "foobar"},
                                  {"content-type", "application/grpc"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  handler.processResponseHeaders(response_headers);
  testing::NiceMock<Envoy::MockBuffer> mock_buffer;
  mock_buffer.writeByte(0);  // gRPC flags
  mock_buffer.writeByte(0);  // gRPC size (32 bit big endian)
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(1);
  mock_buffer.writeByte(128);  // gRPC payload

  ASSERT_EQ(mock_buffer.length(), 6);

  std::chrono::system_clock::time_point start_time =
      std::chrono::system_clock::now();

  // Test: First call is skipped because start time == start time
  EXPECT_CALL(*mock_call_, callReport(_)).Times(0);
  handler.collectEncodeData(mock_buffer, start_time);

  // Test: Next call is skipped because not enough time has passed
  std::chrono::system_clock::time_point time = start_time;
  time += std::chrono::milliseconds(1);
  handler.collectEncodeData(mock_buffer, time);

  // Test: Next call is sent because enough time has passed
  time += std::chrono::milliseconds(200);
  // Now the start_time of streaming_info_ has been set.
  start_time = time;
  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";
  expected_report_info.is_first_report = true;
  expected_report_info.is_final_report = false;
  expected_report_info.status = Status::OK;
  // streaming_request_message_counts and streaming_durations only exist in
  // the final report.
  expected_report_info.streaming_response_message_counts = 0;
  expected_report_info.streaming_durations = 0;
  expected_report_info.request_bytes = headers.byteSizeInternal();
  expected_report_info.response_bytes =
      mock_buffer.length() * 3 + response_headers.byteSizeInternal();
  mock_stream_info_.bytes_received_ = 0;
  mock_stream_info_.bytes_sent_ = mock_buffer.length() * 3;

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.collectEncodeData(mock_buffer, time);

  // Test: Next call is sent. First report is false
  time += std::chrono::milliseconds(200);
  expected_report_info.is_first_report = false;
  expected_report_info.response_bytes =
      mock_buffer.length() * 4 + response_headers.byteSizeInternal();
  mock_stream_info_.bytes_sent_ = mock_buffer.length() * 4;

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.collectEncodeData(mock_buffer, time);
}

TEST_F(HandlerTest, FinalReports) {
  // CollectEncodeData test cases after the boilerplate
  Utils::setStringFilterState(mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  Http::TestHeaderMapImpl headers{{":method", "GET"},
                                  {":path", "/echo"},
                                  {"x-api-key", "foobar"},
                                  {"content-type", "application/grpc"}};
  Http::TestHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  CheckResponseInfo response_info;
  response_info.is_api_key_valid = true;
  response_info.service_is_activated = true;
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(Invoke([&response_info](const CheckRequestInfo&,
                                        Envoy::Tracing::Span&,
                                        CheckDoneFunc on_done) {
        on_done(Status::OK, response_info);
        return nullptr;
      }));
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  handler.processResponseHeaders(response_headers);
  testing::NiceMock<Envoy::MockBuffer> mock_buffer;
  mock_buffer.writeByte(0);  // gRPC flags
  mock_buffer.writeByte(0);  // gRPC size (32 bit big endian)
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(0);
  mock_buffer.writeByte(1);
  mock_buffer.writeByte(128);  // gRPC payload
  ASSERT_EQ(mock_buffer.length(), 6);

  std::chrono::system_clock::time_point start_time =
      std::chrono::system_clock::now();
  std::chrono::system_clock::time_point time = start_time;
  mock_stream_info_.start_time_ = start_time;

  handler.collectDecodeData(mock_buffer, time);
  handler.collectEncodeData(mock_buffer, time);

  time += std::chrono::milliseconds(200);
  int duration =
      std::chrono::duration_cast<std::chrono::microseconds>(time - start_time)
          .count();
  ReportRequestInfo expected_report_info;
  expected_report_info.api_key = "foobar";

  expected_report_info.is_first_report = true;
  expected_report_info.is_final_report = true;
  expected_report_info.status = Status::OK;

  expected_report_info.streaming_durations = duration;
  expected_report_info.streaming_request_message_counts = 1;
  expected_report_info.streaming_response_message_counts = 1;

  // Send 1 mock_buffer and 1 headers.
  expected_report_info.request_bytes =
      mock_buffer.length() * 1 + headers.byteSizeInternal() * 1;
  // Send 2 mock_buffer and 2 response_headers(1 as response_trailers).
  expected_report_info.response_bytes =
      mock_buffer.length() * 1 + response_headers.byteSizeInternal() * 2;

  // Check the final report.
  mock_stream_info_.bytes_sent_ = mock_buffer.length();
  mock_stream_info_.bytes_received_ = mock_buffer.length();
  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.callReport(&headers, &response_headers, &response_headers, time);
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
