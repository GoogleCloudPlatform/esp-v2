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

// Package apigw_backend_auth_tokens_test verifies API Gateway backend
// authentication token injection across HTTP REST, gRPC-Web, and gRPC
// transcoding routes.
package apigw_backend_auth_tokens_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	echoclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	sharedEchoEnv            *env.TestEnv
	sharedEchoStrictEnv      *env.TestEnv
	sharedBookstoreEnv       *env.TestEnv
	sharedBookstoreStrictEnv *env.TestEnv
)

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownAllSharedEnvs(nil)
	os.Exit(code)
}

// TestExactOpenHTTPREST verifies backend token injection on an exact open HTTP
// REST route.
func TestExactOpenHTTPREST(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/exact")
	resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	want := `{"Authorization": "Bearer fake-token-exact", "RequestURI": "/bearertoken/exact"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
	}
}

// TestExactOpenTokenInjection verifies backend token injection on an exact open
// route.
func TestExactOpenTokenInjection(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/exact/inject")
	resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	want := `{"Authorization": "Bearer fake-token-exact-inject", "RequestURI": "/bearertoken/exact/inject"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
	}
}

// TestExactMultiMethodHTTPREST verifies per-method backend token injection on
// an exact path with multiple explicit HTTP methods and 405 on unconfigured
// methods.
func TestExactMultiMethodHTTPREST(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/multi")
	tests := []struct {
		name   string
		method string
		token  string
	}{
		{name: "get", method: http.MethodGet, token: "fake-token-multi-get"},
		{name: "post", method: http.MethodPost, token: "fake-token-multi-post"},
		{name: "put", method: http.MethodPut, token: "fake-token-multi-put"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := echoclient.DoWithHeaders(reqURL, tc.method, "", nil)
			if err != nil {
				t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, tc.method, err)
			}
			want := fmt.Sprintf(`{"Authorization": "Bearer %s", "RequestURI": "/bearertoken/multi"}`, tc.token)
			if err := util.JsonEqual(want, string(resp)); err != nil {
				t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, tc.method, err)
			}
		})
	}

	// Unconfigured DELETE on matched path returns 405 Method Not Allowed.
	t.Run("unconfigured_delete", func(t *testing.T) {
		_, status, body, err := utils.DoHTTP(t.Context(), reqURL, http.MethodDelete, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodDelete, err)
		}
		if status != http.StatusMethodNotAllowed {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodDelete, status, http.StatusMethodNotAllowed, body)
		}
	})
}

// TestPrefixWildcardOpenHTTPREST verifies backend token injection on an open
// prefix/wildcard HTTP REST route.
func TestPrefixWildcardOpenHTTPREST(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/prefix/123")
	resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	want := `{"Authorization": "Bearer fake-token-prefix", "RequestURI": "/bearertoken/prefix/123"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
	}
}

// TestPrefixWildcardOpenTokenInjection verifies backend token injection on an
// open prefix/wildcard route.
func TestPrefixWildcardOpenTokenInjection(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/prefix-inject/123")
	resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
	}
	want := `{"Authorization": "Bearer fake-token-prefix-inject", "RequestURI": "/bearertoken/prefix-inject/123"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
	}
}

// TestPrefixWildcardMultiMethodHTTPREST verifies per-method backend token
// injection on a prefix/wildcard path with multiple explicit HTTP methods and
// 405 on unconfigured methods.
func TestPrefixWildcardMultiMethodHTTPREST(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/prefix-multi/abc")
	tests := []struct {
		name   string
		method string
		token  string
	}{
		{name: "get", method: http.MethodGet, token: "fake-token-prefix-multi-get"},
		{name: "post", method: http.MethodPost, token: "fake-token-prefix-multi-post"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := echoclient.DoWithHeaders(reqURL, tc.method, "", nil)
			if err != nil {
				t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, tc.method, err)
			}
			want := fmt.Sprintf(`{"Authorization": "Bearer %s", "RequestURI": "/bearertoken/prefix-multi/abc"}`, tc.token)
			if err := util.JsonEqual(want, string(resp)); err != nil {
				t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, tc.method, err)
			}
		})
	}

	// Unconfigured DELETE on matched prefix-wildcard path returns 405 Method
	// Not Allowed.
	t.Run("unconfigured_delete", func(t *testing.T) {
		_, status, body, err := utils.DoHTTP(t.Context(), reqURL, http.MethodDelete, "", nil)
		if err != nil {
			t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodDelete, err)
		}
		if status != http.StatusMethodNotAllowed {
			t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodDelete, status, http.StatusMethodNotAllowed, body)
		}
	})
}

