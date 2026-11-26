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

// Package apigw_wildcard_routing_test verifies overlapping wildcard routes
// with JWT, asymmetric authentication, and multi-segment wildcards in ESPv2.
package apigw_wildcard_routing_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

var (
	sharedWildcardRoutingEnv       *env.TestEnv
	sharedWildcardRoutingStrictEnv *env.TestEnv
)

// TestMain runs the package tests and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownWildcardRoutingEnvs(nil)
	os.Exit(code)
}

// TestOverlappingRoutesJWTAuth verifies JWT enforcement and exact-over-wildcard
// precedence under --normalize_path=false.
func TestOverlappingRoutesJWTAuth(t *testing.T) {
	s := setupWildcardRoutingStrictEnv(t)

	jwtHeaders := map[string]string{"Authorization": "Bearer " + testdata.Es256Token}

	// Exact route /v1/wild/jwt/secure matches OverlapJwtExact. Under
	// --normalize_path=false, /v1/wild/jwt/s%65cure and
	// /v1/wild/jwt/item123 both route to wildcard OverlapJwtWild and
	// require JWT authentication.
	tests := []struct {
		name   string
		path   string
		wantOp string
	}{
		{
			name:   "exact_secure",
			path:   "/v1/wild/jwt/secure",
			wantOp: apiName + ".OverlapJwtExact",
		},
		{
			name:   "percent_encoded_secure",
			path:   "/v1/wild/jwt/s%65cure",
			wantOp: apiName + ".OverlapJwtWild",
		},
		{
			name:   "dynamic_segment",
			path:   "/v1/wild/jwt/item123",
			wantOp: apiName + ".OverlapJwtWild",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
			}
			if gotStatus != http.StatusUnauthorized {
				t.Errorf("utils.DoHTTP(GET %q) without JWT status = %d (body %q), want %d", targetURL, gotStatus, body, http.StatusUnauthorized)
			}
			scReqs, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(1) after unauthorized GET %q failed: %v", targetURL, err)
			}
			wantUnauth := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusUnauthorized,
			}
			if err := utils.VerifyReportRequest(scReqs[0], wantUnauth); err != nil {
				t.Errorf("utils.VerifyReportRequest(unauthorized GET %q) failed: %v", targetURL, err)
			}

			_, gotStatus, body, err = utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", jwtHeaders)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) with JWT failed: %v", targetURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTP(GET %q) with JWT status = %d (body %q), want %d", targetURL, gotStatus, body, http.StatusOK)
			}
			scReqs, err = s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(1) after GET %q with JWT failed: %v", targetURL, err)
			}
			if err := utils.VerifyReportRequest(scReqs[0], utils.ServiceControlExpectation{OperationName: tc.wantOp}); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q with JWT) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingRoutesAsymmetricSingleMethod verifies asymmetric auth on an
// overlapping exact open route and a JWT+API-key-protected wildcard route.
func TestOverlappingRoutesAsymmetricSingleMethod(t *testing.T) {
	s := setupWildcardRoutingEnv(t)

	jwtHeaders := map[string]string{"Authorization": "Bearer " + testdata.Es256Token}
	// Exact /v1/wild/asym/open is open, while wildcard /v1/wild/asym/{id}
	// requires JWT + API key.
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "exact_open_without_auth",
			path:       "/v1/wild/asym/open",
			wantStatus: http.StatusOK,
		},
		{
			name:       "wildcard_without_auth",
			path:       "/v1/wild/asym/other",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wildcard_with_jwt_and_key",
			path:       "/v1/wild/asym/other?key=api-key",
			headers:    jwtHeaders,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", tc.headers)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, body, tc.wantStatus)
			}
		})
	}
}

// TestOverlappingRoutesAsymmetricMultipleMethods verifies asymmetric auth
// across HTTP methods on the same wildcard path.
func TestOverlappingRoutesAsymmetricMultipleMethods(t *testing.T) {
	s := setupWildcardRoutingEnv(t)

	// GET /v1/wild/multi/{id} is open, while POST /v1/wild/multi/{id}
	// requires an API key.
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "get_open_without_key",
			method:     http.MethodGet,
			path:       "/v1/wild/multi/123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "post_without_key",
			method:     http.MethodPost,
			path:       "/v1/wild/multi/123",
			body:       `{"message":"test"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "post_with_key",
			method:     http.MethodPost,
			path:       "/v1/wild/multi/123?key=api-key",
			body:       `{"message":"test"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, tc.method, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%s %q) failed: %v", tc.method, targetURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(%s %q) status = %d (body %q), want %d", tc.method, targetURL, gotStatus, body, tc.wantStatus)
			}
		})
	}
}

