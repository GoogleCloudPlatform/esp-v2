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

// Package apigw_service_control_overlap_test contains ESPv2 integration tests
// for Service Control behavior on overlapping static and parameterized routes
// across open, API-key, JWT, and asymmetric per-verb configurations.
package apigw_service_control_overlap_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

// TestOverlappingOpenStrictServiceControl verifies strict Service Control
// reporting on overlapping static and parameterized open routes with path
// normalization disabled.
func TestOverlappingOpenStrictServiceControl(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapOpenFixed"}, {Name: "OverlapOpenParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlap/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlap/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{
			name:   "static_route",
			path:   "/dynamicrouting/overlap/fixed",
			wantOp: fixedOp,
		},
		{
			name:   "wildcard_route",
			path:   "/dynamicrouting/overlap/dynamic",
			wantOp: paramOp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			wantResp := fmt.Sprintf(`{"RequestURI": "%s"}`, tc.path)
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want JSON %q: %v", targetURL, http.MethodGet, string(resp), wantResp, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingOpenReportOnly verifies report-only Service Control behavior
// on overlapping static and parameterized open routes with path normalization
// disabled.
func TestOverlappingOpenReportOnly(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenROFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenROParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapOpenROFixed"}, {Name: "OverlapOpenROParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapro/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapro/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{
			name:   "static_route",
			path:   "/dynamicrouting/overlapro/fixed",
			wantOp: fixedOp,
		},
		{
			name:   "wildcard_route",
			path:   "/dynamicrouting/overlapro/dynamic",
			wantOp: paramOp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			wantResp := fmt.Sprintf(`{"RequestURI": "%s"}`, tc.path)
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want JSON %q: %v", targetURL, http.MethodGet, string(resp), wantResp, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingOpenBypass verifies that overlapping static and parameterized
// open routes with SkipServiceControl enabled send no Service Control requests.
func TestOverlappingOpenBypass(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBypassFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBypassParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapOpenBypassFixed"}, {Name: "OverlapOpenBypassParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbypass/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbypass/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapbypass/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapbypass/dynamic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			wantResp := fmt.Sprintf(`{"RequestURI": "%s"}`, tc.path)
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want JSON %q: %v", targetURL, http.MethodGet, string(resp), wantResp, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}
}

// TestOverlappingOpenBypassAutoCORS verifies Service Control bypass with
// auto-CORS on overlapping static and parameterized open routes.
func TestOverlappingOpenBypassAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBPACFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBPACParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapOpenBPACFixed"}, {Name: "OverlapOpenBPACParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbp/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbp/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	routes := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapbp/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapbp/dynamic"},
	}

	for _, tc := range routes {
		t.Run("get_"+tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			wantResp := fmt.Sprintf(`{"RequestURI": "%s"}`, tc.path)
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want JSON %q: %v", targetURL, http.MethodGet, string(resp), wantResp, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}

	for _, tc := range routes {
		t.Run("options_"+tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			headers, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := headers.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
}

// TestOverlappingOpenBypassCORSCollision verifies Service Control bypass when
// auto-CORS collides with explicit OPTIONS rules on overlapping static and
// parameterized open routes.
func TestOverlappingOpenBypassCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBPCollFixed"
		paramOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenBPCollParam"
		optFixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsOpenBPCollFixed"
		optParamOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsOpenBPCollParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{
			{Name: "OverlapOpenBPCollFixed"},
			{Name: "OverlapOpenBPCollParam"},
			{Name: "OptionsOpenBPCollFixed"},
			{Name: "OptionsOpenBPCollParam"},
		},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbpcoll/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapbpcoll/{param}"}},
		{Selector: optFixedOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapbpcoll/fixed"}}},
		{Selector: optParamOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapbpcoll/{param}"}}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: optFixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: optParamOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapbpcoll/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapbpcoll/dynamic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			wantResp := fmt.Sprintf(`{"RequestURI": "%s"}`, tc.path)
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want JSON %q: %v", targetURL, http.MethodGet, string(resp), wantResp, err)
			}

			headers, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := headers.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET/OPTIONS = %d, want 0", got)
	}
}

// TestOverlappingOpenProxyAuth verifies that Service Control Report requests
// for overlapping static and parameterized open routes include the proxy
// authentication header.
func TestOverlappingOpenProxyAuth(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenPAFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapOpenPAParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapOpenPAFixed"}, {Name: "OverlapOpenPAParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlappa/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlappa/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{name: "static_route", path: "/dynamicrouting/overlappa/fixed", wantOp: fixedOp},
		{name: "wildcard_route", path: "/dynamicrouting/overlappa/dynamic", wantOp: paramOp},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingAPIKeyReportOnly verifies report-only Service Control behavior
// on overlapping static and parameterized routes when API keys are optional.
func TestOverlappingAPIKeyReportOnly(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyROFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyROParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapKeyROFixed"}, {Name: "OverlapKeyROParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeyro/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeyro/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapkeyro/fixed?key=api-key-fixed", wantOp: fixedOp},
		{name: "wildcard_route", path: "/dynamicrouting/overlapkeyro/dynamic?key=api-key-param", wantOp: paramOp},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingAPIKeyBypass verifies that overlapping static and
// parameterized API-key routes with SkipServiceControl enabled send no Service
// Control requests.
func TestOverlappingAPIKeyBypass(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapKeyBPFixed"}, {Name: "OverlapKeyBPParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybp/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybp/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapkeybp/fixed?key=api-key-fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapkeybp/dynamic?key=api-key-param"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}
}

// TestOverlappingAPIKeyBypassAutoCORS verifies Service Control bypass with
// auto-CORS on overlapping static and parameterized API-key routes.
func TestOverlappingAPIKeyBypassAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPACFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPACParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapKeyBPACFixed"}, {Name: "OverlapKeyBPACParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybpac/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybpac/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	getCases := []struct {
		name string
		path string
	}{
		{name: "get_static", path: "/dynamicrouting/overlapkeybpac/fixed?key=api-key-fixed"},
		{name: "get_wildcard", path: "/dynamicrouting/overlapkeybpac/dynamic?key=api-key-param"},
	}

	for _, tc := range getCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}

	optionsCases := []struct {
		name string
		path string
	}{
		{name: "options_static", path: "/dynamicrouting/overlapkeybpac/fixed"},
		{name: "options_wildcard", path: "/dynamicrouting/overlapkeybpac/dynamic"},
	}

	for _, tc := range optionsCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			headers, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := headers.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
}

// TestOverlappingAPIKeyBypassCORSCollision verifies Service Control bypass when
// auto-CORS collides with explicit OPTIONS rules on overlapping static and
// parameterized API-key routes.
func TestOverlappingAPIKeyBypassCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPCollFixed"
		paramOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyBPCollParam"
		optFixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsKeyBPCollFixed"
		optParamOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsKeyBPCollParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{
			{Name: "OverlapKeyBPCollFixed"},
			{Name: "OverlapKeyBPCollParam"},
			{Name: "OptionsKeyBPCollFixed"},
			{Name: "OptionsKeyBPCollParam"},
		},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybpcoll/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeybpcoll/{param}"}},
		{Selector: optFixedOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapkeybpcoll/fixed"}}},
		{Selector: optParamOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapkeybpcoll/{param}"}}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: false},
			{Selector: optFixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: optParamOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	getCases := []struct {
		name string
		path string
	}{
		{name: "get_static", path: "/dynamicrouting/overlapkeybpcoll/fixed?key=api-key-fixed"},
		{name: "get_wildcard", path: "/dynamicrouting/overlapkeybpcoll/dynamic?key=api-key-param"},
	}

	for _, tc := range getCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
		})
	}

	optionsCases := []struct {
		name string
		path string
	}{
		{name: "options_static", path: "/dynamicrouting/overlapkeybpcoll/fixed"},
		{name: "options_wildcard", path: "/dynamicrouting/overlapkeybpcoll/dynamic"},
	}

	for _, tc := range optionsCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			headers, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := headers.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET/OPTIONS = %d, want 0", got)
	}
}

// TestOverlappingAPIKeyProxyAuth verifies that Service Control Check and Report
// requests for overlapping static and parameterized API-key routes include the
// proxy authentication header.
func TestOverlappingAPIKeyProxyAuth(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyPAFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapKeyPAParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapKeyPAFixed"}, {Name: "OverlapKeyPAParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeypa/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapkeypa/{param}"}},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: false},
			{Selector: paramOp, AllowUnregisteredCalls: false},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	testCases := []struct {
		name         string
		path         string
		wantOp       string
		wantConsumer string
	}{
		{
			name:         "static_route",
			path:         "/dynamicrouting/overlapkeypa/fixed?key=api-key-fixed",
			wantOp:       fixedOp,
			wantConsumer: "api_key:api-key-fixed",
		},
		{
			name:         "wildcard_route",
			path:         "/dynamicrouting/overlapkeypa/dynamic?key=api-key-param",
			wantOp:       paramOp,
			wantConsumer: "api_key:api-key-param",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(2)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(2) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				ConsumerID:    tc.wantConsumer,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
				t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingJWTAuthReportOnly verifies report-only Service Control
// behavior on overlapping static and parameterized JWT-authenticated routes.
func TestOverlappingJWTAuthReportOnly(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTROFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTROParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapJWTROFixed"}, {Name: "OverlapJWTROParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtro/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtro/{param}"}},
	})
	setupJWTForOverlap(s, fixedOp, paramOp)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences}
	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapjwtro/fixed", wantOp: fixedOp},
		{name: "wildcard_route", path: "/dynamicrouting/overlapjwtro/dynamic", wantOp: paramOp},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", headers); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingJWTAuthBypass verifies that overlapping static and
// parameterized JWT-authenticated routes with SkipServiceControl enabled send
// no Service Control requests.
func TestOverlappingJWTAuthBypass(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapJWTBPFixed"}, {Name: "OverlapJWTBPParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbp/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbp/{param}"}},
	})
	setupJWTForOverlap(s, fixedOp, paramOp)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences}
	testCases := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapjwtbp/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapjwtbp/dynamic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", headers); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}
}

// TestOverlappingJWTAuthBypassAutoCORS verifies Service Control bypass with
// auto-CORS on overlapping static and parameterized JWT-authenticated routes.
func TestOverlappingJWTAuthBypassAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPACFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPACParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapJWTBPACFixed"}, {Name: "OverlapJWTBPACParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbpac/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbpac/{param}"}},
	})
	setupJWTForOverlap(s, fixedOp, paramOp)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences}
	routes := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapjwtbpac/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapjwtbpac/dynamic"},
	}

	for _, tc := range routes {
		t.Run("get_"+tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", headers); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GETs = %d, want 0", got)
	}

	for _, tc := range routes {
		t.Run("options_"+tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			corsHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := corsHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
}

// TestOverlappingJWTAuthBypassCORSCollision verifies Service Control bypass
// when auto-CORS collides with explicit OPTIONS rules on overlapping static and
// parameterized JWT-authenticated routes.
func TestOverlappingJWTAuthBypassCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	const (
		fixedOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPCollFixed"
		paramOp    = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTBPCollParam"
		optFixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsJWTBPCollFixed"
		optParamOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsJWTBPCollParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{
			{Name: "OverlapJWTBPCollFixed"},
			{Name: "OverlapJWTBPCollParam"},
			{Name: "OptionsJWTBPCollFixed"},
			{Name: "OptionsJWTBPCollParam"},
		},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbpcoll/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtbpcoll/{param}"}},
		{Selector: optFixedOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapjwtbpcoll/fixed"}}},
		{Selector: optParamOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: "/dynamicrouting/overlapjwtbpcoll/{param}"}}},
	})
	setupJWTForOverlap(s, fixedOp, paramOp)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: paramOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: optFixedOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
			{Selector: optParamOp, SkipServiceControl: true, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences}
	testCases := []struct {
		name string
		path string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapjwtbpcoll/fixed"},
		{name: "wildcard_route", path: "/dynamicrouting/overlapjwtbpcoll/dynamic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", headers); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			corsHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodGet, "")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}
			if status != http.StatusOK && status != http.StatusNoContent {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodGet, status)
			}
			if got, want := corsHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
			}
		})
	}
	time.Sleep(100 * time.Millisecond)
	if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
		t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET/OPTIONS = %d, want 0", got)
	}
}

