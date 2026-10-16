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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// setServerlessEnv sets exactly the given serverless environment variables
// for the duration of the test.
func setServerlessEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, k := range []string{"PORT", "ENDPOINTS_SERVICE_NAME", "ENDPOINTS_SERVICE_VERSION", "ENDPOINTS_SERVICE_PATH", "ESPv2_ARGS"} {
		unsetEnvForTest(t, k)
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

// The expected flags are what env_start_proxy.py passed to start_proxy.py in
// env_start_proxy_test.py.
func TestServerlessFlags(t *testing.T) {
	testcases := []struct {
		desc string
		env  map[string]string
		want []string
	}{
		{
			desc: "managed rollout",
			env:  map[string]string{"PORT": "8080", "ENDPOINTS_SERVICE_NAME": "test_bookstore.goog.cloud", "ESPv2_ARGS": "--http_request_timeout_s=1, --disable_tracing "},
			want: []string{"--on_serverless", "--http_port=8080", "--service=test_bookstore.goog.cloud", "--rollout_strategy=managed", "--http_request_timeout_s=1", " --disable_tracing "},
		},
		{
			desc: "fixed rollout and custom delimiter",
			env:  map[string]string{"PORT": "8082", "ENDPOINTS_SERVICE_NAME": "test_bookstore.goog.cloud", "ENDPOINTS_SERVICE_VERSION": "2019-02-21r0", "ESPv2_ARGS": "^++^--cors_preset=basic,++--cors_allow_origin=*"},
			want: []string{"--on_serverless", "--http_port=8082", "--service=test_bookstore.goog.cloud", "--rollout_strategy=fixed", "--version=2019-02-21r0", "--cors_preset=basic,", "--cors_allow_origin=*"},
		},
		{
			desc: "service config file",
			env:  map[string]string{"PORT": "8080", "ENDPOINTS_SERVICE_PATH": "/tmp/service_config.json", "ESPv2_ARGS": "--disable_tracing"},
			want: []string{"--on_serverless", "--http_port=8080", "--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--disable_tracing"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			setServerlessEnv(t, tc.env)
			got, err := serverlessFlags()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestServerlessFlagsErrors(t *testing.T) {
	testcases := []struct {
		desc        string
		env         map[string]string
		wantPort    string
		wantWarning bool
		wantMsg     string
	}{
		{
			desc:    "no PORT: nowhere to serve an error page",
			env:     map[string]string{"ENDPOINTS_SERVICE_NAME": "test.goog.cloud"},
			wantMsg: "Serverless ESPv2 expects PORT in environment variables.\n",
		},
		{
			desc:        "no service: serve a warning",
			env:         map[string]string{"PORT": "8080"},
			wantPort:    "8080",
			wantWarning: true,
			wantMsg:     "Serverless ESPv2 expects ENDPOINTS_SERVICE_NAME in environment variables.\n" + missingServiceConfigMsg,
		},
		{
			desc:     "empty ESPv2_ARGS delimiter: serve an error",
			env:      map[string]string{"PORT": "8080", "ENDPOINTS_SERVICE_NAME": "test.goog.cloud", "ESPv2_ARGS": "^^--disable_tracing"},
			wantPort: "8080",
			wantMsg:  malformedESPv2ArgsMsg,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			setServerlessEnv(t, tc.env)
			_, err := serverlessFlags()
			var se *serverlessError
			if !errors.As(err, &se) {
				t.Fatalf("got %v, want a *serverlessError", err)
			}
			if se.port != tc.wantPort || se.warning != tc.wantWarning || se.msg != tc.wantMsg {
				t.Errorf("got %+v, want port=%q warning=%v msg=%q", *se, tc.wantPort, tc.wantWarning, tc.wantMsg)
			}
		})
	}
}

func TestErrorPageHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	errorPageHandler("broken").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != "broken" || rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("got %d %q %q, want 503 text/plain \"broken\"", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

// TestRunServerless verifies the serverless image's entrypoint: the flags come
// from the environment, and container arguments are ignored, as
// env_start_proxy.py ignored them.
func TestRunServerless(t *testing.T) {
	setServerlessEnv(t, map[string]string{"PORT": "8080", "ENDPOINTS_SERVICE_NAME": "test.goog.cloud", "ESPv2_ARGS": "--enable_debug"})
	h := newHarness(t, func(dir string) (string, string) {
		cm := fmt.Sprintf("#!/bin/sh\ntrap 'exit 0' TERM INT\nprintf '%%s\\n' \"$@\" > %s\necho $$ > %s\nwhile true; do sleep 1 & wait $!; done\n",
			filepath.Join(dir, "cm.args"), filepath.Join(dir, "cm.pid"))
		return cm, trappingChild(filepath.Join(dir, "envoy.pid"), "exit 0")
	})
	h.start(serverlessModeArg, "--bogus", "ignored")
	sigCh := h.signals()
	h.waitForPid("cm.pid")
	h.waitForPid("envoy.pid")

	args, err := parseArgs([]string{"--on_serverless", "--http_port=8080", "--service=test.goog.cloud", "--rollout_strategy=managed", "--enable_debug"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := h.l.genProxyConfig(args)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(h.path("cm.args"))
	if err != nil {
		t.Fatal(err)
	}
	if gotArgs := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n"); !reflect.DeepEqual(gotArgs, want[1:]) {
		t.Errorf("Config Manager args:\ngot  %q\nwant %q", gotArgs, want[1:])
	}

	sigCh <- syscall.SIGTERM
	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
}

// TestRunServerlessServesErrorPage verifies that a misconfigured serverless
// deployment answers requests with the problem until it is told to stop.
func TestRunServerlessServesErrorPage(t *testing.T) {
	port := freePort(t)
	setServerlessEnv(t, map[string]string{"PORT": port})
	h := newHarness(t, func(string) (string, string) { return "", "" })
	h.start(serverlessModeArg)
	sigCh := h.signals()

	var resp *http.Response
	for deadline := time.Now().Add(5 * time.Second); ; {
		var err error
		if resp, err = http.Get("http://127.0.0.1:" + port + "/"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("error page was not served: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "expects ENDPOINTS_SERVICE_NAME") {
		t.Errorf("got %d %q, want 503 explaining the missing ENDPOINTS_SERVICE_NAME", resp.StatusCode, body)
	}

	sigCh <- syscall.SIGTERM
	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
}

func TestRunServerlessWithoutPort(t *testing.T) {
	setServerlessEnv(t, map[string]string{"ENDPOINTS_SERVICE_NAME": "test.goog.cloud"})
	h := newHarness(t, func(string) (string, string) { return "", "" })
	h.start(serverlessModeArg)
	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
}
