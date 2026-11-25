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

// Package apigw_path_mutation_injection_test tests ESPv2 API Gateway handling
// of path mutation injection, malformed percent-encoding, overlong URI
// segments, and Unicode normalization.
package apigw_path_mutation_injection_test

import (
	"fmt"
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

// TestNullByteInjection verifies that requests containing encoded null bytes
// (%00) on an exact route (/v1/items) are rejected with HTTP 400 Bad Request.
func TestNullByteInjection(t *testing.T) {
	s := setupInjectionEnv(t)

	const baselinePath = "/v1/items"
	_, statusOK, bodyOK, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, baselinePath, "", nil)
	if err != nil {
		t.Fatalf("utils.DoRawPathHTTP(GET, %q) failed: %v", baselinePath, err)
	}
	const wantBaselineBody = `"RequestURI": "/v1/items"`
	if statusOK != http.StatusOK || !strings.Contains(bodyOK, wantBaselineBody) {
		t.Fatalf("utils.DoRawPathHTTP(GET, %q) = (%d, %q), want (%d, containing %q)", baselinePath, statusOK, bodyOK, http.StatusOK, wantBaselineBody)
	}

	for _, rawPath := range []string{"/v1/items%00/sub", "/v1/items%00"} {
		_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, rawPath, "", nil)
		if err != nil {
			t.Errorf("utils.DoRawPathHTTP(GET, %q) failed: %v", rawPath, err)
			continue
		}
		if gotStatus != http.StatusBadRequest {
			t.Errorf("utils.DoRawPathHTTP(GET, %q) status = %d, want %d (body: %q)", rawPath, gotStatus, http.StatusBadRequest, gotBody)
		}
	}
}

// TestCRLFHeaderSmuggling verifies that requests containing encoded CRLF
// sequences (%0D%0A) on an exact route (/v1/status) fail route matching and
// return HTTP 404 Not Found.
func TestCRLFHeaderSmuggling(t *testing.T) {
	s := setupInjectionEnv(t)

	const baselinePath = "/v1/status"
	_, statusOK, bodyOK, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, baselinePath, "", nil)
	if err != nil {
		t.Fatalf("utils.DoRawPathHTTP(GET, %q) failed: %v", baselinePath, err)
	}
	const wantBaselineBody = `"RequestURI": "/v1/status"`
	if statusOK != http.StatusOK || !strings.Contains(bodyOK, wantBaselineBody) {
		t.Fatalf("utils.DoRawPathHTTP(GET, %q) = (%d, %q), want (%d, containing %q)", baselinePath, statusOK, bodyOK, http.StatusOK, wantBaselineBody)
	}

	// On an exact route (/v1/status) without a wildcard variable segment,
	// ESPv2 path_matcher treats /v1/status%0d%0a... as an unmatched path
	// and returns 404 Not Found.
	for _, rawPath := range []string{
		"/v1/status%0d%0aInjected-Header:test",
		"/v1/status%0D%0AInjected-Header:test",
	} {
		_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, rawPath, "", nil)
		if err != nil {
			t.Errorf("utils.DoRawPathHTTP(GET, %q) failed: %v", rawPath, err)
			continue
		}
		if gotStatus != http.StatusNotFound {
			t.Errorf("utils.DoRawPathHTTP(GET, %q) status = %d, want %d (body: %q)", rawPath, gotStatus, http.StatusNotFound, gotBody)
		}
	}
}

