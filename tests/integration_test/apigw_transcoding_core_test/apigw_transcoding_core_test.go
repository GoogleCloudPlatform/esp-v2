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

// Package apigw_transcoding_core_test verifies API Gateway core gRPC
// transcoding, HTTP REST, gRPC-Web, and backend token injection across exact,
// prefix/wildcard, and custom-verb routes.
package apigw_transcoding_core_test

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

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var sharedEnv *env.TestEnv

// TestMain runs the test suite and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestExactOpenHTTPREST verifies HTTP REST transcoding on an exact route.
func TestExactOpenHTTPREST(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves) failed: %v", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestExactOpenGRPCWeb verifies gRPC-Web calls on an exact route.
func TestExactOpenGRPCWeb(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, trailer, err := client.MakeGRPCWebCall(s.ListenerAddr(), "ListShelves", "", nil)
	if err != nil {
		t.Fatalf("client.MakeGRPCWebCall(%q) failed: %v", "ListShelves", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeGRPCWebCall(%q) JSON mismatch (got %q, want %q):\n%v", "ListShelves", resp, want, err)
	}
	if got := trailer["grpc-status"]; got != "0" {
		t.Errorf("client.MakeGRPCWebCall(%q) grpc-status = %q, want %q", "ListShelves", got, "0")
	}
}

// TestExactOpenGRPCTranscoding verifies gRPC transcoding on an exact route.
func TestExactOpenGRPCTranscoding(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves) failed: %v", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestExactOpenTokenInjection verifies backend token injection on an exact
// route.
func TestExactOpenTokenInjection(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves with backend auth) failed: %v", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestExactMultiMethodHTTPREST verifies multiple explicit HTTP methods on an
// exact route and 405 rejection for disallowed methods.
func TestExactMultiMethodHTTPREST(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	baseURL := s.ListenerURL("/v1/shelves")

	_, statusGet, bodyGet, err := utils.DoHTTP(t.Context(), baseURL, http.MethodGet, "", nil)
	if err != nil || statusGet != http.StatusOK {
		t.Errorf("GET /v1/shelves status = %d, err = %v, want %d (body: %q)", statusGet, err, http.StatusOK, bodyGet)
	}

	_, statusPost, bodyPost, err := utils.DoHTTP(t.Context(), baseURL, http.MethodPost, `{"id":301,"theme":"History"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || statusPost != http.StatusOK {
		t.Errorf("POST /v1/shelves status = %d, err = %v, want %d (body: %q)", statusPost, err, http.StatusOK, bodyPost)
	} else {
		t.Cleanup(func() {
			delURL := s.ListenerURL("/v1/shelves/301")
			_, status, body, err := utils.DoHTTP(context.Background(), delURL, http.MethodDelete, "", nil)
			if err != nil || status != http.StatusOK {
				t.Errorf("cleanup DELETE /v1/shelves/301 status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
			}
		})
	}

	_, statusPut, bodyPut, err := utils.DoHTTP(t.Context(), baseURL, http.MethodPut, `{}`, nil)
	if err != nil || statusPut != http.StatusMethodNotAllowed {
		t.Errorf("PUT /v1/shelves status = %d, err = %v, want %d (body: %q)", statusPut, err, http.StatusMethodNotAllowed, bodyPut)
	}
}

// TestPrefixWildcardOpenHTTPREST verifies HTTP REST transcoding on a
// prefix/wildcard route.
func TestPrefixWildcardOpenHTTPREST(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves/100", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves/100) failed: %v", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves/100 JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestPrefixWildcardOpenGRPCWeb verifies gRPC-Web calls on a prefix/wildcard
// route.
func TestPrefixWildcardOpenGRPCWeb(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, trailer, err := client.MakeGRPCWebCall(s.ListenerAddr(), "GetShelf", "", nil)
	if err != nil {
		t.Fatalf("client.MakeGRPCWebCall(%q) failed: %v", "GetShelf", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeGRPCWebCall(%q) JSON mismatch (got %q, want %q):\n%v", "GetShelf", resp, want, err)
	}
	if got := trailer["grpc-status"]; got != "0" {
		t.Errorf("client.MakeGRPCWebCall(%q) grpc-status = %q, want %q", "GetShelf", got, "0")
	}
}

// TestPrefixWildcardOpenGRPCTranscoding verifies gRPC transcoding on a
// prefix/wildcard route.
func TestPrefixWildcardOpenGRPCTranscoding(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves/200", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves/200) failed: %v", err)
	}
	want := `{"id":"200","theme":"Classic"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves/200 JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestPrefixWildcardOpenTokenInjection verifies backend token injection on a
// prefix/wildcard route.
func TestPrefixWildcardOpenTokenInjection(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	resp, err := client.MakeCall("http", s.ListenerAddr(), http.MethodGet, "/v1/shelves/100", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("client.MakeCall(GET /v1/shelves/100 with backend auth) failed: %v", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("GET /v1/shelves/100 JSON mismatch (got %q, want %q):\n%v", resp, want, err)
	}
}

// TestPrefixWildcardMultiMethodHTTPREST verifies multiple explicit HTTP
// methods on a prefix/wildcard route without mutating shared fixture shelves.
func TestPrefixWildcardMultiMethodHTTPREST(t *testing.T) {
	s := setupTranscodingCoreEnv(t)

	// Read fixture shelf 100 (BookstoreServerV1Impl.CreateShelf does not
	// persist new shelves for GetShelf), then create and delete a
	// temporary shelf 302 so fixture shelf 100 is never deleted on
	// sharedEnv.
	_, statusGet, bodyGet, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves/100"), http.MethodGet, "", nil)
	if err != nil || statusGet != http.StatusOK {
		t.Errorf("GET /v1/shelves/100 status = %d, err = %v, want %d (body: %q)", statusGet, err, http.StatusOK, bodyGet)
	}

	_, statusCreate, bodyCreate, err := utils.DoHTTP(t.Context(), s.ListenerURL("/v1/shelves"), http.MethodPost, `{"id":302,"theme":"Temp"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || statusCreate != http.StatusOK {
		t.Fatalf("POST /v1/shelves (temp shelf 302) status = %d, err = %v, want %d (body: %q)", statusCreate, err, http.StatusOK, bodyCreate)
	}

	tempShelfURL := s.ListenerURL("/v1/shelves/302")
	_, statusDel, bodyDel, err := utils.DoHTTP(t.Context(), tempShelfURL, http.MethodDelete, "", nil)
	if err != nil || statusDel != http.StatusOK {
		t.Errorf("DELETE /v1/shelves/302 status = %d, err = %v, want %d (body: %q)", statusDel, err, http.StatusOK, bodyDel)
	}

	_, statusPut, bodyPut, err := utils.DoHTTP(t.Context(), tempShelfURL, http.MethodPut, `{}`, nil)
	if err != nil || statusPut != http.StatusMethodNotAllowed {
		t.Errorf("PUT /v1/shelves/302 status = %d, err = %v, want %d (body: %q)", statusPut, err, http.StatusMethodNotAllowed, bodyPut)
	}
}

// TestPathVarCustomVerbGRPCTranscoding verifies gRPC transcoding on a path
// variable route with a registered custom verb.
func TestPathVarCustomVerbGRPCTranscoding(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	targetURL := s.ListenerURL("/v1/shelves/200:registeredCustomVerb")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"id":20060,"title":"CustomVerbBook"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/shelves/200:registeredCustomVerb status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"id":"20060","title":"CustomVerbBook"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /v1/shelves/200:registeredCustomVerb JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestPathVarCustomVerbTokenInjection verifies backend token injection on a
// path variable route with a registered custom verb.
func TestPathVarCustomVerbTokenInjection(t *testing.T) {
	s := setupTranscodingCoreEnv(t)
	targetURL := s.ListenerURL("/v1/shelves/200:registeredCustomVerb")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"id":20061,"title":"CustomVerbAuthBook"}`, map[string]string{
		"Content-Type":           "application/json",
		"x-require-backend-auth": "true",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/shelves/200:registeredCustomVerb with backend auth status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"id":"20061","title":"CustomVerbAuthBook"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /v1/shelves/200:registeredCustomVerb JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

func setupTranscodingCoreEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWTranscodingCore, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})

	const svc = "endpoints.examples.bookstore.Bookstore"
	methods := []string{
		"ListShelves",
		"CreateShelf",
		"GetShelf",
		"DeleteShelf",
		"CreateBookWithCustomVerb",
	}
	var usageRules []*confpb.UsageRule
	var backendRules []*confpb.BackendRule
	for _, m := range methods {
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               svc + "." + m,
			AllowUnregisteredCalls: true,
		})
		backendRules = append(backendRules, &confpb.BackendRule{
			Selector:       svc + "." + m,
			Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "https://bookstore.example.com/" + m},
		})
	}
	s.OverrideUsage(&confpb.Usage{Rules: usageRules})
	s.AppendBackendRules(backendRules)

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(utils.CommonArgs()) failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}
