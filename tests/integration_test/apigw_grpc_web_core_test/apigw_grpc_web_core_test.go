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

// Package apigw_grpc_web_core_test verifies core API Gateway unary gRPC-Web,
// HTTP REST, gRPC transcoding, and backend token injection routes.
package apigw_grpc_web_core_test

import (
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

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestGRPCWebHTTPRESTUnary verifies unary HTTP REST routing against a gRPC
// backend.
func TestGRPCWebHTTPRESTUnary(t *testing.T) {
	s := setupGRPCWebCoreEnv(t)
	addr := s.ListenerAddr()
	resp, err := client.MakeCall("http", addr, http.MethodGet, "/v1/shelves", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(%q, %q, %q) failed: %v", addr, http.MethodGet, "/v1/shelves", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeCall(%q, %q, %q) response mismatch: %v", addr, http.MethodGet, "/v1/shelves", err)
	}
}

// TestGRPCWebUnary verifies unary gRPC-Web calls and OK trailer status.
func TestGRPCWebUnary(t *testing.T) {
	s := setupGRPCWebCoreEnv(t)
	addr := s.ListenerAddr()
	resp, trailer, err := client.MakeGRPCWebCall(addr, "ListShelves", "", nil)
	if err != nil {
		t.Fatalf("client.MakeGRPCWebCall(%q, %q) failed: %v", addr, "ListShelves", err)
	}
	want := `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeGRPCWebCall(%q, %q) response mismatch: %v", addr, "ListShelves", err)
	}
	if got, wantStatus := trailer["grpc-status"], "0"; got != wantStatus {
		t.Errorf("client.MakeGRPCWebCall(%q, %q) grpc-status = %q, want %q", addr, "ListShelves", got, wantStatus)
	}
}

// TestGRPCWebTranscodingUnary verifies unary gRPC JSON transcoding with path
// parameter extraction.
func TestGRPCWebTranscodingUnary(t *testing.T) {
	s := setupGRPCWebCoreEnv(t)
	addr := s.ListenerAddr()
	resp, err := client.MakeCall("http", addr, http.MethodGet, "/v1/shelves/100", "", nil)
	if err != nil {
		t.Fatalf("client.MakeCall(%q, %q, %q) failed: %v", addr, http.MethodGet, "/v1/shelves/100", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeCall(%q, %q, %q) response mismatch: %v", addr, http.MethodGet, "/v1/shelves/100", err)
	}
}

// TestGRPCWebTokenInjectionUnary verifies backend token injection on a unary
// gRPC-Web method with JwtAudience configured.
func TestGRPCWebTokenInjectionUnary(t *testing.T) {
	s := setupGRPCWebCoreEnv(t)
	addr := s.ListenerAddr()
	_, trailer, err := client.MakeGRPCWebCall(addr, "DeleteShelf", "", http.Header{
		"x-require-backend-auth": []string{"true"},
	})
	if err != nil {
		t.Fatalf("client.MakeGRPCWebCall(%q, %q) with backend auth failed: %v", addr, "DeleteShelf", err)
	}
	if got, want := trailer["grpc-status"], "0"; got != want {
		t.Errorf("client.MakeGRPCWebCall(%q, %q) grpc-status = %q, want %q", addr, "DeleteShelf", got, want)
	}
}

// TestGRPCWebMultiMethodHTTPREST verifies multiple explicit HTTP methods on
// /v1/shelves and 405 rejection of an unconfigured PUT method.
func TestGRPCWebMultiMethodHTTPREST(t *testing.T) {
	s := setupGRPCWebCoreEnv(t)
	reqURL := s.ListenerURL("/v1/shelves")

	tests := []struct {
		name       string
		method     string
		body       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "get",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "post",
			method:     http.MethodPost,
			body:       `{"id":300,"theme":"SciFi"}`,
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unconfigured_put",
			method:     http.MethodPut,
			body:       `{}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, status, body, err := utils.DoHTTP(t.Context(), reqURL, tc.method, tc.body, tc.headers)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%q, %q) failed: %v", reqURL, tc.method, err)
			}
			if status != tc.wantStatus {
				t.Errorf("utils.DoHTTP(%q, %q) status = %d, want %d (body: %q)", reqURL, tc.method, status, tc.wantStatus, body)
			}
		})
	}
}

func setupGRPCWebCoreEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWGRPCWebCore, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})

	const svc = "endpoints.examples.bookstore.Bookstore"
	methods := []string{
		"ListShelves",
		"CreateShelf",
		"GetShelf",
		"DeleteShelf",
	}
	var usageRules []*confpb.UsageRule
	for _, m := range methods {
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               svc + "." + m,
			AllowUnregisteredCalls: true,
		})
	}
	s.OverrideUsage(&confpb.Usage{Rules: usageRules})
	s.AppendBackendRules([]*confpb.BackendRule{
		{
			Selector:       svc + ".DeleteShelf",
			Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "https://bookstore.example.com/delete-shelf"},
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
