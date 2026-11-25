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

// Package apigw_service_control_cors_test tests ESPv2 API Gateway auto-CORS
// route deduplication and explicit OPTIONS handling with Service Control
// reporting across same-shape multi-variable routes.
package apigw_service_control_cors_test

import (
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const echoV2Service = "2.echo_api_endpoints_cloudesf_testing_cloud_goog"

// TestSameShapeMultiVarAutoCORS verifies auto-CORS preflight route
// deduplication and Service Control reporting when multiple HTTP methods share
// the same path shape with different variable names
// (/dynamicrouting/shelves/{shelf} and /dynamicrouting/shelves/{shelf_diff}).
func TestSameShapeMultiVarAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlCORS, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: echoV2Service,
			Methods: []*apipb.Method{
				{Name: "GetShelfSameShape"},
				{Name: "UpdateShelfSameShape"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: echoV2Service + ".GetShelfSameShape",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/shelves/{shelf}",
			},
		},
		{
			Selector: echoV2Service + ".UpdateShelfSameShape",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/shelves/{shelf_diff}",
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               echoV2Service + ".GetShelfSameShape",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               echoV2Service + ".UpdateShelfSameShape",
				AllowUnregisteredCalls: true,
			},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("get_request", func(t *testing.T) {
		reqURL := s.ListenerURL("/dynamicrouting/shelves/item100")
		resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
		}
		const wantResp = `{"RequestURI": "/dynamicrouting/shelves/item100"}`
		if err := util.JsonEqual(wantResp, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
		}

		// With Service Control, AllowUnregisteredCalls: true skips
		// Check and sends a single ReportRequest.
		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: echoV2Service + ".GetShelfSameShape",
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", reqURL, err)
		}
	})

	t.Run("preflight_options", func(t *testing.T) {
		reqURL := s.ListenerURL("/dynamicrouting/shelves/item100")
		respHeaders, gotStatus, gotBody, err := utils.DoCORSPreflight(t.Context(), reqURL, "http://cloud.google.com", http.MethodGet, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, GET) failed: %v", reqURL, err)
		}
		if gotStatus != http.StatusOK && gotStatus != http.StatusNoContent {
			t.Fatalf("utils.DoCORSPreflight(%q, GET) status = %d, want 200 or 204 (body: %q)", reqURL, gotStatus, gotBody)
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("utils.DoCORSPreflight(%q, GET) Access-Control-Allow-Origin = %q, want %q", reqURL, got, "*")
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodOptions,
			HTTPStatus: gotStatus,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", reqURL, err)
		}
	})
}

// TestSameShapeMultiVarExplicitOPTIONS verifies that an explicit user-defined
// OPTIONS rule suppresses auto-CORS route generation and reports the explicit
// operation to Service Control across same-shape multi-variable routes.
func TestSameShapeMultiVarExplicitOPTIONS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlCORS, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: echoV2Service,
			Methods: []*apipb.Method{
				{Name: "GetShelfSameShapeColl"},
				{Name: "UpdateShelfSameShapeColl"},
				{Name: "OptionsShelfSameShapeColl"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: echoV2Service + ".GetShelfSameShapeColl",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/shelves/{shelf}",
			},
		},
		{
			Selector: echoV2Service + ".UpdateShelfSameShapeColl",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/shelves/{shelf_diff}",
			},
		},
		{
			Selector: echoV2Service + ".OptionsShelfSameShapeColl",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/dynamicrouting/shelves/{shelf_options}",
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               echoV2Service + ".GetShelfSameShapeColl",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               echoV2Service + ".UpdateShelfSameShapeColl",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               echoV2Service + ".OptionsShelfSameShapeColl",
				AllowUnregisteredCalls: true,
			},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("get_request", func(t *testing.T) {
		reqURL := s.ListenerURL("/dynamicrouting/shelves/item200")
		resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
		}
		const wantResp = `{"RequestURI": "/dynamicrouting/shelves/item200"}`
		if err := util.JsonEqual(wantResp, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: echoV2Service + ".GetShelfSameShapeColl",
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", reqURL, err)
		}
	})

	t.Run("explicit_options", func(t *testing.T) {
		reqURL := s.ListenerURL("/dynamicrouting/shelves/item200")
		respHeaders, gotStatus, gotBody, err := utils.DoCORSPreflight(t.Context(), reqURL, "http://cloud.google.com", http.MethodGet, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, GET) failed: %v", reqURL, err)
		}
		if gotStatus != http.StatusOK && gotStatus != http.StatusNoContent {
			t.Fatalf("utils.DoCORSPreflight(%q, GET) status = %d, want 200 or 204 (body: %q)", reqURL, gotStatus, gotBody)
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("utils.DoCORSPreflight(%q, GET) Access-Control-Allow-Origin = %q, want %q", reqURL, got, "*")
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: echoV2Service + ".OptionsShelfSameShapeColl",
			HTTPMethod:    http.MethodOptions,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", reqURL, err)
		}
	})
}
