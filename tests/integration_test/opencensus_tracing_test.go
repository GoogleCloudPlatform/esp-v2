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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
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
	s := env.NewTestEnv(platform.TestTracesServiceControlCheckWithRetry, platform.GrpcBookstoreSidecar)
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
				"ingress ListShelves",
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
				"router backend-cluster-bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress ListShelves",
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
				"router backend-cluster-bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress ListShelves",
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

	s := env.NewTestEnv(platform.TestTracesServiceControlSkipUsage, platform.EchoSidecar)
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
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Simplegetcors",
			},
		},
		{
			desc:    "succeed, the api with SkipServiceControl set true will skip service control",
			url:     fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:  "POST",
			message: "hello",
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
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

	s := env.NewTestEnv(platform.TestTracesFetchingJwks, platform.GrpcBookstoreSidecar)

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
				"ingress ListShelves",
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
				"router backend-cluster-bookstore.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress ListShelves",
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
			s := env.NewTestEnv(platform.TestTracingSampleRate, platform.GrpcBookstoreSidecar)
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

func TestTracesDynamicRouting(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	args := []string{
		"--service_config_id=" + configId,
		"--rollout_strategy=fixed",
		"--suppress_envoy_headers",
	}

	s := env.NewTestEnv(platform.TestTracesDynamicRouting, platform.EchoRemote)
	s.SetupFakeTraceServer(1)
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
			desc:   "method name is present in span for remote backend routes",
			url:    fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/pet/1/num/2", ""),
			method: util.GET,
			wantSpanNames: []string{
				fmt.Sprintf("router backend-cluster-localhost:%s egress", strconv.Itoa(int(s.Ports().DynamicRoutingBackendPort))),
				"ingress dynamic_routing_GetPetById",
			},
		},
		{
			desc:   "unknown operation has no method name",
			url:    fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/random/path", ""),
			method: util.GET,
			wantSpanNames: []string{
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

func TestTraceContextPropagationHeaders(t *testing.T) {
	t.Parallel()

	// Some real-world examples.
	incomingTraceContexts := map[string]string{
		"traceparent":           "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
	}

	testData := []struct {
		desc          string
		confArgs      []string
		requestHeader map[string]string
		// Headers wanted in the response.
		wantRespHeaders map[string]string
		// Headers that should not exist in the response.
		notWantRespHeaders map[string]string
	}{
		{
			desc: "trace context propagation is disabled, all headers are preserved",
			confArgs: append([]string{
				"--tracing_incoming_context=",
				"--tracing_outgoing_context=",
			}, utils.CommonArgs()...),
			requestHeader: incomingTraceContexts,
			wantRespHeaders: map[string]string{
				// All headers are not changed.
				"Echo-Traceparent":           "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
				"Echo-X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
		},
		{
			desc: "traceparent context propagation is enabled, the trace id is preserved",
			confArgs: append([]string{
				"--tracing_incoming_context=traceparent",
				"--tracing_outgoing_context=traceparent",
			}, utils.CommonArgs()...),
			requestHeader: incomingTraceContexts,
			wantRespHeaders: map[string]string{
				// Trace id is maintained. Span id is changed, so it's not checked.
				"Echo-Traceparent": "00-0af7651916cd43dd8448eb211c80319c-",
				// All other headers are not changed.
				"Echo-X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
			notWantRespHeaders: map[string]string{
				// The span id should have changed.
				"Echo-Traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			},
		},
		{
			desc: "x-cloud-trace-context context propagation is enabled, the trace id is preserved",
			confArgs: append([]string{
				"--tracing_incoming_context=x-cloud-trace-context",
				"--tracing_outgoing_context=x-cloud-trace-context",
			}, utils.CommonArgs()...),
			requestHeader: incomingTraceContexts,
			wantRespHeaders: map[string]string{
				// Trace id is maintained. Span id is changed, so it's not checked.
				"Echo-X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/",
				// All other headers are not changed.
				"Echo-Traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			},
			notWantRespHeaders: map[string]string{
				// The span id should have changed.
				"Echo-X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
		},
		{
			desc: "traceparent context propagation is enabled for outgoing only, so the incoming header is fully overwritten",
			confArgs: append([]string{
				"--tracing_incoming_context=",
				"--tracing_outgoing_context=traceparent",
			}, utils.CommonArgs()...),
			requestHeader: incomingTraceContexts,
			wantRespHeaders: map[string]string{
				// Trace id and span id are changed, so they not checked.
				"Echo-Traceparent": "00-",
				// All other headers are not changed.
				"Echo-X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
			notWantRespHeaders: map[string]string{
				// The trace id and span id should have changed.
				"Echo-Traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			},
		},
		// grpc-trace-bin is not tested, it's difficult to test.
		// We should be safe, opencensus has tests for this.
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			s := env.NewTestEnv(platform.TestTraceContextPropagationHeaders, platform.EchoRemote)
			s.SetupFakeTraceServer(1)
			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoHeader", "")
			headers, _, err := utils.DoWithHeaders(url, util.GET, "", tc.requestHeader)
			if err != nil {
				t.Fatalf("fail to make call to backend: %v", err)
			}

			for wantHeaderName, wantHeaderVal := range tc.wantRespHeaders {
				if !utils.CheckHeaderExist(headers, wantHeaderName, func(gotHeaderVal string) bool {
					return strings.Contains(gotHeaderVal, wantHeaderVal)
				}) {
					t.Errorf("got headers %+q, \ndid not find expected header %s = %s,  ", headers, wantHeaderName, wantHeaderVal)
				}
			}

			for notWantHeaderName, notWantHeaderVal := range tc.notWantRespHeaders {
				if utils.CheckHeaderExist(headers, notWantHeaderName, func(gotHeaderVal string) bool {
					return strings.Contains(gotHeaderVal, notWantHeaderVal)
				}) {
					t.Errorf("got headers %+q, \nfound header %s = %s, but did not want it", headers, notWantHeaderName, notWantHeaderVal)
				}
			}

			// Ignore the spans in this test, we do not check the names.
			time.Sleep(5 * time.Second)
			_, _ = s.FakeStackdriverServer.RetrieveSpanNames()
		})
	}
}
