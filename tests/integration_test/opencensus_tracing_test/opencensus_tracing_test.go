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

package opencensus_tracing_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
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

func checkWantSpans(env *env.TestEnv, wantSpanNames []string) error {

	// Check that we received each expected span
	for _, wantName := range wantSpanNames {

		select {
		case span := <-env.FakeStackdriverServer.RcvSpan:

			// Check name
			if wantName != span.DisplayName.Value {
				return fmt.Errorf("expected span name: %s, got span with name: %s", wantName, span.DisplayName.Value)
			}

			// Check attributes
			if len(span.Attributes.AttributeMap) == 0 {
				return fmt.Errorf("expected span %s to have more than 0 attributes attached to it", wantName)
			}

			// Check for project id
			if !strings.Contains(span.Name, comp.FakeProjectID) {
				return fmt.Errorf("expected span %s to have the project id in its name, but got name: %s", wantName, span.Name)
			}

		// Prevents test from being frozen if envoy fails to create spans
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout on waiting for Stackdriver tracing server to receive spans, expected span name: %s", wantName)
		}
	}

	// Ensure we didn't receive any extra spans
	select {
	case span := <-env.FakeStackdriverServer.RcvSpan:
		return fmt.Errorf("received span name: %s, was not expecting any more spans", span.DisplayName)

	case <-time.After(1 * time.Second):
		// Successful, no more extra spans
		return nil
	}
}

func TestServiceControlCheckTracesWithRetry(t *testing.T) {
	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--backend_protocol=grpc",
		"--rollout_strategy=fixed",
		"--service_control_check_retries=2",
		"--service_control_check_timeout_ms=100",
	}
	s := env.NewTestEnv(comp.TestServiceControlCheckTracesWithRetry, "bookstore")
	s.SetupFakeTraceServer()
	handler := retryServiceHandler{
		m: s.ServiceControlServer,
	}
	s.ServiceControlServer.OverrideCheckHandler(&handler)
	defer s.TearDown()
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
		sleepLengthMs  int
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
		handler.requestCount = 0
		handler.sleepTimes = tc.sleepTimes
		handler.sleepLengthMs = tc.sleepLengthMs

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		_, _ = bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

		if err := checkWantSpans(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}

func TestServiceControlSkipUsageTraces(t *testing.T) {
	configId := "test-config-id"

	args := []string{
		"--service_config_id=" + configId,
		"--backend_protocol=http1",
		"--rollout_strategy=fixed",
		"--suppress_envoy_headers",
	}

	s := env.NewTestEnv(comp.TestServiceControlSkipUsageTraces, "echo")
	s.SetupFakeTraceServer()
	s.AppendUsageRules(
		[]*confpb.UsageRule{
			{
				Selector:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				SkipServiceControl: true,
			},
		},
	)
	defer s.TearDown()
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

		if err := checkWantSpans(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}

func TestFetchingJwksTraces(t *testing.T) {

	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--backend_protocol=grpc",
		"--rollout_strategy=fixed",
	}

	s := env.NewTestEnv(comp.TestAsymmetricKeysTraces, "bookstore")

	s.SetupFakeTraceServer()
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
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	time.Sleep(time.Duration(5 * time.Second))
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

		if err := checkWantSpans(s, tc.wantSpanNames); err != nil {
			t.Errorf("Test (%s) failed: %v", tc.desc, err)
		}
	}
}