// TestPathVarCustomVerbTokenInjection verifies backend token injection on a
// path-variable route with a custom verb.
func TestPathVarCustomVerbTokenInjection(t *testing.T) {
	s := setupEchoBackendAuthEnv(t)
	reqURL := s.ListenerURL("/bearertoken/items/42:customAction")
	resp, err := echoclient.DoWithHeaders(reqURL, http.MethodPost, "", nil)
	if err != nil {
		t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodPost, err)
	}
	want := `{"Authorization": "Bearer fake-token-custom-verb", "RequestURI": "/bearertoken/items/42:customAction"}`
	if err := util.JsonEqual(want, string(resp)); err != nil {
		t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodPost, err)
	}
}

// TestOverlappingRoutesHTTPREST verifies overlapping static and wildcard HTTP
// REST route disambiguation and token injection under strict raw path routing.
func TestOverlappingRoutesHTTPREST(t *testing.T) {
	s := setupEchoBackendAuthStrictEnv(t)
	tests := []struct {
		name  string
		path  string
		token string
	}{
		{
			name:  "static",
			path:  "/bearertoken/overlap/static",
			token: "fake-token-overlap-static",
		},
		{
			// Under --normalize_path=false, percent-encoded %61 is not
			// normalized before routing, so /st%61tic bypasses static
			// /overlap/static and matches wildcard /overlap/{id}.
			name:  "raw_percent_encoded",
			path:  "/bearertoken/overlap/st%61tic",
			token: "fake-token-overlap-wild",
		},
		{
			name:  "dynamic_wildcard",
			path:  "/bearertoken/overlap/dynamic1",
			token: "fake-token-overlap-wild",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.path)
			resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
			}
			want := fmt.Sprintf(`{"Authorization": "Bearer %s", "RequestURI": "%s"}`, tc.token, tc.path)
			if err := util.JsonEqual(want, string(resp)); err != nil {
				t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
			}
		})
	}
}

// TestOverlappingRoutesTokenInjection verifies overlapping static and wildcard
// token injection under strict raw path routing.
func TestOverlappingRoutesTokenInjection(t *testing.T) {
	s := setupEchoBackendAuthStrictEnv(t)
	tests := []struct {
		name  string
		path  string
		token string
	}{
		{
			name:  "static",
			path:  "/bearertoken/overlap-inject/static",
			token: "fake-token-overlap-inject-static",
		},
		{
			// Under --normalize_path=false, /st%61tic routes to wildcard
			// /overlap-inject/{id}.
			name:  "raw_percent_encoded",
			path:  "/bearertoken/overlap-inject/st%61tic",
			token: "fake-token-overlap-inject-wild",
		},
		{
			name:  "dynamic_wildcard",
			path:  "/bearertoken/overlap-inject/dynamic1",
			token: "fake-token-overlap-inject-wild",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.path)
			resp, err := echoclient.DoWithHeaders(reqURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("echoclient.DoWithHeaders(%q, %q) failed: %v", reqURL, http.MethodGet, err)
			}
			want := fmt.Sprintf(`{"Authorization": "Bearer %s", "RequestURI": "%s"}`, tc.token, tc.path)
			if err := util.JsonEqual(want, string(resp)); err != nil {
				t.Errorf("echoclient.DoWithHeaders(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
			}
		})
	}
}