// TestOverlappingJWTAuthProxyAuth verifies that Service Control Report requests
// for overlapping static and parameterized JWT-authenticated routes include the
// proxy authentication header, including for 401 rejections.
func TestOverlappingJWTAuthProxyAuth(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)

	const (
		fixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTPAFixed"
		paramOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OverlapJWTPAParam"
	)
	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: []*apipb.Method{{Name: "OverlapJWTPAFixed"}, {Name: "OverlapJWTPAParam"}},
	}})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{Selector: fixedOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtpa/fixed"}},
		{Selector: paramOp, Pattern: &annotationspb.HttpRule_Get{Get: "/dynamicrouting/overlapjwtpa/{param}"}},
	})
	setupJWTForOverlap(s, fixedOp, paramOp)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: fixedOp, AllowUnregisteredCalls: true},
			{Selector: paramOp, AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(strictRawPathArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences}
	testCases := []struct {
		name   string
		path   string
		wantOp string
	}{
		{name: "static_route", path: "/dynamicrouting/overlapjwtpa/fixed", wantOp: fixedOp},
		{name: "wildcard_route", path: "/dynamicrouting/overlapjwtpa/dynamic", wantOp: paramOp},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", headers); err != nil {
				t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}

	// Also verify unauthenticated request is rejected by JWT filter (401).
	t.Run("missing_jwt", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/overlapjwtpa/fixed")
		_, status, _, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if status != http.StatusUnauthorized {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d", targetURL, http.MethodGet, status, http.StatusUnauthorized)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: fixedOp,
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusUnauthorized,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})
}