// TestMalformedPercentEncoding verifies malformed percent-encoding handling
// in query parameters and path segments on an exact route (/v1/search).
func TestMalformedPercentEncoding(t *testing.T) {
	s := setupInjectionEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "valid_without_query",
			rawPath:    "/v1/search",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/search"`,
		},
		{
			name:       "non_hex_in_query",
			rawPath:    "/v1/search?q=%ZZ%GG",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/search?q=%ZZ%GG"`,
		},
		{
			name:       "incomplete_escape_in_query",
			rawPath:    "/v1/search?q=%Z",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/search?q=%Z"`,
		},
		{
			name:       "non_hex_in_path",
			rawPath:    "/v1/search%ZZ",
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

// TestOverlongURISegmentBuffer verifies that an overlong URI path segment on a
// single-wildcard route (/v1/overlong/*) is matched and forwarded to the
// backend without truncation.
func TestOverlongURISegmentBuffer(t *testing.T) {
	s := setupInjectionEnv(t)

	overlongSegment := strings.Repeat("a", 4500)
	overlongPath := "/v1/overlong/" + overlongSegment

	_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodGet, overlongPath, "", nil)
	if err != nil {
		t.Fatalf("utils.DoRawPathHTTP(GET, /v1/overlong/<%d bytes>) failed: %v", len(overlongSegment), err)
	}
	if gotStatus != http.StatusOK {
		t.Fatalf("utils.DoRawPathHTTP(GET, /v1/overlong/<%d bytes>) status = %d, want %d (body: %q)", len(overlongSegment), gotStatus, http.StatusOK, gotBody)
	}
	wantInBody := fmt.Sprintf(`"RequestURI": "%s"`, overlongPath)
	if !strings.Contains(gotBody, wantInBody) {
		t.Errorf("utils.DoRawPathHTTP(GET, /v1/overlong/<%d bytes>) body = %q, want substring %q", len(overlongSegment), gotBody, wantInBody)
	}
}

// TestUnicodeNormalizationNFCVsNFD verifies UTF-8 NFC versus NFD path matching
// on an exact route (/v1/caf%C3%A9), ensuring NFC paths match and NFD paths
// return HTTP 404 Not Found.
func TestUnicodeNormalizationNFCVsNFD(t *testing.T) {
	s := setupInjectionEnv(t)

	tests := []struct {
		name       string
		rawPath    string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "nfc_percent_encoded",
			rawPath:    "/v1/caf%C3%A9",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/caf%C3%A9"`,
		},
		{
			name:       "nfc_raw_utf8",
			rawPath:    "/v1/caf\xc3\xa9",
			wantStatus: http.StatusOK,
			wantInBody: `"RequestURI": "/v1/caf%C3%A9"`,
		},
		{
			name:       "nfd_percent_encoded",
			rawPath:    "/v1/cafe%CC%81",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
		{
			name:       "nfd_raw_utf8",
			rawPath:    "/v1/cafe\xcc\x81",
			wantStatus: http.StatusNotFound,
			wantInBody: "The current request is not defined by this API.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.rawPath)
			_, gotStatus, gotBody, err := utils.DoHTTP(t.Context(), reqURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET, %q) failed: %v", reqURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(GET, %q) status = %d, want %d (body: %q)", reqURL, gotStatus, tc.wantStatus, gotBody)
			}
			if tc.wantInBody != "" && !strings.Contains(gotBody, tc.wantInBody) {
				t.Errorf("utils.DoHTTP(GET, %q) body = %q, want substring %q", reqURL, gotBody, tc.wantInBody)
			}
		})
	}
}

func setupInjectionEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWPathMutationInjection, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "Items"},
		{Name: "Status"},
		{Name: "Search"},
		{Name: "OverlongWildcard"},
		{Name: "Cafe"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".Items",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/items"},
		},
		{
			Selector: svc + ".Status",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/status"},
		},
		{
			Selector: svc + ".Search",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/search"},
		},
		{
			Selector: svc + ".OverlongWildcard",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlong/*"},
		},
		{
			Selector: svc + ".Cafe",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/caf%C3%A9"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: svc + ".Items", AllowUnregisteredCalls: true},
			{Selector: svc + ".Status", AllowUnregisteredCalls: true},
			{Selector: svc + ".Search", AllowUnregisteredCalls: true},
			{Selector: svc + ".OverlongWildcard", AllowUnregisteredCalls: true},
			{Selector: svc + ".Cafe", AllowUnregisteredCalls: true},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup() failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}
