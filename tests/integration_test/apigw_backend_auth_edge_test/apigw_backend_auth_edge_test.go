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

// Package apigw_backend_auth_edge_test verifies API Gateway backend
// authentication edge cases including multi-segment wildcards, shadowed routes,
// combined API keys, and non-standard or arbitrary HTTP verbs.
package apigw_backend_auth_edge_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const echoAPIName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

var sharedEnv *env.TestEnv

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestMultiSegmentWildcardOpenTokenInjection verifies backend token injection
// on an open multi-segment wildcard (**) route.
func TestMultiSegmentWildcardOpenTokenInjection(t *testing.T) {
	s := setupBackendAuthEdgeEnv(t)
	reqURL := s.ListenerURL("/bearertoken/edge/deep/a/b/c")
	resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	want := `{"Authorization": "Bearer fake-token-edge-deep-open", "RequestURI": "/bearertoken/edge/deep/a/b/c"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
	}
}

// TestExactShadowedWildcardTokenInjection verifies that an open exact route
// shadowing a JWT-protected wildcard route injects distinct backend tokens and
// enforces frontend JWT authentication only on the wildcard route.
func TestExactShadowedWildcardTokenInjection(t *testing.T) {
	s := setupBackendAuthEdgeEnv(t)

	t.Run("open_exact_route", func(t *testing.T) {
		// Open exact route /bearertoken/edge/shadow/public succeeds without
		// frontend JWT and injects fake-token-edge-shadow-public.
		publicURL := s.ListenerURL("/bearertoken/edge/shadow/public")
		resp, err := client.DoWithHeaders(publicURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, %q) failed: %v", publicURL, http.MethodGet, err)
		}
		wantPublic := `{"Authorization": "Bearer fake-token-edge-shadow-public", "RequestURI": "/bearertoken/edge/shadow/public"}`
		if err := util.JsonEqual(wantPublic, string(resp)); err != nil {
			t.Errorf("client.DoWithHeaders(%q, %q) response mismatch: %v", publicURL, http.MethodGet, err)
		}
	})

	t.Run("protected_wildcard_without_jwt", func(t *testing.T) {
		// Wildcard route /bearertoken/edge/shadow/other requires frontend JWT
		// (401 without JWT).
		protectedURL := s.ListenerURL("/bearertoken/edge/shadow/other")
		_, statusNoJWT, body, err := utils.DoHTTP(t.Context(), protectedURL, http.MethodGet, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) without JWT failed: %v", protectedURL, http.MethodGet, err)
		}
		if statusNoJWT != http.StatusUnauthorized {
			t.Errorf("utils.DoHTTP(%q, %q) without JWT status = %d, want %d (body: %q)", protectedURL, http.MethodGet, statusNoJWT, http.StatusUnauthorized, body)
		}
	})

	t.Run("protected_wildcard_with_jwt", func(t *testing.T) {
		// Wildcard route succeeds with frontend JWT and injects backend token.
		protectedURL := s.ListenerURL("/bearertoken/edge/shadow/other")
		_, statusWithJWT, bodyWithJWT, err := utils.DoHTTP(t.Context(), protectedURL, http.MethodGet, "", map[string]string{
			"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences,
		})
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) with JWT failed: %v", protectedURL, http.MethodGet, err)
		}
		if statusWithJWT != http.StatusOK {
			t.Fatalf("utils.DoHTTP(%q, %q) with JWT status = %d, want %d (body: %q)", protectedURL, http.MethodGet, statusWithJWT, http.StatusOK, bodyWithJWT)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(bodyWithJWT), &got); err != nil {
			t.Fatalf("json.Unmarshal(%q) failed: %v", bodyWithJWT, err)
		}
		if gotAuth, wantAuth := got["Authorization"], "Bearer fake-token-edge-shadow-protect"; gotAuth != wantAuth {
			t.Errorf("utils.DoHTTP(%q, %q) Authorization = %q, want %q", protectedURL, http.MethodGet, gotAuth, wantAuth)
		}
	})
}

// TestMultiSegmentWildcardCombinedAPIKeyTokenInjection verifies API key
// enforcement, query-over-header precedence, Service Control reporting, and
// backend token injection on a multi-segment wildcard (**) route.
func TestMultiSegmentWildcardCombinedAPIKeyTokenInjection(t *testing.T) {
	s := setupBackendAuthEdgeEnv(t)

	// Without API key -> 401 Unauthorized.
	noKeyURL := s.ListenerURL("/bearertoken/edge/apikey/x/y")
	_, statusNoKey, bodyNoKey, err := utils.DoHTTP(t.Context(), noKeyURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(%q, %q) without API key failed: %v", noKeyURL, http.MethodGet, err)
	}
	if statusNoKey != http.StatusUnauthorized {
		t.Errorf("utils.DoHTTP(%q, %q) without API key status = %d, want %d (body: %q)", noKeyURL, http.MethodGet, statusNoKey, http.StatusUnauthorized, bodyNoKey)
	}
	// Consume the 401 ReportRequest (and any batched reports from earlier tests
	// sharing this env) before sending the authorized request so no stale
	// report races with the subsequent Check+Report pair.
	if _, err := s.ServiceControlServer.WaitForReportedOperation(echoAPIName+".EdgeDeepAPIKey", 4); err != nil {
		t.Fatalf("ServiceControlServer.WaitForReportedOperation(%q, 4) after 401 on %q failed: %v", echoAPIName+".EdgeDeepAPIKey", noKeyURL, err)
	}

	// With both query ?key=query-api-key-edge and header X-Api-Key: header-key,
	// ESPv2 returns 200 OK, injects the backend token, and selects the query
	// key.
	withKeyURL := s.ListenerURL("/bearertoken/edge/apikey/x/y?key=query-api-key-edge")
	_, statusWithKey, bodyWithKey, err := utils.DoHTTP(t.Context(), withKeyURL, http.MethodGet, "", map[string]string{
		"X-Api-Key": "header-key",
	})
	if err != nil {
		t.Fatalf("utils.DoHTTP(%q, %q) with combined API key failed: %v", withKeyURL, http.MethodGet, err)
	}
	if statusWithKey != http.StatusOK {
		t.Fatalf("utils.DoHTTP(%q, %q) with combined API key status = %d, want %d (body: %q)", withKeyURL, http.MethodGet, statusWithKey, http.StatusOK, bodyWithKey)
	}
	want := `{"Authorization": "Bearer fake-token-edge-deep-apikey", "RequestURI": "/bearertoken/edge/apikey/x/y?key=query-api-key-edge"}`
	if err := util.JsonEqual(want, bodyWithKey); err != nil {
		t.Errorf("utils.DoHTTP(%q, %q) response mismatch: %v", withKeyURL, http.MethodGet, err)
	}

	scReqs, err := s.ServiceControlServer.GetRequests(2)
	if err != nil {
		t.Fatalf("ServiceControlServer.GetRequests(2) for %q failed: %v", withKeyURL, err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: echoAPIName + ".EdgeDeepAPIKey",
		ConsumerID:    "api_key:query-api-key-edge",
		HTTPMethod:    http.MethodGet,
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyCheckAndReportRequests(scReqs, wantSC); err != nil {
		t.Errorf("utils.VerifyCheckAndReportRequests(%q) failed: %v", withKeyURL, err)
	}
}

// TestMultiSegmentWildcardNonStandardMethodsHTTPREST verifies custom OPTIONS
// routing and 405 rejection of unconfigured TRACE on a multi-segment wildcard
// (**) route.
func TestMultiSegmentWildcardNonStandardMethodsHTTPREST(t *testing.T) {
	s := setupBackendAuthEdgeEnv(t)
	reqURL := s.ListenerURL("/bearertoken/edge/nonstd/a/b")

	t.Run("options", func(t *testing.T) {
		_, optionsStatus, optionsBody, err := utils.DoHTTP(t.Context(), reqURL, http.MethodOptions, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodOptions, err)
		}
		if optionsStatus != http.StatusOK {
			t.Fatalf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodOptions, optionsStatus, http.StatusOK, optionsBody)
		}
		want := `{"Authorization": "Bearer fake-token-edge-nonstd-options", "RequestURI": "/bearertoken/edge/nonstd/a/b"}`
		if err := util.JsonEqual(want, optionsBody); err != nil {
			t.Errorf("utils.DoHTTP(%q, %q) response mismatch: %v", reqURL, http.MethodOptions, err)
		}
	})

	t.Run("trace", func(t *testing.T) {
		_, traceStatus, traceBody, err := utils.DoHTTP(t.Context(), reqURL, http.MethodTrace, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodTrace, err)
		}
		if traceStatus != http.StatusMethodNotAllowed {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodTrace, traceStatus, http.StatusMethodNotAllowed, traceBody)
		}
	})
}

// TestMultiSegmentWildcardArbitraryVerbHTTPREST verifies that an arbitrary HTTP
// verb (FOOBAR) sent over raw TCP is rejected with 400 Bad Request.
func TestMultiSegmentWildcardArbitraryVerbHTTPREST(t *testing.T) {
	s := setupBackendAuthEdgeEnv(t)
	addr := s.ListenerAddr()
	rawReq := fmt.Sprintf("FOOBAR /bearertoken/edge/deep/a/b HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", addr)

	statusLine, err := utils.SendRawTCPStatusLine(t.Context(), addr, rawReq, 5*time.Second)
	if err != nil {
		t.Fatalf("utils.SendRawTCPStatusLine(%q, FOOBAR) failed: %v", addr, err)
	}
	if !strings.Contains(statusLine, "400 Bad Request") {
		t.Errorf("utils.SendRawTCPStatusLine(%q, FOOBAR) = %q, want status containing %q", addr, strings.TrimSpace(statusLine), "400 Bad Request")
	}
}

func setupBackendAuthEdgeEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWBackendAuthEdge, platform.EchoRemote)

	backendRoutes := []struct {
		method            string
		audience          string
		token             string
		allowUnregistered bool
	}{
		{method: "EdgeDeepOpen", audience: "https://example.com/edge-deep-open", token: "fake-token-edge-deep-open", allowUnregistered: true},
		{method: "EdgeShadowPublic", audience: "https://example.com/edge-shadow-public", token: "fake-token-edge-shadow-public", allowUnregistered: true},
		{method: "EdgeShadowProtected", audience: "https://example.com/edge-shadow-protect", token: "fake-token-edge-shadow-protect", allowUnregistered: true},
		{method: "EdgeDeepAPIKey", audience: "https://example.com/edge-deep-apikey", token: "fake-token-edge-deep-apikey", allowUnregistered: false},
		{method: "EdgeNonStdOptions", audience: "https://example.com/edge-nonstd-options", token: "fake-token-edge-nonstd-options", allowUnregistered: true},
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
			Selector:               echoAPIName + "." + r.method,
			AllowUnregisteredCalls: r.allowUnregistered,
		})
		backendRules = append(backendRules, &confpb.BackendRule{
			Selector:        echoAPIName + "." + r.method,
			Address:         addr,
			PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
			Authentication:  &confpb.BackendRule_JwtAudience{JwtAudience: r.audience},
		})
	}
	s.OverrideMockMetadata(mockMetadata, 0)
	s.AppendAPIMethods(apiMethods)
	s.AppendUsageRules(usageRules)
	s.AppendBackendRules(backendRules)

	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: echoAPIName + ".EdgeShadowProtected",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.GoogleJwtProvider,
						Audiences:  "admin.cloud.goog",
					},
				},
			},
		},
	})

	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: echoAPIName + ".EdgeDeepOpen",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/edge/deep/**"},
		},
		{
			Selector: echoAPIName + ".EdgeShadowPublic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/edge/shadow/public"},
		},
		{
			Selector: echoAPIName + ".EdgeShadowProtected",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/edge/shadow/*"},
		},
		{
			Selector: echoAPIName + ".EdgeDeepAPIKey",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/edge/apikey/**"},
		},
		{
			Selector: echoAPIName + ".EdgeNonStdOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/bearertoken/edge/nonstd/**",
				},
			},
		},
	})

	args := utils.CommonArgs()
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	sharedEnv = s
	return sharedEnv
}