// TestOverlappingAsymmetricStrictServiceControl verifies strict Service Control
// enforcement on overlapping static and parameterized asymmetric routes (GET
// open, POST API-key) with CORS disabled.
func TestOverlappingAsymmetricStrictServiceControl(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasym",
		corsMode:           "disabled",
		serviceControlMode: "strict",
	})
}

// TestOverlappingAsymmetricAutoCORS verifies strict Service Control enforcement
// with auto-CORS on overlapping static and parameterized asymmetric routes.
func TestOverlappingAsymmetricAutoCORS(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymac",
		corsMode:           "auto_cors",
		serviceControlMode: "strict",
	})
}

// TestOverlappingAsymmetricCORSCollision verifies strict Service Control
// enforcement when auto-CORS collides with explicit OPTIONS rules on
// overlapping static and parameterized asymmetric routes.
func TestOverlappingAsymmetricCORSCollision(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymcoll",
		corsMode:           "cors_collision",
		serviceControlMode: "strict",
	})
}

// TestOverlappingAsymmetricReportOnly verifies report-only Service Control
// behavior on overlapping static and parameterized asymmetric routes with CORS
// disabled.
func TestOverlappingAsymmetricReportOnly(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymro",
		corsMode:           "disabled",
		serviceControlMode: "report_only",
	})
}

