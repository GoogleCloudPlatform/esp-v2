// Copyright 2019 Google Cloud Platform Proxy Authors
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

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/env"

	bsclient "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

type retryServiceHandler struct {
	m             *comp.MockServiceCtrl
	requestCount  int32
	sleepTimes    int32
	sleepLengthMs int
}

func (h *retryServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requestCount += 1
	if h.requestCount <= h.sleepTimes {
		time.Sleep(time.Millisecond * time.Duration(h.sleepLengthMs))
	}

	w.Write([]byte(""))
}

func TestServiceControlCheckRetry(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--service_control_check_retries=2", "--service_control_check_timeout_ms=100"}
	s := env.NewTestEnv(comp.TestServiceControlCheckRetry, "bookstore", nil)
	handler := retryServiceHandler{
		m: s.ServiceControlServer,
	}
	s.ServiceControlServer.OverrideCheckHandler(&handler)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		method                  string
		sleepTimes              int32
		sleepLengthMs           int
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
			wantHandlerRequestCount: 3,
			wantError:               `500 Internal Server Error, INTERNAL:Failed to call service control`,
		},
		{
			desc:                    "Backend responsive, the proxy will retry the check request 3 times and get the response in the last request",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              2,
			sleepLengthMs:           200, // The handler will sleep too long twice, so envoy will retry these requests
			method:                  "/v1/shelves?key=api-key-1",
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
			wantHandlerRequestCount: 1,
			wantResp:                `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
	}

	for _, tc := range tests {
		handler.requestCount = 0
		handler.sleepTimes = tc.sleepTimes
		handler.sleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)
		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}

		if handler.requestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected report request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.requestCount)
		}
	}
}

func TestServiceControlQuotaRetry(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--service_control_quota_retries=2", "--service_control_quota_timeout_ms=100"}
	s := env.NewTestEnv(comp.TestServiceControlQuotaRetry, "bookstore", nil)
	s.OverrideQuota(&conf.Quota{
		MetricRules: []*conf.MetricRule{
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
	handler := retryServiceHandler{
		m: s.ServiceControlServer,
	}
	s.ServiceControlServer.OverrideQuotaHandler(&handler)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		method                  string
		sleepTimes              int32
		sleepLengthMs           int
		wantHandlerRequestCount int32
	}{
		{
			desc:                    "The timeout length is longer than the sleep time of handler so the proxy did 3 times quota requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           200,
			method:                  "/v1/shelves?key=api-key-0",
			wantHandlerRequestCount: 3,
		},
		{
			desc:                    "The timeout length is shorter than the sleep time of handler so the proxy did 1 times quota requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           0,
			method:                  "/v1/shelves/200?key=api-key-1",
			wantHandlerRequestCount: 1,
		},
	}

	for _, tc := range tests {
		handler.requestCount = 0
		handler.sleepTimes = tc.sleepTimes
		handler.sleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

		// Quota is unblocked and wait it to be flushed once after 1s.
		time.Sleep(time.Millisecond * 2000)
		if handler.requestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected quota request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.requestCount)
		}
	}
}

func TestServiceControlReportRetry(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--service_control_report_retries=2", "--service_control_report_timeout_ms=100"}
	s := env.NewTestEnv(comp.TestServiceControlReportRetry, "bookstore", nil)

	handler := retryServiceHandler{
		m: s.ServiceControlServer,
	}
	s.ServiceControlServer.OverrideReportHandler(&handler)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	tests := []struct {
		desc                    string
		clientProtocol          string
		httpMethod              string
		method                  string
		sleepTimes              int32
		sleepLengthMs           int
		wantHandlerRequestCount int32
	}{
		{
			desc:                    "The timeout length is shorter than the sleep time of handler so the proxy did 3 times report requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           200,
			method:                  "/v1/shelves?key=api-key-0",
			wantHandlerRequestCount: 3,
		},
		{
			desc:                    "The timeout length is longer than the sleep time of handler so the proxy did 1 times report requests",
			clientProtocol:          "http",
			httpMethod:              "GET",
			sleepTimes:              3,
			sleepLengthMs:           0,
			method:                  "/v1/shelves/200?key=api-key-1",
			wantHandlerRequestCount: 1,
		},
	}

	for _, tc := range tests {
		handler.requestCount = 0
		handler.sleepTimes = tc.sleepTimes
		handler.sleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

		// Report is unblocked and wait it to be flushed once after 1s.
		// TODO(taoxuy): add customized aggregation options
		time.Sleep(time.Millisecond * 2000)
		if handler.requestCount != tc.wantHandlerRequestCount {
			t.Errorf("Test (%s): failed, expected report request count: %v, got: %v", tc.desc, tc.wantHandlerRequestCount, handler.requestCount)
		}
	}
}
