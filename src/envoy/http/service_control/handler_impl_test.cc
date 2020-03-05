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
#include "extensions/filters/http/grpc_stats/grpc_stats_filter.h"
#include "gmock/gmock.h"
#include "google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "test/mocks/server/mocks.h"
#include "test/mocks/tracing/mocks.h"

#include "src/envoy/http/service_control/handler_impl.h"
#include "src/envoy/http/service_control/mocks.h"
#include "src/envoy/utils/filter_state_utils.h"

using Envoy::Http::TestRequestHeaderMapImpl;
using Envoy::Http::TestRequestTrailerMapImpl;
using Envoy::Http::TestResponseHeaderMapImpl;
using Envoy::Http::TestResponseTrailerMapImpl;
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

    ASSERT_TRUE(TextFormat::ParseFromString(filter_config, &proto_config_));
    EXPECT_CALL(mock_call_factory_, create_(_)).WillOnce(Return(mock_call_));
    cfg_parser_ =
        std::make_unique<FilterConfigParser>(proto_config_, mock_call_factory_);

    mock_span_ = std::make_unique<Envoy::Tracing::MockSpan>();
  }

  void initExpectedReportInfo(ReportRequestInfo& expected_report_info) {
    expected_report_info.api_name = "test_api";
    expected_report_info.api_version = "test_version";
    expected_report_info.url = "/echo";
    expected_report_info.method = "GET";
  }

  testing::NiceMock<MockCheckDoneCallback> mock_check_done_callback_;
  testing::NiceMock<MockStreamInfo> mock_stream_info_;
  testing::NiceMock<MockServiceControlCallFactory> mock_call_factory_;

  // This pointer is managed by cfg_parser
  testing::NiceMock<MockServiceControlCall>* mock_call_;
  FilterConfig proto_config_;
  std::unique_ptr<FilterConfigParser> cfg_parser_;

  std::chrono::time_point<std::chrono::system_clock> epoch_{};

  // Tracing mocks
  std::unique_ptr<Envoy::Tracing::MockSpan> mock_span_;
  TestRequestHeaderMapImpl req_headers_;
  TestRequestTrailerMapImpl req_trailer_;
  TestResponseHeaderMapImpl resp_headers_;
  TestResponseTrailerMapImpl resp_trailer_;
};

#define MATCH(name)                                              \
  if (arg.name != expect.name) {                                 \
    std::cerr << "MATCH fails for " << #name << ": " << arg.name \
              << " != " << expect.name << std::endl;             \
    return false;                                                \
  }
#define MATCH2(name, want)                                       \
  if (arg.name != want) {                                        \
    std::cerr << "MATCH fails for " << #name << ": " << arg.name \
              << " != " << want << std::endl;                    \
    return false;                                                \
  }

MATCHER_P(MatchesCheckInfo, expect, "") {
  // These must match. If not provided in expect, arg should be empty too
  MATCH(api_key);
  MATCH(ios_bundle_id);
  MATCH(referer);
  MATCH(android_package_name);
  MATCH(android_cert_fingerprint);

  // These should not change
  MATCH2(client_ip, "127.0.0.1");

  MATCH2(operation_id, "test-uuid");
  MATCH2(operation_name, "get_header_key");
  MATCH2(producer_project_id, "project-id");
  return true;
}

MATCHER_P(MatchesQuotaInfo, expect, "") {
  MATCH(method_name);
  //  if (arg.metric_cost_vector != expect.metric_cost_vector) return false;
  MATCH(metric_cost_vector);
  MATCH(api_key);

  MATCH2(operation_id, "test-uuid");
  MATCH2(operation_name, expect.method_name);
  MATCH2(producer_project_id, "project-id");
  return true;
}

#define MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name) \
  MATCH2(api_method, operation_name);                          \
  MATCH2(operation_name, operation_name);                      \
  MATCH2(log_message, operation_name + " is called");          \
  MATCH(api_key);                                              \
  MATCH(status);                                               \
  MATCH(request_headers);                                      \
  MATCH(response_headers);                                     \
  MATCH(is_first_report);                                      \
  MATCH(is_final_report);                                      \
  MATCH(url);                                                  \
  MATCH(method);                                               \
  MATCH(api_name);                                             \
  MATCH(api_version);                                          \
  MATCH(streaming_request_message_counts);                     \
  MATCH(streaming_response_message_counts);

// TODO(qiwzhang) b/146579874 to use timeSource MockStreamInfo is using
// different mock system_time, handle_impl.cc is diretly using system_clock
// it causes time duration test fails.
// Once the bug is fixed, add this check to above macro.
//  MATCH(streaming_durations);