// TestOverlappingAsymmetricReportOnlyAutoCORS verifies report-only Service
// Control behavior with auto-CORS on overlapping static and parameterized
// asymmetric routes.
func TestOverlappingAsymmetricReportOnlyAutoCORS(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymroac",
		corsMode:           "auto_cors",
		serviceControlMode: "report_only",
	})
}

// TestOverlappingAsymmetricReportOnlyCORSCollision verifies report-only Service
// Control behavior when auto-CORS collides with explicit OPTIONS rules on
// overlapping static and parameterized asymmetric routes.
func TestOverlappingAsymmetricReportOnlyCORSCollision(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymrocoll",
		corsMode:           "cors_collision",
		serviceControlMode: "report_only",
	})
}

// TestOverlappingAsymmetricBypass verifies Service Control bypass on
// overlapping static and parameterized asymmetric routes with CORS disabled.
func TestOverlappingAsymmetricBypass(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymbp",
		corsMode:           "disabled",
		serviceControlMode: "bypass",
	})
}

// TestOverlappingAsymmetricBypassAutoCORS verifies Service Control bypass with
// auto-CORS on overlapping static and parameterized asymmetric routes.
func TestOverlappingAsymmetricBypassAutoCORS(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymbpac",
		corsMode:           "auto_cors",
		serviceControlMode: "bypass",
	})
}

