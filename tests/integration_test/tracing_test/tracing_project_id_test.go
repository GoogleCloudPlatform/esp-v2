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
	s.SetOtelResourceAttributesEnv("gcp.project_id=test-project-otel,service.name=my-echo-service")

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
	wantScRequests := makeExpectedProjectReport("test-project-otel", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "Standard OTel resource attributes (gcp.project_id)")
}

func TestTracingProjectIdOtelEnvLegacyDot(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdOtelEnvLegacyDot, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtelResourceAttributesEnv("gcp.project.id=test-project-legacy-dot,service.name=my-echo-service")

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
	wantScRequests := makeExpectedProjectReport("test-project-legacy-dot", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "Legacy OTel resource attributes (gcp.project.id)")
}

func TestTracingProjectIdPrecedence(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdPrecedence, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtelResourceAttributesEnv("gcp.project_id=otel-override-project")

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

func TestTracingProjectIdPrecedenceUnderscoreOverDot(t *testing.T) {
	t.Parallel()

	targetTraceId := "0af7651916cd43dd8448eb211c80319c"
	incomingHeaders := map[string]string{
		"traceparent": createTraceparentContext(targetTraceId, "b7ad6b7169203331"),
	}

	s := env.NewTestEnv(platform.TestTracingProjectIdPrecedenceUnderscoreOverDot, platform.EchoSidecar)
	s.SetupFakeTraceServer(1.0)
	s.SetOtelResourceAttributesEnv("gcp.project.id=dot-project,gcp.project_id=underscore-project")

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
	wantScRequests := makeExpectedProjectReport("underscore-project", targetTraceId)
	utils.CheckScRequest(t, scRequests, wantScRequests, "gcp.project_id overrides gcp.project.id and legacy flag")
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
	// ESPv2 discovers project ID from mock metadata server.

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
	utils.CheckScRequest(t, scRequests, wantScRequests, "Default project ID fallback via metadata server")

	// Verify that the metadata server was indeed queried for project ID
	if reqCnt := s.MockMetadataServer.GetReqCnt(util.ProjectIDPath); reqCnt < 1 {
		t.Errorf("MockMetadataServer received %d requests for %s, want at least 1", reqCnt, util.ProjectIDPath)
	}
}

func TestTracingProjectIdNonGcpBypass(t *testing.T) {
	t.Parallel()

	customSa, err := utils.NewServiceAccountForTest()
	if err != nil {
		t.Fatalf("failed to create service account for test: %v", err)
	}
	defer customSa.MockTokenServer.Close()

	s := env.NewTestEnv(platform.TestTracingProjectIdNonGcpBypass, platform.EchoSidecar)
	// Tracing is disabled on non-GCP when no tracing project ID is specified.
	defer s.TearDown(t)

	confArgs := append([]string{
		"--non_gcp",
		"--service_account_key=" + customSa.FileName,
		"--disable_tracing",
	}, utils.CommonArgs()...)

	if err := s.Setup(confArgs); err != nil {
		t.Fatalf("fail to setup test env: %v", err)
	}

	url := fmt.Sprintf("http://%v:%v/echo/nokey", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	if _, err := client.DoWithHeaders(url, "POST", `{"message":"hello"}`, nil); err != nil {
		t.Fatalf("fail to make call to backend: %v", err)
	}

	// Verify that metadata server was never queried for project ID
	if reqCnt := s.MockMetadataServer.GetReqCnt(util.ProjectIDPath); reqCnt != 0 {
		t.Errorf("MockMetadataServer received %d requests for %s under --non_gcp, want 0", reqCnt, util.ProjectIDPath)
	}
	if totalReq := s.MockMetadataServer.GetTotalReqCnt(); totalReq != 0 {
		t.Errorf("MockMetadataServer received %d total requests under --non_gcp, want 0", totalReq)
	}
}