MATCHER_P4(MatchesReportInfo, expect, request_headers, response_headers,
           response_trailers, "") {
  std::string operation_name =
      (expect.operation_name.empty() ? "get_header_key"
                                     : expect.operation_name);
  MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name);

  MATCH2(backend_protocol, Protocol::GRPC);
  MATCH2(frontend_protocol, Protocol::GRPC);

  int64_t request_size = request_headers.byteSize();
  MATCH2(request_bytes, request_size);
  MATCH2(request_size, request_size);

  int64_t response_size =
      response_headers.byteSize() + response_trailers.byteSize();
  MATCH2(response_bytes, response_size);
  MATCH2(response_size, response_size);
  return true;
}

MATCHER_P(MatchesSimpleReportInfo, expect, "") {
  std::string operation_name =
      (expect.operation_name.empty() ? "get_header_key"
                                     : expect.operation_name);
  MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name);
  return true;
}

MATCHER_P(MatchesDataReportInfo, expect, "") {
  std::string operation_name =
      (expect.operation_name.empty() ? "get_header_key"
                                     : expect.operation_name);

  MATCH_DEFAULT_REPORT_INFO(arg, expect, operation_name);

  // the buffer implementation is doing.
  MATCH(request_bytes);
  MATCH(response_bytes);
  return true;
}

