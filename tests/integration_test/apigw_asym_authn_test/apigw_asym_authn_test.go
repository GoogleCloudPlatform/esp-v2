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

// Package apigw_asym_authn_test verifies asymmetric authentication and route
// shadowing behavior for ESPv2 API Gateway configurations.
package apigw_asym_authn_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

var sharedAsymAuthnEnv *env.TestEnv

// TestMain runs the package tests and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedAsymAuthnEnv != nil {
		sharedAsymAuthnEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestExactRouteShadowedByProtectedSingleWildcard verifies that a public exact
// route is not shadowed by a JWT-protected single-segment wildcard route.
func TestExactRouteShadowedByProtectedSingleWildcard(t *testing.T) {
	s := setupAsymAuthnEnv(t)

	jwtHeaders := map[string]string{"Authorization": "Bearer " + testdata.Es256Token}
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{
			// Public exact route must not be shadowed by the
			// JWT-protected /v1/shadow/single/* wildcard.
			name:       "exact_public_without_jwt",
			path:       "/v1/shadow/single/public",
			wantStatus: http.StatusOK,
		},
		{
			// Sibling path matching wildcard requires JWT.
			name:       "wildcard_private_without_jwt",
			path:       "/v1/shadow/single/private",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wildcard_private_with_jwt",
			path:       "/v1/shadow/single/private",
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

// TestExactRouteShadowedByProtectedMultiSegmentWildcard verifies that a public
// exact route is not shadowed by a JWT-protected multi-segment wildcard route.
func TestExactRouteShadowedByProtectedMultiSegmentWildcard(t *testing.T) {
	s := setupAsymAuthnEnv(t)

	jwtHeaders := map[string]string{"Authorization": "Bearer " + testdata.Es256Token}
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{
			// Public exact route /v1/shadow/deep/public/health must
			// not be shadowed by JWT-protected /v1/shadow/deep/**.
			name:       "exact_public_without_jwt",
			path:       "/v1/shadow/deep/public/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "wildcard_private_without_jwt",
			path:       "/v1/shadow/deep/private/secret",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wildcard_private_with_jwt",
			path:       "/v1/shadow/deep/private/secret",
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

// TestExactOpenArbitraryHTTPVerb verifies that an arbitrary HTTP verb on an
// open exact route is rejected with 400 Bad Request.
func TestExactOpenArbitraryHTTPVerb(t *testing.T) {
	s := setupAsymAuthnEnv(t)

	addr := s.ListenerAddr()
	rawReq := "FOOBAR /v1/shadow/single/public HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	gotStatusLine, err := utils.SendRawTCPStatusLine(t.Context(), addr, rawReq, 5*time.Second)
	if err != nil {
		t.Fatalf("utils.SendRawTCPStatusLine(%q, FOOBAR) failed: %v", addr, err)
	}
	if want := "400 Bad Request"; !strings.Contains(gotStatusLine, want) {
		t.Errorf("utils.SendRawTCPStatusLine(%q, FOOBAR) = %q, want substring %q", addr, gotStatusLine, want)
	}
}

func setupAsymAuthnEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedAsymAuthnEnv != nil {
		return sharedAsymAuthnEnv
	}
	s := env.NewTestEnv(platform.TestAPIGWAsymAuthn, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	s.AppendAPIMethods([]*apipb.Method{
		{Name: "ShadowExactPublic"},
		{Name: "ShadowWildcardJwt"},
		{Name: "ShadowDeepPublic"},
		{Name: "ShadowDeepWildcardJwt"},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{Selector: apiName + ".ShadowExactPublic", AllowUnregisteredCalls: true},
		{Selector: apiName + ".ShadowWildcardJwt", AllowUnregisteredCalls: true},
		{Selector: apiName + ".ShadowDeepPublic", AllowUnregisteredCalls: true},
		{Selector: apiName + ".ShadowDeepWildcardJwt", AllowUnregisteredCalls: true},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".ShadowExactPublic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shadow/single/public"},
		},
		{
			Selector: apiName + ".ShadowWildcardJwt",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shadow/single/*"},
		},
		{
			Selector: apiName + ".ShadowDeepPublic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shadow/deep/public/health"},
		},
		{
			Selector: apiName + ".ShadowDeepWildcardJwt",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shadow/deep/**"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: apiName + ".ShadowWildcardJwt",
				Requirements: []*confpb.AuthRequirement{
					{ProviderId: testdata.TestAuthProvider, Audiences: "ok_audience"},
				},
			},
			{
				Selector: apiName + ".ShadowDeepWildcardJwt",
				Requirements: []*confpb.AuthRequirement{
					{ProviderId: testdata.TestAuthProvider, Audiences: "ok_audience"},
				},
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup() failed: %v", err)
	}
	sharedAsymAuthnEnv = s
	return s
}
