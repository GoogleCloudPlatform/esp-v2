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

// Package apigw_path_mutation_traversal_test tests ESPv2 API Gateway directory
// traversal and dot-segment normalization on exact and multi-segment wildcard
// routes.
package apigw_path_mutation_traversal_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var sharedEnv *env.TestEnv

// TestMain runs the package tests and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestExactRouteDirectoryTraversal verifies dot-segment (/../ and %2E%2E)
// normalization and escaped-slash handling on an exact route (/v1/admin).
func TestExactRouteDirectoryTraversal(t *testing.T) {
	s := setupTraversalEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "direct_exact",
			rawPath:    "/v1/admin",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/admin"`,
		},
		{
			name:       "dot_segment_normalized",
			rawPath:    "/v1/user/../admin",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/admin"`,
		},
		{
			name:       "encoded_dot_segment_normalized",
			rawPath:    "/v1/user/%2E%2E/admin",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/admin"`,
		},
		{
			// Under ESPv2's default KEEP_UNCHANGED policy, %2F is
			// not unescaped to '/', so /v1/user/..%2Fadmin remains
			// a 3-segment path and returns 404 Not Found.
			name:       "escaped_slash_in_dot_segment_unmatched",
			rawPath:    "/v1/user/..%2Fadmin",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, tc.rawPath, "", nil)
			if err != nil {
				t.Fatalf("utils.DoRawPathHTTP(GET, %q) failed: %v", tc.rawPath, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoRawPathHTTP(GET, %q) status = %d, want %d (body: %q)", tc.rawPath, gotStatus, tc.wantStatus, gotBody)
			}
			if tc.wantInBody != "" && !strings.Contains(gotBody, tc.wantInBody) {
				t.Errorf("utils.DoRawPathHTTP(GET, %q) body = %q, want substring %q", tc.rawPath, gotBody, tc.wantInBody)
			}
		})
	}
}

// TestMultiWildcardPathTraversalJailBreakout verifies dot-segment traversal
// containment and breakout rejection on a multi-segment wildcard route
// (/static/**).
func TestMultiWildcardPathTraversalJailBreakout(t *testing.T) {
	s := setupTraversalEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "valid_multisegment",
			rawPath:    "/static/app/bundle.js",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/static/app/bundle.js"`,
		},
		{
			name:       "in_jail_dot_traversal",
			rawPath:    "/static/dir/../bundle.js",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/static/bundle.js"`,
		},
		{
			// Traverses above /static to /etc/passwd, failing the
			// /static/** route match with 404 Not Found.
			name:       "escape_jail_dot_traversal",
			rawPath:    "/static/../../etc/passwd",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
		{
			name:       "escape_jail_encoded_dots",
			rawPath:    "/static/%2E%2E/%2E%2E/etc/passwd",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
		{
			// Under ESPv2's default KEEP_UNCHANGED policy, %2f is
			// not decoded to '/' before RFC 3986 normalization, so
			// "..%2f..%2fetc" remains a literal sub-segment under
			// /static/** and returns 200 OK with %2f preserved.
			name:       "escaped_slash_kept_in_jail",
			rawPath:    "/static/..%2f..%2fetc/passwd",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/static/..%2f..%2fetc/passwd"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, tc.rawPath, "", nil)
			if err != nil {
				t.Fatalf("utils.DoRawPathHTTP(GET, %q) failed: %v", tc.rawPath, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoRawPathHTTP(GET, %q) status = %d, want %d (body: %q)", tc.rawPath, gotStatus, tc.wantStatus, gotBody)
			}
			if tc.wantInBody != "" && !strings.Contains(gotBody, tc.wantInBody) {
				t.Errorf("utils.DoRawPathHTTP(GET, %q) body = %q, want substring %q", tc.rawPath, gotBody, tc.wantInBody)
			}
		})
	}
}

func setupTraversalEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWPathMutationTraversal, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "Admin"},
		{Name: "StaticFiles"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".Admin",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/admin"},
		},
		{
			Selector: svc + ".StaticFiles",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/static/**"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".Admin", AllowUnregisteredCalls: true},
			{Selector: svc + ".StaticFiles", AllowUnregisteredCalls: true},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup() failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}
