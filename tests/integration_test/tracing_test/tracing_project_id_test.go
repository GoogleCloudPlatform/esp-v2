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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func makeExpectedProjectReport(expectedProjectId, targetTraceId string) []interface{} {
	return []interface{}{
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
			Trace:             "projects/" + expectedProjectId + "/traces/" + targetTraceId,
		},
	}
}

func TestTracingProjectIdLegacyFlag(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdLegacyFlag, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	confArgs := append([]string{"--tracing_project_id=legacy-test-project"}, utils.CommonArgs()...)
	if err := s.Setup(confArgs); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, incomingHeaders); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	wantScRequests := makeExpectedProjectReport("legacy-test-project", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "Legacy flag fallback")
}

func TestTracingProjectIdOtelEnv(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdOtelEnv, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtelResourceAttributesEnv("gcp.project.id=otel-custom-project,service.name=my-echo-service")

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, incomingHeaders); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	wantScRequests := makeExpectedProjectReport("otel-custom-project", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "OTel resource attributes")
}

func TestTracingProjectIdPrecedence(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdPrecedence, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtelResourceAttributesEnv("gcp.project.id=otel-override-project")

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	confArgs := append([]string{"--tracing_project_id=legacy-ignored-project"}, utils.CommonArgs()...)
	if err := s.Setup(confArgs); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, incomingHeaders); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	wantScRequests := makeExpectedProjectReport("otel-override-project", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "OTel env var overrides legacy flag")
}

func TestTracingProjectIdDefaultFallback(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdDefaultFallback, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	// Neither OTEL_RESOURCE_ATTRIBUTES nor --tracing_project_id is set.

	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, incomingHeaders); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	wantScRequests := makeExpectedProjectReport(comp.FakeProjectID, targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "Default project ID fallback")
}