// TestOverlappingAsymmetricBypassCORSCollision verifies Service Control bypass
// when auto-CORS collides with explicit OPTIONS rules on overlapping static and
// parameterized asymmetric routes.
func TestOverlappingAsymmetricBypassCORSCollision(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasymbpcoll",
		corsMode:           "cors_collision",
		serviceControlMode: "bypass",
	})
}

// TestOverlappingAsymmetricProxyAuth verifies Service Control proxy
// authentication headers on overlapping static and parameterized asymmetric
// routes with CORS disabled.
func TestOverlappingAsymmetricProxyAuth(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasympa",
		corsMode:           "disabled",
		serviceControlMode: "proxy_auth",
	})
}

// TestOverlappingAsymmetricProxyAuthAutoCORS verifies Service Control proxy
// authentication headers with auto-CORS on overlapping static and parameterized
// asymmetric routes.
func TestOverlappingAsymmetricProxyAuthAutoCORS(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasympaac",
		corsMode:           "auto_cors",
		serviceControlMode: "proxy_auth",
	})
}

// TestOverlappingAsymmetricProxyAuthCORSCollision verifies Service Control
// proxy authentication headers when auto-CORS collides with explicit OPTIONS
// rules on overlapping static and parameterized asymmetric routes.
func TestOverlappingAsymmetricProxyAuthCORSCollision(t *testing.T) {
	runAsymmetricOverlapTest(t, asymmetricOverlapConfig{
		prefix:             "/dynamicrouting/overlapasympacoll",
		corsMode:           "cors_collision",
		serviceControlMode: "proxy_auth",
	})
}

type asymmetricOverlapConfig struct {
	prefix string
	// One of "disabled", "auto_cors", or "cors_collision".
	corsMode string
	// One of "strict", "report_only", "bypass", or "proxy_auth".
	serviceControlMode string
}

func strictRawPathArgs() []string {
	return append([]string{"--normalize_path=false"}, utils.CommonArgs()...)
}

func setupJWTForOverlap(s *env.TestEnv, fixedSelector, paramSelector string) {
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: fixedSelector,
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.GoogleJwtProvider,
						Audiences:  "admin.cloud.goog",
					},
				},
			},
			{
				Selector: paramSelector,
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.GoogleJwtProvider,
						Audiences:  "admin.cloud.goog",
					},
				},
			},
		},
	})
}

