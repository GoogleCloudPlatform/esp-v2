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

// Package apigw_grpc_web_cors_test verifies API Gateway unary gRPC-Web CORS
// preflight and collision routing.
package apigw_grpc_web_cors_test

import (
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

const (
	corsOrigin  = "http://cloud.google.com"
	corsMethods = "GET, POST, OPTIONS"
	corsHeaders = "content-type,x-grpc-web,x-user-agent"
	corsExpose  = "grpc-status,grpc-message"
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

// TestGRPCWebAutoCORSUnary verifies Envoy CORS preflight handling and unary
// gRPC-Web calls when no explicit OPTIONS rule is configured.
func TestGRPCWebAutoCORSUnary(t *testing.T) {
	s := setupGRPCWebCORSEnv(t)
	addr := s.ListenerAddr()

	// Preflight OPTIONS on /endpoints.examples.bookstore.Bookstore/GetShelf
	// (no explicit OPTIONS rule -> handled by Envoy CORS filter).
	preflightURL := s.ListenerURL("/endpoints.examples.bookstore.Bookstore/GetShelf")
	if respHeaders, status, body, err := utils.DoCORSPreflight(t.Context(), preflightURL, corsOrigin, http.MethodPost, "content-type,x-grpc-web"); err != nil {
		t.Errorf("utils.DoCORSPreflight(%q, %q) failed: %v", preflightURL, http.MethodPost, err)
	} else {
		if status != http.StatusOK {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", preflightURL, http.MethodPost, status, http.StatusOK, body)
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got != corsOrigin {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", preflightURL, http.MethodPost, got, corsOrigin)
		}
	}

	// Actual gRPC-Web call with Origin header succeeds.
	resp, trailer, err := client.MakeGRPCWebCall(addr, "GetShelf", "", http.Header{
		"Origin": []string{corsOrigin},
	})
	if err != nil {
		t.Fatalf("client.MakeGRPCWebCall(%q, %q) failed: %v", addr, "GetShelf", err)
	}
	want := `{"id":"100","theme":"Kids"}`
	if err := util.JsonEqual(want, resp); err != nil {
		t.Errorf("client.MakeGRPCWebCall(%q, %q) response mismatch: %v", addr, "GetShelf", err)
	}
	if got, wantStatus := trailer["grpc-status"], "0"; got != wantStatus {
		t.Errorf("client.MakeGRPCWebCall(%q, %q) grpc-status = %q, want %q", addr, "GetShelf", got, wantStatus)
	}
}

// TestGRPCWebCORSCollisionUnary verifies CORS preflight and unary gRPC-Web
// calls when an explicit OPTIONS HTTP rule collides with CORS handling.
func TestGRPCWebCORSCollisionUnary(t *testing.T) {
	s := setupGRPCWebCORSEnv(t)
	addr := s.ListenerAddr()

	// Explicit OPTIONS rule on
	// /endpoints.examples.bookstore.Bookstore/ListShelves.
	preflightURL := s.ListenerURL("/endpoints.examples.bookstore.Bookstore/ListShelves")
	if respHeaders, status, body, err := utils.DoCORSPreflight(t.Context(), preflightURL, corsOrigin, http.MethodPost, "content-type,x-grpc-web"); err != nil {
		t.Errorf("utils.DoCORSPreflight(%q, %q) failed: %v", preflightURL, http.MethodPost, err)
	} else {
		if status != http.StatusOK {
			t.Errorf("utils.DoCORSPreflight(%q, %q) status = %d, want %d (body: %q)", preflightURL, http.MethodPost, status, http.StatusOK, body)
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got != corsOrigin {
			t.Errorf("utils.DoCORSPreflight(%q, %q) Access-Control-Allow-Origin = %q, want %q", preflightURL, http.MethodPost, got, corsOrigin)
		}
	}

	// Actual gRPC-Web POST on
	// /endpoints.examples.bookstore.Bookstore/ListShelves succeeds.
	resp, trailer, err := client.MakeGRPCWebCall(addr, "ListShelves", "", http.Header{
		"Origin": []string{corsOrigin},
	})
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

func setupGRPCWebCORSEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWGRPCWebCORS, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "endpoints.examples.bookstore.Bookstore.ListShelves",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "endpoints.examples.bookstore.Bookstore.GetShelf",
				AllowUnregisteredCalls: true,
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: http.MethodOptions,
					Path: "/endpoints.examples.bookstore.Bookstore/ListShelves",
				},
			},
		},
	})

	args := append([]string{
		"--cors_preset=basic",
		"--cors_allow_origin=" + corsOrigin,
		"--cors_allow_methods=" + corsMethods,
		"--cors_allow_headers=" + corsHeaders,
		"--cors_expose_headers=" + corsExpose,
	}, utils.CommonArgs()...)

	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	sharedEnv = s
	return sharedEnv
}
