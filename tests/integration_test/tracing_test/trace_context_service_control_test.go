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

package tracing_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestTraceContextPropagationHeadersForScCheck(t *testing.T) {
	t.Parallel()

	traceId := "0af7651916cd43dd8448eb211c80319c"
	spanId := "b7ad6b7169203331"
	incomingTraceContexts := map[string][]string{
		"traceparent": {
			createTraceparentContext(traceId, spanId),
		},
	}
	expectedTraceContexts := map[string][]string{
		// Only the trace id is checked. Span id should be changed.
		// By default, both trace contexts are generated.
		"Traceparent": {
			createTraceparentContextPrefix(traceId),
		},
		"X-Cloud-Trace-Context": {
			createCloudTraceContextPrefix(traceId),
		},
	}

	tests := []struct {
		desc                 string
		testId               uint16
		tracingSampleRate    float32
		expectedScReqHeaders map[string][]string
	}{
		{
			desc:                 "SC Check receives trace context propagation header.",
			testId:               platform.TestTraceContextPropagationHeadersForScCheck,
			tracingSampleRate:    1,
			expectedScReqHeaders: expectedTraceContexts,
		},
		{
			desc:                 "Trace context is propagated even when sampling rate is 0.",
			testId:               210,
			tracingSampleRate:    0,
			expectedScReqHeaders: expectedTraceContexts,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(tc.testId, platform.GrpcBookstoreSidecar)
			s.SetupFakeTraceServer(tc.tracingSampleRate)

			handler := utils.ExpectHeaderHandler{
				T:               t,
				ExpectedHeaders: tc.expectedScReqHeaders,
			}
			s.ServiceControlServer.OverrideCheckHandler(&handler)

			defer s.TearDown(t)
			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			_, err := bsclient.MakeCall("http", addr, "GET", "/v1/shelves?key=api-key-2", testdata.FakeCloudTokenLongClaims, incomingTraceContexts)
			if err != nil {
				t.Errorf("expected no err, got err: %v", err)
				return
			}

			if handler.RequestCount != 1 {
				t.Errorf("SC Check was expected to be called once, but it was called %v times.", handler.RequestCount)
				return
			}

			// Ignore the spans in this test, we do not check the names.
			time.Sleep(5 * time.Second)
			_, _ = s.FakeStackdriverServer.RetrieveSpanNames()
		})
	}
}

func TestReportTraceId(t *testing.T) {
	t.Parallel()

	traceparentTraceId := "0af7651916cd43dd8448eb211c80319c"
	traceparentSpanId := "b7ad6b7169203331"
	incomingTraceContexts := map[string]string{
		"traceparent": createTraceparentContext(traceparentTraceId, traceparentSpanId),
	}

	testData := []struct {
		desc              string
		testId            uint16
		tracingSampleRate float32
		wantScRequests    []interface{}
	}{
		{
			desc:              "Trace ID is extracted from the incoming trace context and placed in the SC Report.",
			testId:            platform.TestReportTraceId,
			tracingSampleRate: 1,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:        "1.0.0",
					ApiKeyState:       "NOT CHECKED",
					ProducerProjectID: "producer-project",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
					Trace:             "projects/" + comp.FakeProjectID + "/traces/" + traceparentTraceId,
				},
			},
		},
		{
			desc:              "Trace ID is in SC Report even when requests are not sampled.",
			testId:            212,
			tracingSampleRate: 0,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:        "1.0.0",
					ApiKeyState:       "NOT CHECKED",
					ProducerProjectID: "producer-project",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
					Trace:             "projects/" + comp.FakeProjectID + "/traces/" + traceparentTraceId,
				},
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			s := env.NewTestEnv(tc.testId, platform.EchoSidecar)
			s.SetupFakeTraceServer(tc.tracingSampleRate)
			defer s.TearDown(t)
			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo/nokey", "")
			_, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, incomingTraceContexts)
			if err != nil {
				t.Fatalf("fail to make call to backend: %v", err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("GetRequests returns error: %v", err)
			}
			utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)

			// Ignore the spans in this test, we do not check the names.
			time.Sleep(5 * time.Second)
			_, _ = s.FakeStackdriverServer.RetrieveSpanNames()
		})
	}
}