// runAsymmetricOverlapTest verifies Service Control behavior for overlapping
// static and parameterized routes with asymmetric per-verb authentication (GET
// open, POST API-key).
func runAsymmetricOverlapTest(t *testing.T, cfg asymmetricOverlapConfig) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlOverlap, platform.EchoSidecar)
	defer s.TearDown(t)
	if cfg.corsMode == "auto_cors" || cfg.corsMode == "cors_collision" {
		s.SetAllowCors()
	}

	const (
		getFixedOp  = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetAsymFixed"
		postFixedOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.PostAsymFixed"
		getParamOp  = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetAsymParam"
		postParamOp = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.PostAsymParam"
		optFixedOp  = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsAsymFixed"
		optParamOp  = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.OptionsAsymParam"
	)

	methods := []*apipb.Method{
		{Name: "GetAsymFixed"},
		{Name: "PostAsymFixed"},
		{Name: "GetAsymParam"},
		{Name: "PostAsymParam"},
	}
	httpRules := []*annotationspb.HttpRule{
		{Selector: getFixedOp, Pattern: &annotationspb.HttpRule_Get{Get: cfg.prefix + "/fixed"}},
		{Selector: postFixedOp, Pattern: &annotationspb.HttpRule_Post{Post: cfg.prefix + "/fixed"}},
		{Selector: getParamOp, Pattern: &annotationspb.HttpRule_Get{Get: cfg.prefix + "/{param}"}},
		{Selector: postParamOp, Pattern: &annotationspb.HttpRule_Post{Post: cfg.prefix + "/{param}"}},
	}
	skipSC := cfg.serviceControlMode == "bypass"
	postAllowUnregistered := cfg.serviceControlMode == "report_only"

	usageRules := []*confpb.UsageRule{
		{Selector: getFixedOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: true},
		{Selector: postFixedOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: postAllowUnregistered},
		{Selector: getParamOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: true},
		{Selector: postParamOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: postAllowUnregistered},
	}

	if cfg.corsMode == "cors_collision" {
		methods = append(methods, &apipb.Method{Name: "OptionsAsymFixed"}, &apipb.Method{Name: "OptionsAsymParam"})
		httpRules = append(httpRules,
			&annotationspb.HttpRule{Selector: optFixedOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: cfg.prefix + "/fixed"}}},
			&annotationspb.HttpRule{Selector: optParamOp, Pattern: &annotationspb.HttpRule_Custom{Custom: &annotationspb.CustomHttpPattern{Kind: http.MethodOptions, Path: cfg.prefix + "/{param}"}}},
		)
		usageRules = append(usageRules,
			&confpb.UsageRule{Selector: optFixedOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: true},
			&confpb.UsageRule{Selector: optParamOp, SkipServiceControl: skipSC, AllowUnregisteredCalls: true},
		)
	}

	s.AppendAPIs([]*apipb.Api{{
		Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
		Methods: methods,
	}})
	s.AppendHttpRules(httpRules)
	s.OverrideUsage(&confpb.Usage{Rules: usageRules})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("static_get_open", func(t *testing.T) {
		targetURL := s.ListenerURL(cfg.prefix + "/fixed")
		if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if !skipSC {
			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: getFixedOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		}
	})

	t.Run("static_post_auth", func(t *testing.T) {
		targetURL := s.ListenerURL(cfg.prefix + "/fixed?key=api-key-fixed")
		if _, err := client.DoWithHeaders(targetURL, http.MethodPost, "hello", nil); err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if !skipSC {
			if postAllowUnregistered {
				scRequests, err := s.ServiceControlServer.GetRequests(1)
				if err != nil {
					t.Fatalf("s.ServiceControlServer.GetRequests(1) for POST %q failed: %v", targetURL, err)
				}
				wantSC := utils.ServiceControlExpectation{
					OperationName: postFixedOp,
					HTTPMethod:    http.MethodPost,
					HTTPStatus:    http.StatusOK,
					AuthHeader:    components.FakeBearerAuthHeader,
				}
				if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
					t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", targetURL, err)
				}
			} else {
				scRequests, err := s.ServiceControlServer.GetRequests(2)
				if err != nil {
					t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
				}
				wantSC := utils.ServiceControlExpectation{
					OperationName: postFixedOp,
					ConsumerID:    "api_key:api-key-fixed",
					HTTPMethod:    http.MethodPost,
					HTTPStatus:    http.StatusOK,
					AuthHeader:    components.FakeBearerAuthHeader,
				}
				if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
					t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
				}
			}
		}
	})

	t.Run("wildcard_get_open", func(t *testing.T) {
		targetURL := s.ListenerURL(cfg.prefix + "/dynamic")
		if _, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil); err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if !skipSC {
			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: getParamOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusOK,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		}
	})

	t.Run("wildcard_post_auth", func(t *testing.T) {
		targetURL := s.ListenerURL(cfg.prefix + "/dynamic?key=api-key-param")
		if _, err := client.DoWithHeaders(targetURL, http.MethodPost, "hello", nil); err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if !skipSC {
			if postAllowUnregistered {
				scRequests, err := s.ServiceControlServer.GetRequests(1)
				if err != nil {
					t.Fatalf("s.ServiceControlServer.GetRequests(1) for POST %q failed: %v", targetURL, err)
				}
				wantSC := utils.ServiceControlExpectation{
					OperationName: postParamOp,
					HTTPMethod:    http.MethodPost,
					HTTPStatus:    http.StatusOK,
					AuthHeader:    components.FakeBearerAuthHeader,
				}
				if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
					t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", targetURL, err)
				}
			} else {
				scRequests, err := s.ServiceControlServer.GetRequests(2)
				if err != nil {
					t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
				}
				wantSC := utils.ServiceControlExpectation{
					OperationName: postParamOp,
					ConsumerID:    "api_key:api-key-param",
					HTTPMethod:    http.MethodPost,
					HTTPStatus:    http.StatusOK,
					AuthHeader:    components.FakeBearerAuthHeader,
				}
				if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
					t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
				}
			}
		}
	})

	if !skipSC && !postAllowUnregistered {
		t.Run("static_post_unauth", func(t *testing.T) {
			targetURL := s.ListenerURL(cfg.prefix + "/fixed")
			_, status, _, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, "hello", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", targetURL, http.MethodPost, err)
			}
			if status != http.StatusUnauthorized {
				t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d", targetURL, http.MethodPost, status, http.StatusUnauthorized)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) for POST %q failed: %v", targetURL, err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: postFixedOp,
				HTTPMethod:    http.MethodPost,
				HTTPStatus:    http.StatusUnauthorized,
				AuthHeader:    components.FakeBearerAuthHeader,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", targetURL, err)
			}
		})
	}

	if skipSC {
		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET/POST = %d, want 0", got)
		}
	}

	if cfg.corsMode == "auto_cors" || cfg.corsMode == "cors_collision" {
		optionsCases := []struct {
			name          string
			suffix        string
			wantOptionsOp string
		}{
			{name: "options_static", suffix: "/fixed", wantOptionsOp: optFixedOp},
			{name: "options_wildcard", suffix: "/dynamic", wantOptionsOp: optParamOp},
		}

		for _, tc := range optionsCases {
			t.Run(tc.name, func(t *testing.T) {
				targetURL := s.ListenerURL(cfg.prefix + tc.suffix)
				headers, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://example.com", http.MethodPost, "")
				if err != nil {
					t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
				}
				if status != http.StatusOK && status != http.StatusNoContent {
					t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
				}
				if got, want := headers.Get("Access-Control-Allow-Origin"), "*"; got != want {
					t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
				}
				if !skipSC || cfg.corsMode == "auto_cors" {
					scRequests, err := s.ServiceControlServer.GetRequests(1)
					if err != nil {
						t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
					}
					wantSC := utils.ServiceControlExpectation{
						HTTPMethod: http.MethodOptions,
						HTTPStatus: status,
						AuthHeader: components.FakeBearerAuthHeader,
					}
					if cfg.corsMode == "cors_collision" {
						wantSC.OperationName = tc.wantOptionsOp
					}
					if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
						t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
					}
				}
			})
		}
		if skipSC && cfg.corsMode == "cors_collision" {
			time.Sleep(100 * time.Millisecond)
			if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
				t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed explicit OPTIONS = %d, want 0", got)
			}
		}
	}
}
