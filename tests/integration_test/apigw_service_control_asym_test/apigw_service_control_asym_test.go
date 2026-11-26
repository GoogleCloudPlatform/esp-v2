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

// Package apigw_service_control_asym_test contains ESPv2 integration tests for
// Service Control behavior on exact routes with asymmetric per-verb
// authentication and CORS configurations.
package apigw_service_control_asym_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

// TestExactAsymmetricAutoCORS verifies Service Control behavior with auto-CORS
// on an exact route where GET allows unregistered calls and POST requires an
// API key.
func TestExactAsymmetricAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymGet"},
				{Name: "DynamicRoutingAsymPost"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asym",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asym",
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPost",
				AllowUnregisteredCalls: false,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asym")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodGet, "", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asym"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asym?key=api-key-1")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodPost, "test body", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asym"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPost",
			ConsumerID:    "api_key:api-key-1",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("preflight_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asym")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodOptions,
			HTTPStatus: status,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})

	t.Run("unauth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asym")
		_, status, _, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusUnauthorized {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d", targetURL, http.MethodPost, status, http.StatusUnauthorized)
		}
	})
}

// TestExactAsymmetricCORSCollision verifies Service Control behavior when
// auto-CORS collides with an explicit OPTIONS rule on an asymmetric exact route
// where GET allows unregistered calls and POST requires an API key.
func TestExactAsymmetricCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymCollGet"},
				{Name: "DynamicRoutingAsymCollPost"},
				{Name: "DynamicRoutingAsymCollOptions"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asymcoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asymcoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/dynamicrouting/asymcoll",
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollPost",
				AllowUnregisteredCalls: false,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollOptions",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymcoll")
		resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymcoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymcoll?key=api-key-1")
		resp, err := client.DoWithHeaders(targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymcoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollPost",
			ConsumerID:    "api_key:api-key-1",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("explicit_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymcoll")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymCollOptions",
			HTTPMethod:    http.MethodOptions,
			HTTPStatus:    status,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})
}

// TestExactAsymmetricProxyAuthAutoCORS verifies Service Control proxy
// authentication headers and auto-CORS on an exact route where GET allows
// unregistered calls and POST requires an API key.
func TestExactAsymmetricProxyAuthAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymPAGet"},
				{Name: "DynamicRoutingAsymPAPost"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asympa",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asympa",
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAPost",
				AllowUnregisteredCalls: false,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympa")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodGet, "", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asympa"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympa?key=api-key-1")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodPost, "test body", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asympa"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPAPost",
			ConsumerID:    "api_key:api-key-1",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("preflight_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympa")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodOptions,
			HTTPStatus: status,
			AuthHeader: components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})

	t.Run("unauth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympa")
		_, status, _, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusUnauthorized {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d", targetURL, http.MethodPost, status, http.StatusUnauthorized)
		}
	})
}

// TestExactAsymmetricProxyAuthCORSCollision verifies Service Control proxy
// authentication headers when auto-CORS collides with an explicit OPTIONS rule
// on an asymmetric exact route.
func TestExactAsymmetricProxyAuthCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymPACollGet"},
				{Name: "DynamicRoutingAsymPACollPost"},
				{Name: "DynamicRoutingAsymPACollOptions"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asympacoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asympacoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/dynamicrouting/asympacoll",
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollPost",
				AllowUnregisteredCalls: false,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollOptions",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympacoll")
		resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asympacoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympacoll?key=api-key-1")
		resp, err := client.DoWithHeaders(targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asympacoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollPost",
			ConsumerID:    "api_key:api-key-1",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("explicit_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympacoll")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymPACollOptions",
			HTTPMethod:    http.MethodOptions,
			HTTPStatus:    status,
			AuthHeader:    components.FakeBearerAuthHeader,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})

	t.Run("unauth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asympacoll")
		_, status, _, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusUnauthorized {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d", targetURL, http.MethodPost, status, http.StatusUnauthorized)
		}
	})
}

// TestExactAsymmetricBypassAutoCORS verifies Service Control bypass with
// auto-CORS on an asymmetric exact route, ensuring GET and POST skip Service
// Control while autogenerated OPTIONS preflight still reports.
func TestExactAsymmetricBypassAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymBypassGet"},
				{Name: "DynamicRoutingAsymBypassPost"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asymbypass",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asymbypass",
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassGet",
				AllowUnregisteredCalls: true,
				SkipServiceControl:     true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassPost",
				AllowUnregisteredCalls: false,
				SkipServiceControl:     true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypass")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodGet, "", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymbypass"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
		}

		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET %q = %d, want 0", targetURL, got)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypass?key=api-key-1")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodPost, "test body", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymbypass"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed POST %q = %d, want 0", targetURL, got)
		}
	})

	t.Run("preflight_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypass")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodOptions,
			HTTPStatus: status,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})
}

