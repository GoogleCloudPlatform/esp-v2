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

// Package apigw_path_mutation_matrix_test tests ESPv2 API Gateway handling of
// matrix parameters (;k=v) on exact and single-wildcard routes.
package apigw_path_mutation_matrix_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

// TestMatrixParametersExactRoute verifies that matrix parameters (;k=v) on an
// exact route (/v1/users) are not stripped and fail exact route matching with
// HTTP 404 Not Found.
func TestMatrixParametersExactRoute(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWPathMutationMatrix, platform.EchoSidecar)
	defer s.TearDown(t)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "Users"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".Users",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/users"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".Users", AllowUnregisteredCalls: true},
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
			name:       "exact_route",
			rawPath:    "/v1/users",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/users"`,
		},
		{
			// Matrix parameters (;session=123) remain part of the
			// path segment and fail exact matching with 404.
			name:       "matrix_param_rejected",
			rawPath:    "/v1/users;session=123",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
		{
			name:       "matrix_param_subpath_rejected",
			rawPath:    "/v1/users;session=123/info",
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

// TestMatrixParametersSingleWildcard verifies that matrix parameters (;k=v)
// within a single segment match a single-wildcard route (/v1/*) and are
// forwarded unchanged, while extra segments return HTTP 404 Not Found.
func TestMatrixParametersSingleWildcard(t *testing.T) {
	s := env.NewTestEnv(platform.TestAPIGWPathMutationMatrix, platform.EchoSidecar)
	defer s.TearDown(t)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "WildcardInfo"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".WildcardInfo",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/*"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".WildcardInfo", AllowUnregisteredCalls: true},
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
			name:       "trailing_matrix_param",
			rawPath:    "/v1/info;filter=active",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/info;filter=active"`,
		},
		{
			name:       "standard_segment",
			rawPath:    "/v1/info",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/info"`,
		},
		{
			name:       "extra_segment_rejected",
			rawPath:    "/v1/info;filter=active/extra",
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
