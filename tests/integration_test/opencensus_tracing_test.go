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
	"reflect"
	"testing"
	"time"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func checkSpanNames(env *env.TestEnv, wantSpanNames []string) error {
	time.Sleep(5 * time.Second)

	gotSpanNames, err := env.FakeStackdriverServer.RetrieveSpanNames()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(gotSpanNames, wantSpanNames) {
		return fmt.Errorf("got span names: %+q, want span names: %+q", gotSpanNames, wantSpanNames)
	}

	return nil
}

func TestTracesServiceControlCheckWithRetry(t *testing.T) {
	t.Parallel()
	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--rollout_strategy=fixed",
		"--service_control_check_retries=2",
		"--service_control_check_timeout_ms=100",
	}
	s := env.NewTestEnv(comp.TestTracesServiceControlCheckWithRetry, platform.GrpcBookstoreSidecar)
	s.SetupFakeTraceServer(1)
	handler := utils.RetryServiceHandler{}
	s.ServiceControlServer.OverrideCheckHandler(&handler)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc           string
		clientProtocol string
		httpMethod     string
		method         string
		token          string
		sleepTimes     int32
		sleepLengthMs  int32
		wantSpanNames  []string
	}{
		{
			desc:           "Backend unresponsive, the proxy will retry the check request 3 times and fail",
			clientProtocol: "http",
			httpMethod:     "GET",
			sleepTimes:     3,
			sleepLengthMs:  200,
			method:         "/v1/shelves?key=api-key-0",
			token:          testdata.FakeCloudTokenMultiAudiences,
			wantSpanNames: []string{
				"JWT Remote PubKey Fetch", // The first request will result in a JWT call
				"Service Control remote call: Check",
				"Service Control remote call: Check - Retry 1",
				"Service Control remote call: Check - Retry 2",
				"ingress",
			},
		},
		{
			desc:           "Backend responsive, the proxy will retry the check request 3 times and get the response in the last request",
			clientProtocol: "http",
			httpMethod:     "GET",
			sleepTimes:     2,
			sleepLengthMs:  200, // The handler will sleep too long twice, so envoy will retry these requests
			method:         "/v1/shelves?key=api-key-1",
			token:          testdata.FakeCloudTokenMultiAudiences,
			wantSpanNames: []string{
				"Service Control remote call: Check",
				"Service Control remote call: Check - Retry 1",
				"Service Control remote call: Check - Retry 2",
				"router bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress",
			},
		},
		{
			desc:           "Backend responsive, the proxy will do a check request once and get a response with no retries",
			clientProtocol: "http",
			httpMethod:     "GET",
			sleepTimes:     3,
			sleepLengthMs:  0, // The handler will not sleep, so envoy's request to the backend should be successful
			method:         "/v1/shelves?key=api-key-2",
			token:          testdata.FakeCloudTokenMultiAudiences,
			wantSpanNames: []string{
				"Service Control remote call: Check",
				"router bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress",
			},
		},
	}

	for _, tc := range tests {
		handler.RequestCount = 0
		handler.SleepTimes = tc.sleepTimes
		handler.SleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

		if err := checkSpanNames(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}

func TestTracesServiceControlSkipUsage(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	args := []string{
		"--service_config_id=" + configId,
		"--rollout_strategy=fixed",
		"--suppress_envoy_headers",
	}

	s := env.NewTestEnv(comp.TestTracesServiceControlSkipUsage, platform.EchoSidecar)
	s.SetupFakeTraceServer(1)
	s.AppendUsageRules(
		[]*confpb.UsageRule{
			{
				Selector:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				SkipServiceControl: true,
			},
		},
	)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		url           string
		method        string
		requestHeader map[string]string
		message       string
		wantSpanNames []string
	}{
		{
			desc:   "succeed, just show the service control works for normal request",
			url:    fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/simplegetcors", "?key=api-key"),
			method: "GET",
			wantSpanNames: []string{
				"Service Control remote call: Check",
				"router echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress",
			},
		},
		{
			desc:    "succeed, the api with SkipServiceControl set true will skip service control",
			url:     fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:  "POST",
			message: "hello",
			wantSpanNames: []string{
				"router echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress",
			},
		},
	}
	for _, tc := range testData {
		_, _ = client.DoWithHeaders(tc.url, tc.method, tc.message, tc.requestHeader)

		if err := checkSpanNames(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}

func TestTracesFetchingJwks(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--rollout_strategy=fixed",
	}

	s := env.NewTestEnv(comp.TestTracesFetchingJwks, platform.GrpcBookstoreSidecar)

	s.SetupFakeTraceServer(1)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	time.Sleep(5 * time.Second)
	tests := []struct {
		desc           string
		clientProtocol string
		httpMethod     string
		method         string
		queryInToken   bool
		token          string
		headers        map[string][]string
		wantSpanNames  []string
	}{
		{
			desc:           "Failed, no JWT passed in.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantSpanNames: []string{
				"ingress",
			},
		},
		{
			desc:           "Succeeded, with right token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Es256Token,
			wantSpanNames: []string{
				"JWT Remote PubKey Fetch",
				"Service Control remote call: Check",
				"router bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress",
			},
		},
	}

	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		if tc.queryInToken {
			_, _ = bsclient.MakeTokenInQueryCall(addr, tc.httpMethod, tc.method, tc.token)
		} else {
			_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)
		}

		if err := checkSpanNames(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}

