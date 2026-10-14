// Copyright 2026 Google LLC
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
	"net"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestTracingOtlpEndpointExclusive(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestTracingOtlpEndpoint, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtlpEndpointEnv(fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().FakeStackdriverPort))
	s.SetOmitTracingStackdriverAddress(true)

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, nil); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	time.Sleep(5 * time.Second)
	spans, err := s.FakeStackdriverServer.RetrieveSpanNames()
	if err != nil {
		t.Fatalf("RetrieveSpanNames failed: %v", err)
	}
	if len(spans) == 0 {
		t.Errorf("expected spans on FakeStackdriverServer, got 0")
	}
}

func TestTracingOtlpEndpointPrecedence(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestTracingOtlpEndpointPrecedence, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)

	// Collector B on an ephemeral port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	portB := uint16(lis.Addr().(*net.TCPAddr).Port)
	_ = lis.Close()

	collectorB := comp.NewFakeStackdriver()
	collectorB.StartStackdriverServer(portB)
	defer collectorB.StopAndWait()

	// Collector A (env var) should receive spans.
	s.SetOtlpEndpointEnv(fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().FakeStackdriverPort))

	// Collector B is pointed to by the legacy flag.
	confArgs := append([]string{
		fmt.Sprintf("--tracing_stackdriver_address=dns:%v:%v", platform.GetLoopbackAddress(), portB),
	}, utils.CommonArgs()...)
	s.SetOmitTracingStackdriverAddress(true)

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	if err := s.Setup(confArgs); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, nil); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	time.Sleep(5 * time.Second)

	// Collector A must receive spans.
	spansA, err := s.FakeStackdriverServer.RetrieveSpanNames()
	if err != nil {
		t.Fatalf("Collector A RetrieveSpanNames failed: %v", err)
	}
	if len(spansA) == 0 {
		t.Errorf("expected spans on Collector A, got 0")
	}

	// Collector B must receive 0 spans.
	spansB, err := collectorB.RetrieveSpanNames()
	if err != nil {
		t.Fatalf("Collector B RetrieveSpanNames failed: %v", err)
	}
	if len(spansB) != 0 {
		t.Errorf("expected 0 spans on Collector B, got %v", spansB)
	}
}

func TestTracingOtlpEndpointFallback(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestTracingOtlpEndpointFallback, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	// Do not set OTEL_EXPORTER_OTLP_ENDPOINT; fallback to default --tracing_stackdriver_address.

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, nil); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	time.Sleep(5 * time.Second)
	spans, err := s.FakeStackdriverServer.RetrieveSpanNames()
	if err != nil {
		t.Fatalf("RetrieveSpanNames failed: %v", err)
	}
	if len(spans) == 0 {
		t.Errorf("expected spans on FakeStackdriverServer, got 0")
	}
}

func TestTracingOtlpEndpointSchemeVariations(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc           string
		testId         uint16
		endpointFormat string
	}{
		{
			desc:           "Scheme: https://",
			testId:         platform.TestTracingOtlpEndpointSchemeHttps,
			endpointFormat: "https://%v:%v",
		},
		{
			desc:           "Scheme: bare host:port (no scheme)",
			testId:         platform.TestTracingOtlpEndpointSchemeBare,
			endpointFormat: "%v:%v",
		},
		{
			desc:           "Scheme: dns: prefix",
			testId:         platform.TestTracingOtlpEndpointSchemeDns,
			endpointFormat: "dns:%v:%v",
		},
	}

	for _, tc := range testData {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			s := env.NewTestEnv(tc.testId, platform.EchoSidecar)
			s.SetupFakeTraceServer(1.0)
			endpoint := fmt.Sprintf(tc.endpointFormat, platform.GetLoopbackAddress(), s.Ports().FakeStackdriverPort)
			s.SetOtlpEndpointEnv(endpoint)
			s.SetOmitTracingStackdriverAddress(true)

			defer func() {
				drainSpans(s)
				s.TearDown(t)
			}()

			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env: %v", err)
			}

			url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, nil); err != nil {
				t.Fatalf("fail to make call to backend: %v", err)
			}

			time.Sleep(5 * time.Second)
			spans, err := s.FakeStackdriverServer.RetrieveSpanNames()
			if err != nil {
				t.Fatalf("RetrieveSpanNames failed: %v", err)
			}
			if len(spans) == 0 {
				t.Errorf("expected spans on FakeStackdriverServer for %s, got 0", tc.desc)
			}
		})
	}
}
