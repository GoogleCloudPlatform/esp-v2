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

// Package apigw_path_mutation_slashes_test tests ESPv2 API Gateway handling of
// escaped, double-encoded, redundant, and trailing slash path mutations.
package apigw_path_mutation_slashes_test

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

var sharedExactEnv *env.TestEnv

// TestMain runs the package tests and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedExactEnv != nil {
		sharedExactEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestExactRouteEscapedSlashes verifies that escaped slashes (%2F) are not
// unescaped on an exact route (/v1/users/profile) and fail route matching with
// HTTP 404 Not Found.
func TestExactRouteEscapedSlashes(t *testing.T) {
	s := setupSlashesExactEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "exact_match",
			rawPath:    "/v1/users/profile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile"`,
		},
		{
			// Under ESPv2's default KEEP_UNCHANGED policy, %2F is
			// not unescaped to '/', so /v1/users%2Fprofile fails
			// exact route matching and returns 404 Not Found.
			name:       "escaped_slash_unmatched",
			rawPath:    "/v1/users%2Fprofile",
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

// TestDoubleEncodedSlashes verifies that double-encoded slashes (%252F) are
// not decoded on an exact route (/v1/files/all) and return HTTP 404 Not Found.
func TestDoubleEncodedSlashes(t *testing.T) {
	s := setupSlashesExactEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "exact_match",
			rawPath:    "/v1/files/all",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/files/all"`,
		},
		{
			name:       "double_encoded_slash_unmatched",
			rawPath:    "/v1/files%252Fall",
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

// TestRedundantSlashesMerging verifies that redundant internal and leading
// slashes (//) are merged and normalized before matching an exact route
// (/v1/users/profile).
func TestRedundantSlashesMerging(t *testing.T) {
	s := setupSlashesExactEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "single_slash",
			rawPath:    "/v1/users/profile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile"`,
		},
		{
			name:       "redundant_internal_slashes",
			rawPath:    "/v1//users///profile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile"`,
		},
		{
			name:       "leading_double_slash",
			rawPath:    "//v1/users/profile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile"`,
		},
		{
			name:       "redundant_slashes_unmatched",
			rawPath:    "/v1//users///other",
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

// TestTrailingSlashExactRoute verifies that single and multiple trailing
// slashes on an exact route (/v1/users/profile) match the route and collapse
// redundant trailing slashes.
func TestTrailingSlashExactRoute(t *testing.T) {
	s := setupSlashesExactEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "no_trailing_slash",
			rawPath:    "/v1/users/profile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile"`,
		},
		{
			name:       "single_trailing_slash",
			rawPath:    "/v1/users/profile/",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile/"`,
		},
		{
			name:       "multiple_trailing_slashes",
			rawPath:    "/v1/users/profile///",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users/profile/"`,
		},
		{
			name:       "trailing_slash_unmatched",
			rawPath:    "/v1/users/other/",
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

// TestSingleWildcardRouteEscapedSlashes verifies that an escaped slash (%2F)
// remains within a single segment and matches a single-wildcard route (/v1/*),
// whereas an unescaped slash fails route matching.
func TestSingleWildcardRouteEscapedSlashes(t *testing.T) {
	if sharedExactEnv != nil {
		sharedExactEnv.TearDown(t)
		sharedExactEnv = nil
	}
	s := env.NewTestEnv(platform.TestAPIGWPathMutationSlashes, platform.EchoSidecar)
	defer s.TearDown(t)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "WildcardUser"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".WildcardUser",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/*"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".WildcardUser", AllowUnregisteredCalls: true},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "escaped_slash_matches",
			rawPath:    "/v1/user%2Fprofile",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/user%2Fprofile"`,
		},
		{
			name:       "unescaped_slash_unmatched",
			rawPath:    "/v1/user/profile",
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

func setupSlashesExactEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedExactEnv != nil {
		return sharedExactEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWPathMutationSlashes, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "UsersProfile"},
		{Name: "FilesAll"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".UsersProfile",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/users/profile"},
		},
		{
			Selector: svc + ".FilesAll",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/files/all"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".UsersProfile", AllowUnregisteredCalls: true},
			{Selector: svc + ".FilesAll", AllowUnregisteredCalls: true},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup() failed: %v", err)
	}
	sharedExactEnv = s
	return sharedExactEnv
}
