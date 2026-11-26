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

// Package apigw_method_routing_test verifies HTTP method routing, API key
// enforcement, and strict path matching for ESPv2 API Gateway configurations.
package apigw_method_routing_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

var (
	sharedMethodRoutingEnv       *env.TestEnv
	sharedMethodRoutingStrictEnv *env.TestEnv
)

// TestMain runs the package tests and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownMethodRoutingEnvs(nil)
	os.Exit(code)
}

// TestExactOpenSingleMethod verifies routing on an open exact path with a
// single allowed HTTP method, returning 405 for unallowed methods and 404 for
// undefined paths.
func TestExactOpenSingleMethod(t *testing.T) {
	s := setupMethodRoutingEnv(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "allowed_get",
			method:     http.MethodGet,
			path:       "/v1/method/open",
			wantStatus: http.StatusOK,
		},
		{
			// An unallowed method on a matched path returns 405.
			name:       "unallowed_post",
			method:     http.MethodPost,
			path:       "/v1/method/open",
			body:       `{"message":"test"}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			// An undefined path returns 404.
			name:       "undefined_path",
			method:     http.MethodGet,
			path:       "/v1/method/nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, tc.method, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%s %q) failed: %v", tc.method, targetURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(%s %q) status = %d (body %q), want %d", tc.method, targetURL, gotStatus, body, tc.wantStatus)
			}
		})
	}
}

// TestExactOpenMultipleMethods verifies that multiple explicit methods on an
// open exact route succeed while an unconfigured method returns 405.
func TestExactOpenMultipleMethods(t *testing.T) {
	s := setupMethodRoutingEnv(t)

	multiURL := s.ListenerURL("/v1/method/multi")
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{name: "get", method: http.MethodGet, wantStatus: http.StatusOK},
		{name: "post", method: http.MethodPost, body: `{"message":"test"}`, wantStatus: http.StatusOK},
		{name: "put", method: http.MethodPut, body: `{"message":"test"}`, wantStatus: http.StatusOK},
		{name: "unallowed_delete", method: http.MethodDelete, wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), multiURL, tc.method, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%s %q) failed: %v", tc.method, multiURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(%s %q) status = %d (body %q), want %d", tc.method, multiURL, gotStatus, body, tc.wantStatus)
			}
		})
	}
}

// TestExactAPIKeySingleMethod verifies API key enforcement and strict
// percent-encoded path matching under --normalize_path=false.
func TestExactAPIKeySingleMethod(t *testing.T) {
	s := setupMethodRoutingStrictEnv(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			// A missing API key returns 401 Unauthorized.
			name:       "missing_api_key",
			path:       "/v1/method/key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid_api_key",
			path:       "/v1/method/key?key=valid-api-key",
			wantStatus: http.StatusOK,
		},
		{
			// Under --normalize_path=false, /v1/method/k%65y does
			// not match exact /v1/method/key and returns 404.
			name:       "percent_encoded_path",
			path:       "/v1/method/k%65y?key=valid-api-key",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, body, tc.wantStatus)
			}
		})
	}
}

func tearDownMethodRoutingEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedMethodRoutingEnv != nil {
		sharedMethodRoutingEnv.TearDown(t)
		sharedMethodRoutingEnv = nil
	}
	if sharedMethodRoutingStrictEnv != nil {
		sharedMethodRoutingStrictEnv.TearDown(t)
		sharedMethodRoutingStrictEnv = nil
	}
}

func buildMethodRoutingEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWMethodRouting, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	s.AppendAPIMethods([]*apipb.Method{
		{Name: "OpenSingleGet"},
		{Name: "KeySingleGet"},
		{Name: "OpenMultiGet"},
		{Name: "OpenMultiPost"},
		{Name: "OpenMultiPut"},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{
			Selector:               apiName + ".OpenSingleGet",
			AllowUnregisteredCalls: true,
		},
		{
			Selector:               apiName + ".KeySingleGet",
			AllowUnregisteredCalls: false,
		},
		{
			Selector:               apiName + ".OpenMultiGet",
			AllowUnregisteredCalls: true,
		},
		{
			Selector:               apiName + ".OpenMultiPost",
			AllowUnregisteredCalls: true,
		},
		{
			Selector:               apiName + ".OpenMultiPut",
			AllowUnregisteredCalls: true,
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".OpenSingleGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/method/open"},
		},
		{
			Selector: apiName + ".KeySingleGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/method/key"},
		},
		{
			Selector: apiName + ".OpenMultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/method/multi"},
		},
		{
			Selector: apiName + ".OpenMultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/method/multi"},
		},
		{
			Selector: apiName + ".OpenMultiPut",
			Pattern:  &annotationspb.HttpRule_Put{Put: "/v1/method/multi"},
		},
	})

	args := append(utils.CommonArgs(), extraArgs...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	return s
}

func setupMethodRoutingEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedMethodRoutingEnv != nil {
		return sharedMethodRoutingEnv
	}
	tearDownMethodRoutingEnvs(t)
	sharedMethodRoutingEnv = buildMethodRoutingEnv(t)
	return sharedMethodRoutingEnv
}

func setupMethodRoutingStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedMethodRoutingStrictEnv != nil {
		return sharedMethodRoutingStrictEnv
	}
	tearDownMethodRoutingEnvs(t)
	sharedMethodRoutingStrictEnv = buildMethodRoutingEnv(t, "--normalize_path=false")
	return sharedMethodRoutingStrictEnv
}
