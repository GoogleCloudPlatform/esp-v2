// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlCheckRetry(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed", "--service_control_check_retries=2", "--service_control_check_timeout_ms=100"}
	s := env.NewTestEnv(platform.TestServiceControlCheckRetry, platform.GrpcBookstoreSidecar)
	handler := utils.RetryServiceHandler{}
	s.ServiceControlServer.OverrideCheckHandler(&handler)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		token                   string
		method                  string
		sleepTimes              int32
		sleepLengthMs           int32
		wantResp                string
		wantError               string
		wantHandlerRequestCount int32
	}{
		{
			desc:                    "Backend unresponsive, the proxy will retry the check request 3 times and fail",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           200,
			method:                  "/v1/shelves?key=api-key-0",
			token:                   testdata.FakeCloudTokenMultiAudiences,
			wantHandlerRequestCount: 3,
			wantError:               `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:Calling Google Service Control API failed with: 504 and body: upstream request timeout"}`,
		},
		{
			desc:                    "Backend responsive, the proxy will retry the check request 3 times and get the response in the last request",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              2,
			sleepLengthMs:           200, // The handler will sleep too long twice, so envoy will retry these requests
			method:                  "/v1/shelves?key=api-key-1",
			token:                   testdata.FakeCloudTokenMultiAudiences,
			wantHandlerRequestCount: 3,
			wantResp:                `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:                    "Backend responsive, the proxy will do a check request once and get a response with no retries",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           0, // The handler will not sleep, so envoy's request to the backend should be successful
			method:                  "/v1/shelves?key=api-key-2",
			token:                   testdata.FakeCloudTokenMultiAudiences,
			wantHandlerRequestCount: 1,
			wantResp:                `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
	}

	for _, tc := range tests {
		handler.RequestCount = 0
		handler.SleepTimes = tc.sleepTimes
		handler.SleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)
		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}

		if handler.RequestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected report request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.RequestCount)
		}
	}
}

func TestServiceControlQuotaRetry(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed", "--service_control_quota_retries=2", "--service_control_quota_timeout_ms=100"}
	s := env.NewTestEnv(platform.TestServiceControlQuotaRetry, platform.GrpcBookstoreSidecar)
	s.OverrideQuota(&confpb.Quota{
		MetricRules: []*confpb.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
			{
				Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	handler := utils.RetryServiceHandler{}
	s.ServiceControlServer.OverrideQuotaHandler(&handler)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		method                  string
		token                   string
		sleepTimes              int32
		sleepLengthMs           int32
		wantHandlerRequestCount int32
	}{
		{
			desc:                    "The timeout length is longer than the sleep time of handler so the proxy did 3 times quota requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           200,
			method:                  "/v1/shelves?key=api-key-0",
			token:                   testdata.FakeCloudTokenMultiAudiences,
			wantHandlerRequestCount: 3,
		},
		{
			desc:                    "The timeout length is shorter than the sleep time of handler so the proxy did 1 times quota requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           0,
			method:                  "/v1/shelves/200?key=api-key-1",
			token:                   testdata.FakeCloudTokenMultiAudiences,
			wantHandlerRequestCount: 1,
		},
	}

	for _, tc := range tests {
		handler.RequestCount = 0
		handler.SleepTimes = tc.sleepTimes
		handler.SleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

		// Quota is unblocked and wait it to be flushed once after 1s.
		time.Sleep(time.Millisecond * 2000)
		if handler.RequestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected quota request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.RequestCount)
		}
	}
}

func TestServiceControlReportRetry(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{
		"--service=" + serviceName,
		"--service_config_id=" + configID,

		"--rollout_strategy=fixed",
		// Number of times our filter will retry the report request
		"--service_control_report_retries=2",
		// How long each report request waits before timing out (and possibly being retried)
		"--service_control_report_timeout_ms=500",
	}
	s := env.NewTestEnv(platform.TestServiceControlReportRetry, platform.GrpcBookstoreSidecar)

	handler := utils.RetryServiceHandler{}
	s.ServiceControlServer.OverrideReportHandler(&handler)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		method                  string
		sleepTimes              int32
		sleepLengthMs           int32
		wantHandlerRequestCount int32
	}{
		{
			desc:                    "The proxy will retry the report request 3 times and get the response in the last request",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           750, // The handler will sleep longer than the report timeout for the first two requests
			method:                  "/v1/shelves?key=api-key-0",
			wantHandlerRequestCount: 3,
		},
		{
			desc:                    "The proxy will do a check report once and get a response with no retries",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           100, // The handler will respond back before the report timeout in the first request
			method:                  "/v1/shelves/200?key=api-key-1",
			wantHandlerRequestCount: 1,
		},
	}

	for _, tc := range tests {
		handler.RequestCount = 0
		handler.SleepTimes = tc.sleepTimes
		handler.SleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

		// Report is unblocked and wait it to be flushed for 1 second after call to handler are made.
		// TODO(taoxuy): add customized aggregation options
		time.Sleep(time.Millisecond * 3000)
		if handler.RequestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected report request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.RequestCount)
		}
	}
}