func TestTracingSampleRate(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--rollout_strategy=fixed",
	}

	time.Sleep(5 * time.Second)
	tests := []struct {
		desc              string
		clientProtocol    string
		httpMethod        string
		tracingSampleRate float32
		numRequests       int
		numWantSpansMin   int
		numWantSpansMax   int
	}{
		{
			desc:              "A single request with sample rate 1.0 has 1 span",
			clientProtocol:    "http",
			httpMethod:        "GET",
			tracingSampleRate: 1,
			numRequests:       1,
			numWantSpansMin:   1,
			numWantSpansMax:   1,
		},
		{
			desc:              "20 requests with sample rate 0.0 has 0 spans",
			clientProtocol:    "http",
			httpMethod:        "GET",
			tracingSampleRate: 0,
			numRequests:       20,
			numWantSpansMin:   0,
			numWantSpansMax:   0,
		},
		{
			// Don't make too many requests, as Envoy will batch writes with multiple minutes of delay.
			// Binomial distribution tells us this test has < 0.3% chance of a false negative.
			desc:              "10 requests with sample rate 0.1 has [0, 4] spans",
			clientProtocol:    "http",
			httpMethod:        "GET",
			tracingSampleRate: 0.1,
			numRequests:       10,
			numWantSpansMin:   0,
			numWantSpansMax:   4,
		},
		{
			// Don't make too many requests, as Envoy will batch writes with multiple minutes of delay.
			// Binomial distribution tells us this test has < 0.3% chance of a false negative.
			desc:              "5 requests with sample rate 0.9 has [2, 5] spans",
			clientProtocol:    "http",
			httpMethod:        "GET",
			tracingSampleRate: 0.9,
			numRequests:       5,
			numWantSpansMin:   2,
			numWantSpansMax:   5,
		},
	}

	for _, tc := range tests {
		// Place in closure to allow deferring in loop.
		func() {
			s := env.NewTestEnv(comp.TestTracingSampleRate, platform.GrpcBookstoreSidecar)
			s.SetupFakeTraceServer(tc.tracingSampleRate)

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			// Use a path that results in 404, so only 1 ingress span is created per request.
			path := "/v9/non-existent-path"

			for i := 0; i < tc.numRequests; i++ {
				addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
				_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, path, "", nil)
			}

			time.Sleep(5 * time.Second)
			gotSpans, err := s.FakeStackdriverServer.RetrieveSpanNames()
			if err != nil {
				t.Errorf("Test (%s) failed: %v", tc.desc, err)
			}

			numGotSpans := len(gotSpans)
			if numGotSpans < tc.numWantSpansMin || numGotSpans > tc.numWantSpansMax {
				t.Errorf("Test (%s) failed: got num spans %v, want num spans range [%v, %v]", tc.desc, numGotSpans, tc.numWantSpansMin, tc.numWantSpansMax)
			}
		}()
	}
}