// TestExactAsymmetricBypassCORSCollision verifies Service Control bypass when
// auto-CORS collides with an explicit OPTIONS rule and all methods skip Service
// Control.
func TestExactAsymmetricBypassCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymBypassCollGet"},
				{Name: "DynamicRoutingAsymBypassCollPost"},
				{Name: "DynamicRoutingAsymBypassCollOptions"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asymbypasscoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asymbypasscoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/dynamicrouting/asymbypasscoll",
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollGet",
				AllowUnregisteredCalls: true,
				SkipServiceControl:     true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollPost",
				AllowUnregisteredCalls: false,
				SkipServiceControl:     true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymBypassCollOptions",
				AllowUnregisteredCalls: true,
				SkipServiceControl:     true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypasscoll")
		resp, err := client.DoWithHeaders(targetURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymbypasscoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}

		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed GET %q = %d, want 0", targetURL, got)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypasscoll?key=api-key-1")
		resp, err := client.DoWithHeaders(targetURL, http.MethodPost, "test body", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymbypasscoll"; !strings.Contains(got, want) {
			t.Errorf("client.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}

		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed POST %q = %d, want 0", targetURL, got)
		}
	})

	t.Run("explicit_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymbypasscoll")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		time.Sleep(100 * time.Millisecond)
		if got := s.ServiceControlServer.GetRequestCount(); got != 0 {
			t.Errorf("s.ServiceControlServer.GetRequestCount() after bypassed OPTIONS %q = %d, want 0", targetURL, got)
		}
	})
}

// TestExactAsymmetricReportOnlyAutoCORS verifies report-only Service Control
// behavior with auto-CORS on an exact route where GET, POST, and autogenerated
// OPTIONS preflight each emit only a Report request.
func TestExactAsymmetricReportOnlyAutoCORS(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymROGet"},
				{Name: "DynamicRoutingAsymROPost"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asymro",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asymro",
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROPost",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymro")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodGet, "", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymro"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymro?key=api-key-1")
		headers := map[string]string{
			"Origin": "http://cloud.google.com",
		}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodPost, "test body", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymro"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROPost",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("preflight_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymro")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodOptions,
			HTTPStatus: status,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})
}

// TestExactAsymmetricReportOnlyCORSCollision verifies report-only Service
// Control behavior when auto-CORS collides with an explicit OPTIONS rule and
// GET, POST, and OPTIONS each emit only a Report request.
func TestExactAsymmetricReportOnlyCORSCollision(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAsym, platform.EchoSidecar)
	defer s.TearDown(t)
	s.SetAllowCors()

	s.AppendAPIs([]*apipb.Api{
		{
			Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "DynamicRoutingAsymROCollGet"},
				{Name: "DynamicRoutingAsymROCollPost"},
				{Name: "DynamicRoutingAsymROCollOptions"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollGet",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/dynamicrouting/asymrocoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollPost",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/dynamicrouting/asymrocoll",
			},
		},
		{
			Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/dynamicrouting/asymrocoll",
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollGet",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollPost",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollOptions",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	t.Run("open_get", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymrocoll")
		headers := map[string]string{"Origin": "http://cloud.google.com"}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodGet, "", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodGet, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymrocoll"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodGet, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodGet, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for GET %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollGet",
			HTTPMethod:    http.MethodGet,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
		}
	})

	t.Run("auth_post", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymrocoll?key=api-key-1")
		headers := map[string]string{"Origin": "http://cloud.google.com"}
		respHeaders, resp, err := utils.DoWithHeaders(targetURL, http.MethodPost, "test body", headers)
		if err != nil {
			t.Fatalf("utils.DoWithHeaders(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if got, want := string(resp), "/dynamicrouting/asymrocoll"; !strings.Contains(got, want) {
			t.Errorf("utils.DoWithHeaders(%q, %q) body = %q, want to contain %q", targetURL, http.MethodPost, got, want)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoWithHeaders(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for POST %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollPost",
			HTTPMethod:    http.MethodPost,
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", targetURL, err)
		}
	})

	t.Run("explicit_options", func(t *testing.T) {
		targetURL := s.ListenerURL("/dynamicrouting/asymrocoll")
		respHeaders, status, _, err := utils.DoCORSPreflight(t.Context(), targetURL, "http://cloud.google.com", http.MethodPost, "Authorization")
		if err != nil {
			t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", targetURL, http.MethodPost, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want 200 or 204", targetURL, http.MethodPost, status)
		}
		if got, want := respHeaders.Get("Access-Control-Allow-Origin"), "*"; got != want {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", targetURL, http.MethodPost, got, want)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) for OPTIONS %q failed: %v", targetURL, err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.DynamicRoutingAsymROCollOptions",
			HTTPMethod:    http.MethodOptions,
			HTTPStatus:    status,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(OPTIONS %q) failed: %v", targetURL, err)
		}
	})
}
