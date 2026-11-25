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

// Package apigw_backend_auth_cors_test verifies API Gateway backend
// authentication token injection with CORS preflight routing.
package apigw_backend_auth_cors_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
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

const preflightOrigin = "http://cloud.google.com"

var (
	sharedEnv       *env.TestEnv
	sharedStrictEnv *env.TestEnv
)

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownCORSSharedEnvs(nil)
	os.Exit(code)
}

// TestExactAutoCORSTokenInjection verifies backend token injection for an exact
// route with auto-generated CORS preflight.
func TestExactAutoCORSTokenInjection(t *testing.T) {
	s := setupBackendAuthCORSEnv(t)
	reqURL := s.ListenerURL("/bearertoken/cors/exact")

	// GET request injects fake-token-cors-exact.
	if resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil); err != nil {
		t.Errorf("client.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	} else {
		want := `{"Authorization": "Bearer fake-token-cors-exact", "RequestURI": "/bearertoken/cors/exact"}`
		if err := util.JsonEqual(want, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
		}
	}

	// Auto-generated OPTIONS route (allow_cors: true) forwards to backend with
	// the original method's backend_auth token.
	respHeader, status, body, err := utils.DoCORSPreflight(t.Context(), reqURL, preflightOrigin, http.MethodGet, "X-PINGOTHER")
	if err != nil {
		t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	if status != http.StatusOK {
		t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodGet, status, http.StatusOK, body)
	}
	if got, want := respHeader.Get("Access-Control-Expose-Headers"), "X-Token: Bearer fake-token-cors-exact"; got != want {
		t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Expose-Headers = %q, want %q", reqURL, http.MethodGet, got, want)
	}
}

// TestExactCORSCollisionTokenInjection verifies backend token injection when an
// explicit OPTIONS route suppresses the auto-generated CORS route.
func TestExactCORSCollisionTokenInjection(t *testing.T) {
	s := setupBackendAuthCORSEnv(t)
	reqURL := s.ListenerURL("/bearertoken/cors/collision")

	// GET uses CollisionGet token.
	if resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil); err != nil {
		t.Errorf("client.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	} else {
		want := `{"Authorization": "Bearer fake-token-collision-get", "RequestURI": "/bearertoken/cors/collision"}`
		if err := util.JsonEqual(want, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
		}
	}

	// Explicit OPTIONS suppresses auto-generated CORS route and uses
	// CollisionOptions token.
	respHeader, status, body, err := utils.DoCORSPreflight(t.Context(), reqURL, preflightOrigin, http.MethodGet, "X-PINGOTHER")
	if err != nil {
		t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	if status != http.StatusOK {
		t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodGet, status, http.StatusOK, body)
	}
	if got, want := respHeader.Get("Access-Control-Expose-Headers"), "X-Token: Bearer fake-token-collision-options"; got != want {
		t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Expose-Headers = %q, want %q", reqURL, http.MethodGet, got, want)
	}
}

// TestSingleWildcardAutoCORSTokenInjection verifies backend token injection for
// a single-segment wildcard route with auto-generated CORS preflight.
func TestSingleWildcardAutoCORSTokenInjection(t *testing.T) {
	s := setupBackendAuthCORSEnv(t)
	reqURL := s.ListenerURL("/bearertoken/cors/wild/seg1")

	if resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil); err != nil {
		t.Errorf("client.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	} else {
		want := `{"Authorization": "Bearer fake-token-cors-wild", "RequestURI": "/bearertoken/cors/wild/seg1"}`
		if err := util.JsonEqual(want, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
		}
	}

	respHeader, status, body, err := utils.DoCORSPreflight(t.Context(), reqURL, preflightOrigin, http.MethodGet, "X-PINGOTHER")
	if err != nil {
		t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	if status != http.StatusOK {
		t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodGet, status, http.StatusOK, body)
	}
	if got, want := respHeader.Get("Access-Control-Expose-Headers"), "X-Token: Bearer fake-token-cors-wild"; got != want {
		t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Expose-Headers = %q, want %q", reqURL, http.MethodGet, got, want)
	}
}