// TestExactOpenGRPCWeb verifies backend token injection for an exact open
// gRPC-Web route.
func TestExactOpenGRPCWeb(t *testing.T) {
	s := setupBookstoreBackendAuthEnv(t)
	addr := s.ListenerAddr()
	resp, _, err := bsclient.MakeGRPCWebCall(addr, "ListShelves", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("bsclient.MakeGRPCWebCall(%q, %q) failed: %v", addr, "ListShelves", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("bsclient.MakeGRPCWebCall(%q, %q) response mismatch: %v", addr, "ListShelves", err)
	}
}

// TestExactOpenGRPCTranscoding verifies backend token injection for an exact
// open gRPC transcoding route.
func TestExactOpenGRPCTranscoding(t *testing.T) {
	s := setupBookstoreBackendAuthEnv(t)
	addr := s.ListenerAddr()
	resp, err := bsclient.MakeCall("http", addr, http.MethodGet, "/v1/shelves", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("bsclient.MakeCall(%q, %q, %q) failed: %v", addr, http.MethodGet, "/v1/shelves", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("bsclient.MakeCall(%q, %q, %q) response mismatch: %v", addr, http.MethodGet, "/v1/shelves", err)
	}
}

// TestPrefixWildcardOpenGRPCWeb verifies backend token injection for a
// prefix/wildcard open gRPC-Web route.
func TestPrefixWildcardOpenGRPCWeb(t *testing.T) {
	s := setupBookstoreBackendAuthEnv(t)
	addr := s.ListenerAddr()
	resp, _, err := bsclient.MakeGRPCWebCall(addr, "GetShelf", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("bsclient.MakeGRPCWebCall(%q, %q) failed: %v", addr, "GetShelf", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("bsclient.MakeGRPCWebCall(%q, %q) response mismatch: %v", addr, "GetShelf", err)
	}
}

// TestPrefixWildcardOpenGRPCTranscoding verifies backend token injection for a
// prefix/wildcard open gRPC transcoding route.
func TestPrefixWildcardOpenGRPCTranscoding(t *testing.T) {
	s := setupBookstoreBackendAuthEnv(t)
	addr := s.ListenerAddr()
	resp, err := bsclient.MakeCall("http", addr, http.MethodGet, "/v1/shelves/100", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("bsclient.MakeCall(%q, %q, %q) failed: %v", addr, http.MethodGet, "/v1/shelves/100", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("bsclient.MakeCall(%q, %q, %q) response mismatch: %v", addr, http.MethodGet, "/v1/shelves/100", err)
	}
}

// TestPathVarCustomVerbGRPCTranscoding verifies backend token injection for a
// path-variable custom verb gRPC transcoding route.
func TestPathVarCustomVerbGRPCTranscoding(t *testing.T) {
	s := setupBookstoreBackendAuthEnv(t)
	reqURL := s.ListenerURL("/v1/shelves/200:registeredCustomVerb")
	_, status, body, err := utils.DoHTTP(t.Context(), reqURL, http.MethodPost, `{"id":20055,"title":"AuthBook"}`, map[string]string{
		"Content-Type":           "application/json",
		"x-require-backend-auth": "true",
	})
	if err != nil {
		t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodPost, err)
	}
	if status != http.StatusOK {
		t.Fatalf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodPost, status, http.StatusOK, body)
	}
	want := `{"id":"20055","title":"AuthBook"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("utils.DoHTTP(%q, %q) response mismatch: %v", reqURL, http.MethodPost, err)
	}
}

// TestOverlappingRoutesGRPCWeb verifies per-route backend_auth differentiation
// across overlapping gRPC-Web routes under strict raw path routing.
func TestOverlappingRoutesGRPCWeb(t *testing.T) {
	s := setupBookstoreBackendAuthStrictEnv(t)
	addr := s.ListenerAddr()

	tests := []struct {
		name       string
		method     string
		headers    http.Header
		wantStatus string
	}{
		{
			// ListBooks has JwtAudience configured -> backend_auth injects a
			// Bearer token and succeeds.
			name:       "list_books_with_backend_auth",
			method:     "ListBooks",
			headers:    http.Header{"x-require-backend-auth": []string{"true"}},
			wantStatus: "0",
		},
		{
			// GetBook omits JwtAudience -> backend rejects with UNAUTHENTICATED
			// (grpc-status 16) when x-require-backend-auth is set.
			name:       "get_book_missing_backend_auth",
			method:     "GetBook",
			headers:    http.Header{"x-require-backend-auth": []string{"true"}},
			wantStatus: "16",
		},
		{
			// GetBook without x-require-backend-auth succeeds normally.
			name:       "get_book_open",
			method:     "GetBook",
			wantStatus: "0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, trailer, err := bsclient.MakeGRPCWebCall(addr, tc.method, "", tc.headers)
			if err != nil {
				t.Fatalf("bsclient.MakeGRPCWebCall(%q, %q) failed: %v", addr, tc.method, err)
			}
			if got := trailer["grpc-status"]; got != tc.wantStatus {
				t.Errorf("bsclient.MakeGRPCWebCall(%q, %q) grpc-status = %q, want %q", addr, tc.method, got, tc.wantStatus)
			}
		})
	}
}

// TestOverlappingRoutesGRPCTranscoding verifies per-route backend_auth
// differentiation and Service Control operation reporting across overlapping
// gRPC transcoding routes under strict raw path routing.
func TestOverlappingRoutesGRPCTranscoding(t *testing.T) {
	s := setupBookstoreBackendAuthStrictEnv(t)

	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
		wantBody   string
		wantOp     string
	}{
		{
			// Static /v1/shelves/featured routes to ListShelves (has
			// JwtAudience) -> 200 OK with backend_auth token injected.
			name:       "static_featured_with_backend_auth",
			path:       "/v1/shelves/featured",
			headers:    map[string]string{"x-require-backend-auth": "true"},
			wantStatus: http.StatusOK,
			wantBody:   `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantOp:     "endpoints.examples.bookstore.Bookstore.ListShelves",
		},
		{
			// Parameterized /v1/shelves/100 routes to GetShelf (no JwtAudience)
			// -> 401 when x-require-backend-auth is set.
			name:       "wildcard_shelf_missing_backend_auth",
			path:       "/v1/shelves/100",
			headers:    map[string]string{"x-require-backend-auth": "true"},
			wantStatus: http.StatusUnauthorized,
			wantOp:     "endpoints.examples.bookstore.Bookstore.GetShelf",
		},
		{
			// Parameterized /v1/shelves/100 without x-require-backend-auth
			// succeeds normally on GetShelf.
			name:       "wildcard_shelf_open",
			path:       "/v1/shelves/100",
			wantStatus: http.StatusOK,
			wantBody:   `{"id":"100","theme":"Kids"}`,
			wantOp:     "endpoints.examples.bookstore.Bookstore.GetShelf",
		},
		{
			// Under --normalize_path=false, /v1/shelves/f%65atured does NOT
			// match static /v1/shelves/featured (ListShelves). It routes to
			// wildcard /v1/shelves/{shelf} (GetShelf), where
			// grpc_json_transcoder rejects the non-int64 shelf with 400.
			name:       "raw_percent_encoded_featured",
			path:       "/v1/shelves/f%65atured",
			headers:    map[string]string{"x-require-backend-auth": "true"},
			wantStatus: http.StatusBadRequest,
			wantOp:     "endpoints.examples.bookstore.Bookstore.GetShelf",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.path)
			_, status, body, err := utils.DoHTTP(t.Context(), reqURL, http.MethodGet, "", tc.headers)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, http.MethodGet, err)
			}
			if status != tc.wantStatus {
				t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, http.MethodGet, status, tc.wantStatus, body)
			} else if tc.wantBody != "" {
				if err := util.JsonEqual(tc.wantBody, body); err != nil {
					t.Errorf("utils.DoHTTP(%q, %q) response mismatch: %v", reqURL, http.MethodGet, err)
				}
			}
			if _, err := s.ServiceControlServer.WaitForReportedOperation(tc.wantOp, 4); err != nil {
				t.Errorf("ServiceControlServer.WaitForReportedOperation(%q, 4) failed: %v", tc.wantOp, err)
			}
		})
	}
}

func tearDownAllSharedEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedEchoEnv != nil {
		sharedEchoEnv.TearDown(t)
		sharedEchoEnv = nil
	}
	if sharedEchoStrictEnv != nil {
		sharedEchoStrictEnv.TearDown(t)
		sharedEchoStrictEnv = nil
	}
	if sharedBookstoreEnv != nil {
		sharedBookstoreEnv.TearDown(t)
		sharedBookstoreEnv = nil
	}
	if sharedBookstoreStrictEnv != nil {
		sharedBookstoreStrictEnv.TearDown(t)
		sharedBookstoreStrictEnv = nil
	}
}

func setupEchoBackendAuthEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEchoEnv != nil {
		return sharedEchoEnv
	}
	tearDownAllSharedEnvs(t)
	sharedEchoEnv = buildEchoBackendAuthEnv(t)
	return sharedEchoEnv
}

func setupEchoBackendAuthStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEchoStrictEnv != nil {
		return sharedEchoStrictEnv
	}
	tearDownAllSharedEnvs(t)
	sharedEchoStrictEnv = buildEchoBackendAuthEnv(t, "--normalize_path=false")
	return sharedEchoStrictEnv
}

func buildEchoBackendAuthEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWBackendAuthTokens, platform.EchoRemote)

	const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	backendRoutes := []struct {
		method   string
		audience string
		token    string
	}{
		{method: "ExactREST", audience: "https://example.com/exact", token: "fake-token-exact"},
		{method: "ExactInject", audience: "https://example.com/exact-inject", token: "fake-token-exact-inject"},
		{method: "MultiGet", audience: "https://example.com/multi-get", token: "fake-token-multi-get"},
		{method: "MultiPost", audience: "https://example.com/multi-post", token: "fake-token-multi-post"},
		{method: "MultiPut", audience: "https://example.com/multi-put", token: "fake-token-multi-put"},
		{method: "PrefixREST", audience: "https://example.com/prefix", token: "fake-token-prefix"},
		{method: "PrefixInject", audience: "https://example.com/prefix-inject", token: "fake-token-prefix-inject"},
		{method: "PrefixMultiGet", audience: "https://example.com/prefix-multi-get", token: "fake-token-prefix-multi-get"},
		{method: "PrefixMultiPost", audience: "https://example.com/prefix-multi-post", token: "fake-token-prefix-multi-post"},
		{method: "CustomVerbInject", audience: "https://example.com/custom-verb", token: "fake-token-custom-verb"},
		{method: "OverlapStatic", audience: "https://example.com/overlap-static", token: "fake-token-overlap-static"},
		{method: "OverlapWild", audience: "https://example.com/overlap-wild", token: "fake-token-overlap-wild"},
		{method: "OverlapInjectStatic", audience: "https://example.com/overlap-inject-static", token: "fake-token-overlap-inject-static"},
		{method: "OverlapInjectWild", audience: "https://example.com/overlap-inject-wild", token: "fake-token-overlap-inject-wild"},
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
			Selector: apiName + ".ExactREST",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/exact"},
		},
		{
			Selector: apiName + ".ExactInject",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/exact/inject"},
		},
		{
			Selector: apiName + ".MultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/multi"},
		},
		{
			Selector: apiName + ".MultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/bearertoken/multi"},
			Body:     "*",
		},
		{
			Selector: apiName + ".MultiPut",
			Pattern:  &annotationspb.HttpRule_Put{Put: "/bearertoken/multi"},
			Body:     "*",
		},
		{
			Selector: apiName + ".PrefixREST",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/prefix/{id}"},
		},
		{
			Selector: apiName + ".PrefixInject",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/prefix-inject/{id}"},
		},
		{
			Selector: apiName + ".PrefixMultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/prefix-multi/{id}"},
		},
		{
			Selector: apiName + ".PrefixMultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/bearertoken/prefix-multi/{id}"},
			Body:     "*",
		},
		{
			Selector: apiName + ".CustomVerbInject",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/bearertoken/items/{id}:customAction"},
			Body:     "*",
		},
		{
			Selector: apiName + ".OverlapStatic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/overlap/static"},
		},
		{
			Selector: apiName + ".OverlapWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/overlap/{id}"},
		},
		{
			Selector: apiName + ".OverlapInjectStatic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/overlap-inject/static"},
		},
		{
			Selector: apiName + ".OverlapInjectWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/bearertoken/overlap-inject/{id}"},
		},
	})

	args := append(extraArgs, utils.CommonArgs()...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) for EchoRemote failed: %v", args, err)
	}
	return s
}

func setupBookstoreBackendAuthEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedBookstoreEnv != nil {
		return sharedBookstoreEnv
	}
	tearDownAllSharedEnvs(t)
	authMethods := map[string]bool{
		"ListShelves":              true,
		"GetShelf":                 true,
		"CreateBookWithCustomVerb": true,
		"ListBooks":                true,
		"GetBook":                  true,
	}
	sharedBookstoreEnv = buildBookstoreBackendAuthEnv(t, authMethods)
	return sharedBookstoreEnv
}

func setupBookstoreBackendAuthStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedBookstoreStrictEnv != nil {
		sharedBookstoreStrictEnv.ServiceControlServer.DrainRequests()
		return sharedBookstoreStrictEnv
	}
	tearDownAllSharedEnvs(t)
	// For strict raw-path overlapping route tests, attach JwtAudience to
	// ListShelves and ListBooks while omitting JwtAudience on GetShelf and
	// GetBook to verify per-route backend_auth differentiation.
	authMethods := map[string]bool{
		"ListShelves": true,
		"ListBooks":   true,
	}
	sharedBookstoreStrictEnv = buildBookstoreBackendAuthEnv(t, authMethods, "--normalize_path=false")
	return sharedBookstoreStrictEnv
}

func buildBookstoreBackendAuthEnv(t *testing.T, authMethods map[string]bool, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWBackendAuthTokens, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})

	const svc = "endpoints.examples.bookstore.Bookstore"
	methods := []string{
		"ListShelves",
		"GetShelf",
		"CreateBookWithCustomVerb",
		"ListBooks",
		"GetBook",
	}
	var usageRules []*confpb.UsageRule
	var backendRules []*confpb.BackendRule
	for _, m := range methods {
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               svc + "." + m,
			AllowUnregisteredCalls: true,
		})
		if authMethods[m] {
			backendRules = append(backendRules, &confpb.BackendRule{
				Selector:       svc + "." + m,
				Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "https://bookstore.example.com/" + m},
			})
		}
	}
	s.OverrideUsage(&confpb.Usage{Rules: usageRules})
	s.AppendBackendRules(backendRules)
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".ListShelves",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shelves/featured"},
		},
	})

	args := append(extraArgs, utils.CommonArgs()...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) for GrpcBookstoreSidecar failed: %v", args, err)
	}
	return s
}