TEST_F(HandlerTest, HandlerNoOperationFound) {
  // Test: If no operation is found, check should 404 and report should be
  // called.

  // Note: The operation is set in mock_stream_info_.filter_state_. This test
  // should not set that value.
  TestRequestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  EXPECT_CALL(mock_check_done_callback_,
              onCheckDone(Status(Code::NOT_FOUND, "Method does not exist.")));
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_name = "";
  expected_report_info.api_version = "";
  expected_report_info.status = Status::OK;
  expected_report_info.operation_name = "<Unknown Operation Name>";

  EXPECT_CALL(*mock_call_,
              callReport(MatchesSimpleReportInfo(expected_report_info)));
  handler.callReport(&headers, &resp_headers_, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerMissingHeaders) {
  // Test: If the request is missing :method and :path headers,
  // report should still be created without crashes.

  // Note: This test builds off of `HandlerNoOperationFound` to keep mocks
  // simple
  ServiceControlHandlerImpl handler(req_headers_, mock_stream_info_,
                                    "test-uuid", *cfg_parser_);

  EXPECT_CALL(mock_check_done_callback_,
              onCheckDone(Status(Code::NOT_FOUND, "Method does not exist.")));
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  handler.callCheck(req_headers_, *mock_span_, mock_check_done_callback_);

  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_name = "";
  expected_report_info.api_version = "";
  expected_report_info.status = Status::OK;
  expected_report_info.operation_name = "<Unknown Operation Name>";
  expected_report_info.url = "";
  expected_report_info.method = "";

  EXPECT_CALL(*mock_call_,
              callReport(MatchesSimpleReportInfo(expected_report_info)));
  handler.callReport(&req_headers_, &resp_headers_, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerNoRequirementMatched) {
  // Test: If no requirement is matched for the operation, check should 404
  // and report should do nothing
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "bad-operation-name");
  TestRequestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(mock_check_done_callback_,
              onCheckDone(Status(Code::NOT_FOUND, "Method does not exist.")));
  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_name = "";
  expected_report_info.api_version = "";
  expected_report_info.status = Status::OK;
  expected_report_info.operation_name = "<Unknown Operation Name>";
  EXPECT_CALL(*mock_call_,
              callReport(MatchesSimpleReportInfo(expected_report_info)));
  handler.callReport(&headers, &resp_headers_, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerCheckNotNeeded) {
  // Test: If the operation does not require check, check should return OK
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_no_key");
  TestRequestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  TestResponseHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  EXPECT_CALL(*mock_call_, callCheck(_, _, _)).Times(0);
  EXPECT_CALL(*mock_call_, callQuota(_, _)).Times(0);
  EXPECT_CALL(mock_check_done_callback_, onCheckDone(Status::OK));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // no api key is set on this info
  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.status = Status::OK;
  expected_report_info.operation_name = "get_no_key";
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerCheckMissingApiKey) {
  // Test: If the operation requires a check but none is found, check fails
  // and a report is made
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{{":method", "GET"}, {":path", "/echo"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckSyncWithApiKeyRestrictionFields) {
  // Test: Check is required and succeeds, and api key restriction fields are
  // present on the check request
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{{":method", "GET"},
                                   {":path", "/echo"},
                                   {"x-api-key", "foobar"},
                                   {"x-ios-bundle-identifier", "ios-bundle-id"},
                                   {"referer", "referer"},
                                   {"x-android-package", "android-package"},
                                   {"x-android-cert", "cert-123"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckSyncWithoutApiKeyRestrictionFields) {
  // Test: Check is required and succeeds. The api key restriction fields are
  // left blank if not provided.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulQuotaSync) {
  // Test: Quota is required and succeeds.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerCallQuotaWithoutCheck) {
  // Test: Quota is required but the Check is not
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "call_quota_without_check");
  TestRequestHeaderMapImpl headers{{":method", "GET"},
                                   {":path", "/echo?key=foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerFailCheckSync) {
  // Test: Check is required and a request is made, but service control
  // returns a bad status.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerFailQuotaSync) {
  // Test: Check is required and a request is made, but service control
  // returns a bad status.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulCheckAsync) {
  // Test: Check is required and succeeds, even when the done callback is not
  // called until later.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerSuccessfulQuotaAsync) {
  // Test: Check is required and succeeds, even when the done callback is not
  // called until later.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.operation_name = "get_header_key_quota";
  expected_report_info.api_key = "foobar";
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerFailCheckAsync) {
  // Test: Check is required and a request is made, but later on service
  // control returns a bad status.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerFailQuotaAsync) {
  // Test: Quota is required and a request is made, but later on service
  // control returns a bad status.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key_quota");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
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
  initExpectedReportInfo(expected_report_info);
  expected_report_info.operation_name = "get_header_key_quota";
  expected_report_info.api_key = "foobar";
  expected_report_info.status = bad_status;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, HandlerCancelFuncResetOnDone) {
  // Test: Cancel function will not be called if on_done is called
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
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
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  MockFunction<void()> mock_cancel;
  CancelFunc cancel_fn = mock_cancel.AsStdFunction();

  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(
          Invoke([cancel_fn](const CheckRequestInfo&, Envoy::Tracing::Span&,
                             CheckDoneFunc) { return cancel_fn; }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // onDestroy() will call cancel function if on_done is not called.
  EXPECT_CALL(mock_cancel, Call()).Times(1);
  handler.onDestroy();
}

TEST_F(HandlerTest, HandlerCancelFuncNotCalledOnDestroyForSyncOnDone) {
  // Test: Cancel function will not be called if on_done is called synchronously
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  MockFunction<void()> mock_cancel;
  CancelFunc cancel_fn = mock_cancel.AsStdFunction();

  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);
  EXPECT_CALL(*mock_call_, callCheck(_, _, _))
      .WillOnce(
          Invoke([cancel_fn](const CheckRequestInfo&, Envoy::Tracing::Span&,
                             CheckDoneFunc on_done) {
            CheckResponseInfo response_info;
            on_done(Status::OK, response_info);
            return cancel_fn;
          }));
  handler.callCheck(headers, *mock_span_, mock_check_done_callback_);

  // onDestroy() will not call cancel function if on_done is called
  // synchronously.
  EXPECT_CALL(mock_cancel, Call()).Times(0);
  handler.onDestroy();
}

TEST_F(HandlerTest, HandlerReportWithoutCheck) {
  // Test: Test that callReport works when callCheck is not called first.
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/echo"}, {"x-api-key", "foobar"}};
  TestResponseHeaderMapImpl response_headers{
      {"content-type", "application/grpc"}};
  CheckDoneFunc stored_on_done;
  CheckResponseInfo response_info;
  ServiceControlHandlerImpl handler(headers, mock_stream_info_, "test-uuid",
                                    *cfg_parser_);

  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";
  // The default value of status if a check is not made is OK
  expected_report_info.status = Status::OK;
  EXPECT_CALL(*mock_call_,
              callReport(MatchesReportInfo(expected_report_info, headers,
                                           response_headers, resp_trailer_)));
  handler.callReport(&headers, &response_headers, &resp_trailer_, epoch_);
}

TEST_F(HandlerTest, TryIntermediateReport) {
  // CollectDecodeData test cases after the boilerplate
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{{":method", "GET"},
                                   {":path", "/echo"},
                                   {"x-api-key", "foobar"},
                                   {"content-type", "application/grpc"}};
  TestResponseHeaderMapImpl response_headers{
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

  std::chrono::system_clock::time_point start_time =
      std::chrono::system_clock::now();

  handler.processResponseHeaders(response_headers);
  // Test: First call is skipped because start time == start time
  EXPECT_CALL(*mock_call_, callReport(_)).Times(0);
  handler.tryIntermediateReport(start_time);

  // Test: Next call is skipped because not enough time has passed
  std::chrono::system_clock::time_point time = start_time;
  time += std::chrono::milliseconds(1);
  handler.tryIntermediateReport(time);

  // Test: Next call is sent because enough time has passed
  time += std::chrono::milliseconds(200);
  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";
  expected_report_info.is_first_report = true;
  expected_report_info.is_final_report = false;
  expected_report_info.status = Status::OK;
  // streaming_request_message_counts and streaming_durations only exist in
  // the final report.
  expected_report_info.streaming_request_message_counts = 0;
  expected_report_info.streaming_durations = 0;

  // Mock stream_info_ bytes
  mock_stream_info_.bytes_received_ = 123;
  mock_stream_info_.bytes_sent_ = 456;
  // request_bytes = mock_stream_info.bytes_received_ + headers.
  expected_report_info.request_bytes =
      mock_stream_info_.bytes_received_ + headers.byteSize();
  // response_bytes = mock_stream_info_.bytes_sent_ + response headers
  expected_report_info.response_bytes =
      mock_stream_info_.bytes_sent_ + response_headers.byteSize();

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.tryIntermediateReport(time);

  // Test: Next call is sent. First report is false
  time += std::chrono::milliseconds(200);
  expected_report_info.is_first_report = false;

  mock_stream_info_.bytes_received_ = 789;
  mock_stream_info_.bytes_sent_ = 1456;
  expected_report_info.request_bytes =
      mock_stream_info_.bytes_received_ + headers.byteSize();
  expected_report_info.response_bytes =
      mock_stream_info_.bytes_sent_ + response_headers.byteSize();

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.tryIntermediateReport(time);
}

TEST_F(HandlerTest, FinalReports) {
  // CollectEncodeData test cases after the boilerplate
  Utils::setStringFilterState(*mock_stream_info_.filter_state_,
                              Utils::kOperation, "get_header_key");
  TestRequestHeaderMapImpl headers{{":method", "GET"},
                                   {":path", "/echo"},
                                   {"x-api-key", "foobar"},
                                   {"content-type", "application/grpc"}};
  TestResponseHeaderMapImpl response_headers{
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

  std::chrono::system_clock::time_point start_time =
      std::chrono::system_clock::now();
  std::chrono::system_clock::time_point time = start_time;
  mock_stream_info_.start_time_ = start_time;

  handler.tryIntermediateReport(time);

  time += std::chrono::milliseconds(200);
  int duration =
      std::chrono::duration_cast<std::chrono::microseconds>(time - start_time)
          .count();
  ReportRequestInfo expected_report_info;
  initExpectedReportInfo(expected_report_info);
  expected_report_info.api_key = "foobar";

  expected_report_info.is_first_report = true;
  expected_report_info.is_final_report = true;
  expected_report_info.status = Status::OK;

  expected_report_info.streaming_durations = duration;

  {
    // message_counts is from grpc_stats filterState.
    auto grpc_state = std::make_unique<GrpcStats::GrpcStatsObject>();
    grpc_state->request_message_count = 123;
    grpc_state->response_message_count = 456;
    mock_stream_info_.filter_state_->setData(
        HttpFilterNames::get().GrpcStats, std::move(grpc_state),
        StreamInfo::FilterState::StateType::Mutable);
  }
  expected_report_info.streaming_request_message_counts = 123;
  expected_report_info.streaming_response_message_counts = 456;

  // Check the final report.
  mock_stream_info_.bytes_received_ = 123;
  mock_stream_info_.bytes_sent_ = 456;
  // request_bytes = mock_stream_info.bytes_received_ + 1 headers.
  expected_report_info.request_bytes =
      mock_stream_info_.bytes_received_ + headers.byteSize();
  // response_bytes = mock_stream_info_.bytes_sent_
  //  + response_headers + response_trailers.
  expected_report_info.response_bytes = mock_stream_info_.bytes_sent_ +
                                        response_headers.byteSize() +
                                        resp_trailer_.byteSize();

  EXPECT_CALL(*mock_call_,
              callReport(MatchesDataReportInfo(expected_report_info)))
      .Times(1);
  handler.callReport(&headers, &response_headers, &resp_trailer_, time);
}

}  // namespace
}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
