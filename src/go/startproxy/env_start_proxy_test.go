// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package startproxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func clearServerlessEnv() {
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("ENDPOINTS_SERVICE_NAME")
	_ = os.Unsetenv("ENDPOINTS_SERVICE_VERSION")
	_ = os.Unsetenv("ENDPOINTS_SERVICE_PATH")
	_ = os.Unsetenv("ESPv2_ARGS")
}

func TestEnvStartProxyMain(t *testing.T) {
	testcases := []struct {
		port        string
		service     string
		version     string
		servicePath string
		args        string
		wantArgs    []string
	}{
		{
			port:    "8080",
			service: "test_bookstore.goog.cloud",
			args:    "--http_request_timeout_s=1, --disable_tracing ",
			wantArgs: []string{
				"/apiproxy/start_proxy.py",
				"/apiproxy/start_proxy.py",
				"--on_serverless",
				"--http_port=8080",
				"--service=test_bookstore.goog.cloud",
				"--rollout_strategy=managed",
				"--http_request_timeout_s=1",
				" --disable_tracing ",
			},
		},
		{
			port:    "8082",
			service: "test_bookstore.goog.cloud",
			version: "2019-02-21r0",
			args:    "^++^--cors_preset=basic,++--cors_allow_origin=*",
			wantArgs: []string{
				"/apiproxy/start_proxy.py",
				"/apiproxy/start_proxy.py",
				"--on_serverless",
				"--http_port=8082",
				"--service=test_bookstore.goog.cloud",
				"--rollout_strategy=fixed",
				"--version=2019-02-21r0",
				"--cors_preset=basic,",
				"--cors_allow_origin=*",
			},
		},
		{
			port:        "8080",
			servicePath: "/tmp/service_config.json",
			args:        "--disable_tracing",
			wantArgs: []string{
				"/apiproxy/start_proxy.py",
				"/apiproxy/start_proxy.py",
				"--on_serverless",
				"--http_port=8080",
				"--rollout_strategy=fixed",
				"--service_json_path=/tmp/service_config.json",
				"--disable_tracing",
			},
		},
	}

	defer clearServerlessEnv()
	for i, tc := range testcases {
		clearServerlessEnv()
		_ = os.Setenv("PORT", tc.port)
		if tc.service != "" {
			_ = os.Setenv("ENDPOINTS_SERVICE_NAME", tc.service)
		}
		if tc.version != "" {
			_ = os.Setenv("ENDPOINTS_SERVICE_VERSION", tc.version)
		}
		if tc.servicePath != "" {
			_ = os.Setenv("ENDPOINTS_SERVICE_PATH", tc.servicePath)
		}
		_ = os.Setenv("ESPv2_ARGS", tc.args)

		gotArgs, err := GenServerlessArgs("/apiproxy/start_proxy.py")
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		if !reflect.DeepEqual(gotArgs, tc.wantArgs) {
			t.Errorf("case %d:\ngot  %v\nwant %v", i, gotArgs, tc.wantArgs)
		}
	}
}

func TestEnvStartProxyErrorsAndHandler(t *testing.T) {
	defer clearServerlessEnv()

	// Missing PORT -> error with serveErrPage=false
	clearServerlessEnv()
	_, _, serveErrPage, err := BuildServerlessFlags()
	if err == nil || serveErrPage {
		t.Fatalf("expected missing PORT error with serveErrPage=false, got err=%v serveErrPage=%v", err, serveErrPage)
	}

	// Missing ENDPOINTS_SERVICE_NAME and ENDPOINTS_SERVICE_PATH -> serveErrPage=true
	clearServerlessEnv()
	_ = os.Setenv("PORT", "8080")
	_, port, serveErrPage, err := BuildServerlessFlags()
	if err == nil || !serveErrPage || port != "8080" {
		t.Fatalf("expected missing service config error with serveErrPage=true on port 8080, got err=%v serveErrPage=%v port=%q", err, serveErrPage, port)
	}

	// Malformed ESPv2_ARGS (`^^--foo`) -> serveErrPage=true
	clearServerlessEnv()
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("ENDPOINTS_SERVICE_NAME", "test.goog.cloud")
	_ = os.Setenv("ESPv2_ARGS", "^^--disable_tracing")
	_, port, serveErrPage, err = BuildServerlessFlags()
	if err == nil || !serveErrPage || port != "8080" {
		t.Fatalf("expected malformed ESPv2_ARGS error with serveErrPage=true, got err=%v serveErrPage=%v", err, serveErrPage)
	}

	// Verify 503 error HTTP handler
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	MakeErrorHandler(MissingServiceConfigError).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
	if rec.Body.String() != MissingServiceConfigError {
		t.Errorf("unexpected response body: %q", rec.Body.String())
	}
}