// TestMultiSegmentWildcardOpen verifies that an open multi-segment wildcard
// (**) route matches nested paths without authentication.
func TestMultiSegmentWildcardOpen(t *testing.T) {
	s := setupWildcardRoutingEnv(t)

	targetURL := s.ListenerURL("/v1/wild/deep/seg1/seg2/seg3")
	_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
	}
	if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, body, http.StatusOK)
	}
}

func tearDownWildcardRoutingEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedWildcardRoutingEnv != nil {
		sharedWildcardRoutingEnv.TearDown(t)
		sharedWildcardRoutingEnv = nil
	}
	if sharedWildcardRoutingStrictEnv != nil {
		sharedWildcardRoutingStrictEnv.TearDown(t)
		sharedWildcardRoutingStrictEnv = nil
	}
}

func buildWildcardRoutingEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWWildcardRouting, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	s.AppendAPIMethods([]*apipb.Method{
		{Name: "OverlapJwtExact"},
		{Name: "OverlapJwtWild"},
		{Name: "OverlapAsymSingleExact"},
		{Name: "OverlapAsymSingleWild"},
		{Name: "OverlapAsymMultiGet"},
		{Name: "OverlapAsymMultiPost"},
		{Name: "DeepWildcardOpen"},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{Selector: apiName + ".OverlapJwtExact", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapJwtWild", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapAsymSingleExact", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapAsymSingleWild", AllowUnregisteredCalls: false},
		{Selector: apiName + ".OverlapAsymMultiGet", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapAsymMultiPost", AllowUnregisteredCalls: false},
		{Selector: apiName + ".DeepWildcardOpen", AllowUnregisteredCalls: true},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".OverlapJwtExact",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/jwt/secure"},
		},
		{
			Selector: apiName + ".OverlapJwtWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/jwt/{id}"},
		},
		{
			Selector: apiName + ".OverlapAsymSingleExact",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/asym/open"},
		},
		{
			Selector: apiName + ".OverlapAsymSingleWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/asym/{id}"},
		},
		{
			Selector: apiName + ".OverlapAsymMultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/multi/{id}"},
		},
		{
			Selector: apiName + ".OverlapAsymMultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/wild/multi/{id}"},
		},
		{
			Selector: apiName + ".DeepWildcardOpen",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/wild/deep/**"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: apiName + ".OverlapJwtExact",
				Requirements: []*confpb.AuthRequirement{
					{ProviderId: testdata.TestAuthProvider, Audiences: "ok_audience"},
				},
			},
			{
				Selector: apiName + ".OverlapJwtWild",
				Requirements: []*confpb.AuthRequirement{
					{ProviderId: testdata.TestAuthProvider, Audiences: "ok_audience"},
				},
			},
			{
				Selector: apiName + ".OverlapAsymSingleWild",
				Requirements: []*confpb.AuthRequirement{
					{ProviderId: testdata.TestAuthProvider, Audiences: "ok_audience"},
				},
			},
		},
	})

	args := append(utils.CommonArgs(), extraArgs...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	return s
}

func setupWildcardRoutingEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedWildcardRoutingEnv != nil {
		return sharedWildcardRoutingEnv
	}
	tearDownWildcardRoutingEnvs(t)
	sharedWildcardRoutingEnv = buildWildcardRoutingEnv(t)
	return sharedWildcardRoutingEnv
}

func setupWildcardRoutingStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedWildcardRoutingStrictEnv != nil {
		sharedWildcardRoutingStrictEnv.ServiceControlServer.DrainRequests()
		return sharedWildcardRoutingStrictEnv
	}
	tearDownWildcardRoutingEnvs(t)
	sharedWildcardRoutingStrictEnv = buildWildcardRoutingEnv(t, "--normalize_path=false")
	return sharedWildcardRoutingStrictEnv
}