// TestOverlappingCORSCollisionTokenInjection verifies overlapping static and
// wildcard CORS preflight token injection under strict raw path routing.
func TestOverlappingCORSCollisionTokenInjection(t *testing.T) {
	s := setupBackendAuthCORSStrictEnv(t)

	tests := []struct {
		name       string
		path       string
		wantHeader string
	}{
		{
			// Static path has explicit OPTIONS (OverlapStaticOptions ->
			// fake-token-overlap-static-opt).
			name:       "static_explicit_options",
			path:       "/bearertoken/cors/overlap/static",
			wantHeader: "X-Token: Bearer fake-token-overlap-static-opt",
		},
		{
			// Wildcard sibling path routes to auto-generated wildcard OPTIONS
			// (ESPv2_Autogenerated_CORS_OverlapWildGet ->
			// fake-token-overlap-wild-get).
			name:       "wildcard_sibling",
			path:       "/bearertoken/cors/overlap/item1",
			wantHeader: "X-Token: Bearer fake-token-overlap-wild-get",
		},
		{
			// Raw percent-encoded /st%61tic (under --normalize_path=false)
			// routes to auto-generated wildcard OPTIONS.
			name:       "raw_percent_encoded_static",
			path:       "/bearertoken/cors/overlap/st%61tic",
			wantHeader: "X-Token: Bearer fake-token-overlap-wild-get",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.path)
			respHeader, status, body, err := utils.DoCORSPreflight(t.Context(), reqURL, preflightOrigin, http.MethodGet, "X-PINGOTHER")
			if err != nil {
				t.Fatalf("utils.DoCORSPreflight(%q, %q) failed: %v", reqURL, http.MethodGet, err)
			}
			if status != http.StatusOK {
				t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodGet, status, http.StatusOK, body)
			}
			if got := respHeader.Get("Access-Control-Expose-Headers"); got != tc.wantHeader {
				t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Expose-Headers = %q, want %q", reqURL, http.MethodGet, got, tc.wantHeader)
			}
		})
	}
}

func tearDownCORSSharedEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedEnv != nil {
		sharedEnv.TearDown(t)
		sharedEnv = nil
	}
	if sharedStrictEnv != nil {
		sharedStrictEnv.TearDown(t)
		sharedStrictEnv = nil
	}
}

func setupBackendAuthCORSEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}
	tearDownCORSSharedEnvs(t)
	sharedEnv = buildBackendAuthCORSEnv(t)
	return sharedEnv
}

func setupBackendAuthCORSStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedStrictEnv != nil {
		return sharedStrictEnv
	}
	tearDownCORSSharedEnvs(t)
	sharedStrictEnv = buildBackendAuthCORSEnv(t, "--normalize_path=false")
	return sharedStrictEnv
}

func buildBackendAuthCORSEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()

	s := env.NewTestEnv(platform.TestAPIGWBackendAuthCORS, platform.EchoRemote)
	s.SetAllowCors()

	const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	backendRoutes := []struct {
		method   string
		audience string
		token    string
	}{
		{method: "CorsExactGet", audience: "https://example.com/cors-exact", token: "fake-token-cors-exact"},
		{method: "CollisionGet", audience: "https://example.com/collision-get", token: "fake-token-collision-get"},
		{method: "CollisionOptions", audience: "https://example.com/collision-options", token: "fake-token-collision-options"},
		{method: "CorsWildGet", audience: "https://example.com/cors-wild", token: "fake-token-cors-wild"},
		{method: "OverlapStaticGet", audience: "https://example.com/overlap-static-get", token: "fake-token-overlap-static-get"},
		{method: "OverlapStaticOptions", audience: "https://example.com/overlap-static-opt", token: "fake-token-overlap-static-opt"},
		{method: "OverlapWildGet", audience: "https://example.com/overlap-wild-get", token: "fake-token-overlap-wild-get"},
	}

	addr := "https://" + net.JoinHostPort(platform.GetLoopbackAddress(), platform.WorkingBackendPort)
	mockMetadata := make(map[string]string, len(backendRoutes))
	var apiMethods []*apipb.Method
	var usageRules []*confpb.UsageRule
	var backendRules []*confpb.BackendRule
	for _, r := range backendRoutes {
		mockMetadata[fmt.Sprintf("%s?format=standard&audience=%s", util.IdentityTokenPath, r.audience)] = r.token
		apiMethods = append(apiMethods, &apipb.Method{Name: r.method})
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               apiName + "." + r.method,
			AllowUnregisteredCalls: true,
		})
		backendRules = append(backendRules, &confpb.BackendRule{
			Selector:        apiName + "." + r.method,
			Address:         addr,
			PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
			Authentication:  &confpb.BackendRule_JwtAudience{JwtAudience: r.audience},
		})
	}
	s.OverrideMockMetadata(mockMetadata, 0)
	s.AppendAPIMethods(apiMethods)
	s.AppendUsageRules(usageRules)
	s.AppendBackendRules(backendRules)

	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".CorsExactGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/cors/exact"},
		},
		{
			Selector: apiName + ".CollisionGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/cors/collision"},
		},
		{
			Selector: apiName + ".CollisionOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/bearertoken/cors/collision",
				},
			},
		},
		{
			Selector: apiName + ".CorsWildGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/cors/wild/*"},
		},
		{
			Selector: apiName + ".OverlapStaticGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/cors/overlap/static"},
		},
		{
			Selector: apiName + ".OverlapStaticOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/bearertoken/cors/overlap/static",
				},
			},
		},
		{
			Selector: apiName + ".OverlapWildGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/cors/overlap/{id}"},
		},
	})

	args := append(extraArgs, utils.CommonArgs()...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	return s
}
