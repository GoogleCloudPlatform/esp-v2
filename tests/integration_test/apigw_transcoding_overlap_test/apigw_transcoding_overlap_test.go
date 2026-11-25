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

// Package apigw_transcoding_overlap_test verifies API Gateway routing for
// overlapping static, wildcard, and custom-verb routes under raw path
// matching.
package apigw_transcoding_overlap_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const bookstoreSvc = "endpoints.examples.bookstore.Bookstore"

var sharedEnv *env.TestEnv

// TestMain runs the test suite and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestOverlappingRoutesHTTPREST verifies HTTP REST precedence between static,
// wildcard, and percent-encoded routes under raw path matching.
func TestOverlappingRoutesHTTPREST(t *testing.T) {
	s := setupTranscodingOverlapEnv(t)

	// Exact /v1/shelves/static resolves to ListShelves ahead of
	// /v1/shelves/{shelf}.
	_, statusStatic, bodyStatic, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/static"), http.MethodGet, "", nil)
	if err != nil || statusStatic != http.StatusOK {
		t.Fatalf("GET /v1/shelves/static status = %d, err = %v, want %d (body: %q)", statusStatic, err, http.StatusOK, bodyStatic)
	}
	wantStatic := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(wantStatic, bodyStatic); err != nil {
		t.Errorf("GET /v1/shelves/static JSON mismatch (got %q, want %q):\n%v", bodyStatic, wantStatic, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".ListShelves", 4); err != nil {
		t.Errorf("GET /v1/shelves/static WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".ListShelves", err)
	}

	// Parameterized /v1/shelves/100 resolves to GetShelf.
	_, statusWild, bodyWild, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/100"), http.MethodGet, "", nil)
	if err != nil || statusWild != http.StatusOK {
		t.Fatalf("GET /v1/shelves/100 status = %d, err = %v, want %d (body: %q)", statusWild, err, http.StatusOK, bodyWild)
	}
	wantWild := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(wantWild, bodyWild); err != nil {
		t.Errorf("GET /v1/shelves/100 JSON mismatch (got %q, want %q):\n%v", bodyWild, wantWild, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/100 WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}

	// Under --normalize_path=false, raw /v1/shelves/st%61tic does not
	// match static /v1/shelves/static; it routes to wildcard
	// /v1/shelves/{shelf} (GetShelf), where grpc_json_transcoder rejects
	// non-int64 shelf with 400.
	_, statusRaw, bodyRaw, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/st%61tic"), http.MethodGet, "", nil)
	if err != nil || statusRaw != http.StatusBadRequest {
		t.Fatalf("GET /v1/shelves/st%%61tic status = %d, err = %v, want %d (body: %q)", statusRaw, err, http.StatusBadRequest, bodyRaw)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/st%%61tic WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}
}

// TestOverlappingRoutesGRPCWeb verifies gRPC-Web calls succeed when
// overlapping HTTP routes are configured.
func TestOverlappingRoutesGRPCWeb(t *testing.T) {
	s := setupTranscodingOverlapEnv(t)
	addr := s.ListenerAddr()
	for _, method := range []string{"ListShelves", "GetShelf", "ListBooks", "GetBook"} {
		_, trailer, err := client.MakeGRPCWebCall(addr, method, "", nil)
		if err != nil {
			t.Errorf("client.MakeGRPCWebCall(%q) failed: %v", method, err)
			continue
		}
		if got := trailer["grpc-status"]; got != "0" {
			t.Errorf("client.MakeGRPCWebCall(%q) grpc-status = %q, want %q", method, got, "0")
		}
	}
}

// TestOverlappingRoutesGRPCTranscoding verifies gRPC transcoding precedence
// across overlapping static, wildcard, and percent-encoded routes.
func TestOverlappingRoutesGRPCTranscoding(t *testing.T) {
	s := setupTranscodingOverlapEnv(t)

	// Static /v1/shelves/static routes to ListShelves.
	_, statusStatic, bodyStatic, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/static"), http.MethodGet, "", nil)
	if err != nil || statusStatic != http.StatusOK {
		t.Fatalf("GET /v1/shelves/static status = %d, err = %v, want %d (body: %q)", statusStatic, err, http.StatusOK, bodyStatic)
	}
	wantStatic := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(wantStatic, bodyStatic); err != nil {
		t.Errorf("GET /v1/shelves/static JSON mismatch (got %q, want %q):\n%v", bodyStatic, wantStatic, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".ListShelves", 4); err != nil {
		t.Errorf("GET /v1/shelves/static WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".ListShelves", err)
	}

	// Parameterized /v1/shelves/200 routes to GetShelf.
	_, statusWild, bodyWild, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/200"), http.MethodGet, "", nil)
	if err != nil || statusWild != http.StatusOK {
		t.Fatalf("GET /v1/shelves/200 status = %d, err = %v, want %d (body: %q)", statusWild, err, http.StatusOK, bodyWild)
	}
	wantWild := `{"id":"200","theme":"Classic"}`
	if err := util.JsonEqual(wantWild, bodyWild); err != nil {
		t.Errorf("GET /v1/shelves/200 JSON mismatch (got %q, want %q):\n%v", bodyWild, wantWild, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/200 WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}

	// Under --normalize_path=false, /v1/shelves/st%61tic routes to wildcard
	// GetShelf (400 on non-int64 shelf).
	_, statusRaw, bodyRaw, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/st%61tic"), http.MethodGet, "", nil)
	if err != nil || statusRaw != http.StatusBadRequest {
		t.Fatalf("GET /v1/shelves/st%%61tic status = %d, err = %v, want %d (body: %q)", statusRaw, err, http.StatusBadRequest, bodyRaw)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/st%%61tic WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}
}

// TestOverlappingRoutesTokenInjection verifies per-route backend auth token
// injection differentiation across overlapping static and wildcard routes.
func TestOverlappingRoutesTokenInjection(t *testing.T) {
	s := setupTranscodingOverlapEnv(t)

	// Static /v1/shelves/static (ListShelves, has JwtAudience) injects
	// backend_auth token -> 200 OK.
	_, statusStatic, bodyStatic, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/static"), http.MethodGet, "", map[string]string{
		"x-require-backend-auth": "true",
	})
	if err != nil || statusStatic != http.StatusOK {
		t.Fatalf("GET /v1/shelves/static with backend auth status = %d, err = %v, want %d (body: %q)", statusStatic, err, http.StatusOK, bodyStatic)
	}
	wantStatic := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(wantStatic, bodyStatic); err != nil {
		t.Errorf("GET /v1/shelves/static JSON mismatch (got %q, want %q):\n%v", bodyStatic, wantStatic, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".ListShelves", 4); err != nil {
		t.Errorf("GET /v1/shelves/static WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".ListShelves", err)
	}

	// Parameterized /v1/shelves/100 (GetShelf, no JwtAudience) returns
	// 401 when x-require-backend-auth is set.
	_, statusWildAuth, bodyWildAuth, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/100"), http.MethodGet, "", map[string]string{
		"x-require-backend-auth": "true",
	})
	if err != nil || statusWildAuth != http.StatusUnauthorized {
		t.Fatalf("GET /v1/shelves/100 with backend auth status = %d, err = %v, want %d (body: %q)", statusWildAuth, err, http.StatusUnauthorized, bodyWildAuth)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/100 with backend auth WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}

	// Without x-require-backend-auth, /v1/shelves/100 succeeds on GetShelf.
	_, statusWildOpen, bodyWildOpen, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/100"), http.MethodGet, "", nil)
	if err != nil || statusWildOpen != http.StatusOK {
		t.Fatalf("GET /v1/shelves/100 without backend auth status = %d, err = %v, want %d (body: %q)", statusWildOpen, err, http.StatusOK, bodyWildOpen)
	}
	wantWild := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(wantWild, bodyWildOpen); err != nil {
		t.Errorf("GET /v1/shelves/100 JSON mismatch (got %q, want %q):\n%v", bodyWildOpen, wantWild, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/100 WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}

	// Under --normalize_path=false, /v1/shelves/st%61tic bypasses static
	// ListShelves and routes to GetShelf (400 from grpc_json_transcoder for
	// non-int64 shelf).
	_, statusRaw, bodyRaw, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/st%61tic"), http.MethodGet, "", map[string]string{
		"x-require-backend-auth": "true",
	})
	if err != nil || statusRaw != http.StatusBadRequest {
		t.Fatalf("GET /v1/shelves/st%%61tic with backend auth status = %d, err = %v, want %d (body: %q)", statusRaw, err, http.StatusBadRequest, bodyRaw)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".GetShelf", 4); err != nil {
		t.Errorf("GET /v1/shelves/st%%61tic WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".GetShelf", err)
	}
}

// TestOverlappingRoutesCustomVerbCreate verifies precedence between static and
// parameterized custom-verb routes under raw path matching.
func TestOverlappingRoutesCustomVerbCreate(t *testing.T) {
	s := setupTranscodingOverlapEnv(t)

	// Static POST /v1/shelves/100:create routes to CreateShelf.
	_, statusStatic, bodyStatic, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/100:create"), http.MethodPost, `{"id":100,"theme":"CustomVerbStatic"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || statusStatic != http.StatusOK {
		t.Fatalf("POST /v1/shelves/100:create status = %d, err = %v, want %d (body: %q)", statusStatic, err, http.StatusOK, bodyStatic)
	}
	t.Cleanup(func() {
		restoreURL := s.ListenerURL("/v1/shelves/100:create")
		_, status, body, err := utils.DoHTTP(context.Background(), restoreURL, http.MethodPost, `{"id":100,"theme":"Kids"}`, map[string]string{
			"Content-Type": "application/json",
		})
		if err != nil || status != http.StatusOK {
			t.Errorf("cleanup restore POST /v1/shelves/100:create status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
			return
		}
		if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".CreateShelf", 4); err != nil {
			t.Errorf("cleanup WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".CreateShelf", err)
		}
	})

	wantStatic := `{"id":"100","theme":"CustomVerbStatic"}`
	if err := util.JsonEqual(wantStatic, bodyStatic); err != nil {
		t.Errorf("POST /v1/shelves/100:create JSON mismatch (got %q, want %q):\n%v", bodyStatic, wantStatic, err)
	}
	if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".CreateShelf", 4); err != nil {
		t.Errorf("POST /v1/shelves/100:create WaitForReportedOperation(%q) failed: %v", bookstoreSvc+".CreateShelf", err)
	}

	// Parameterized POST /v1/shelves/200:create and raw percent-encoded
	// POST /v1/shelves/%3100:create (under --normalize_path=false) bypass
	// static /v1/shelves/100:create and route to CreateBookWithCustomVerb.
	for _, tc := range []struct {
		suffix string
		req    string
		want   string
	}{
		{
			suffix: "/v1/shelves/200:create",
			req:    `{"id":20090,"title":"OverlapCustomVerb"}`,
			want:   `{"id":"20090","title":"OverlapCustomVerb"}`,
		},
		{
			suffix: "/v1/shelves/%3100:create",
			req:    `{"id":10090,"title":"OverlapCustomVerbRaw"}`,
			want:   `{"id":"10090","title":"OverlapCustomVerbRaw"}`,
		},
	} {
		_, status, body, err := utils.DoHTTP(t.Context(), s.ListenerURL(tc.suffix), http.MethodPost, tc.req, map[string]string{
			"Content-Type": "application/json",
		})
		if err != nil || status != http.StatusOK {
			t.Errorf("POST %s status = %d, err = %v, want %d (body: %q)", tc.suffix, status, err, http.StatusOK, body)
			continue
		}
		if err := util.JsonEqual(tc.want, body); err != nil {
			t.Errorf("POST %s JSON mismatch (got %q, want %q):\n%v", tc.suffix, body, tc.want, err)
		}
		if _, err := s.ServiceControlServer.WaitForReportedOperation(bookstoreSvc+".CreateBookWithCustomVerb", 4); err != nil {
			t.Errorf("POST %s WaitForReportedOperation(%q) failed: %v", tc.suffix, bookstoreSvc+".CreateBookWithCustomVerb", err)
		}
	}
}

func setupTranscodingOverlapEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		sharedEnv.ServiceControlServer.DrainRequests()
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWTranscodingOverlap, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})

	methods := []string{
		"ListShelves",
		"CreateShelf",
		"GetShelf",
		"ListBooks",
		"GetBook",
		"CreateBook",
		"CreateBookWithCustomVerb",
	}
	var usageRules []*confpb.UsageRule
	var backendRules []*confpb.BackendRule
	for _, m := range methods {
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               bookstoreSvc + "." + m,
			AllowUnregisteredCalls: true,
		})
		// Omit JwtAudience on GetShelf so token-injection tests verify
		// per-route backend_auth differentiation between static
		// ListShelves (/v1/shelves/static) and wildcard GetShelf
		// (/v1/shelves/{shelf}).
		if m != "GetShelf" {
			backendRules = append(backendRules, &confpb.BackendRule{
				Selector:       bookstoreSvc + "." + m,
				Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "https://bookstore.example.com/" + m},
			})
		}
	}
	s.OverrideUsage(&confpb.Usage{Rules: usageRules})
	s.AppendBackendRules(backendRules)

	// Configure overlapping routes:
	// - Exact /v1/shelves/static on ListShelves (alongside /v1/shelves) vs
	//   /v1/shelves/{shelf=*} on GetShelf
	// - Overlapping custom verb /v1/shelves/100:create on CreateShelf vs
	//   /v1/shelves/{shelf}:create on CreateBookWithCustomVerb
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: bookstoreSvc + ".ListShelves",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shelves"},
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Get{Get: "/v1/shelves/static"},
				},
			},
		},
		{
			Selector: bookstoreSvc + ".CreateShelf",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/shelves/100:create"},
			Body:     "shelf",
		},
		{
			Selector: bookstoreSvc + ".CreateBookWithCustomVerb",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/shelves/{shelf}:registeredCustomVerb"},
			Body:     "book",
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Post{Post: "/v1/shelves/{shelf}:create"},
					Body:    "book",
				},
			},
		},
	})

	args := append([]string{"--normalize_path=false"}, utils.CommonArgs()...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	sharedEnv = s
	return sharedEnv
}
