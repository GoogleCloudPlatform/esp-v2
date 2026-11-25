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

// Package apigw_service_control_norm_test tests ESPv2 API Gateway path
// normalization and mutation edge cases with API key Service Control checks
// and reporting.
package apigw_service_control_norm_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const echoOperation = "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo"

// TestExactAPIKeyEscapedSlashes verifies that requests with escaped slashes
// (%2f) on an API-key-protected exact POST route (/echo) return HTTP 404 Not
// Found and emit a Service Control report, while canonical requests pass Check
// and Report.
func TestExactAPIKeyEscapedSlashes(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	t.Run("authorized_exact", func(t *testing.T) {
		reqURL := s.ListenerURL("/echo?key=api-key-72")
		resp, err := client.DoWithHeaders(reqURL, http.MethodPost, "test message", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, POST) failed: %v", reqURL, err)
		}
		if !strings.Contains(string(resp), "test message") {
			t.Errorf("client.DoWithHeaders(%q, POST) = %q, want substring %q", reqURL, string(resp), "test message")
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: echoOperation,
			ConsumerID:    "api_key:api-key-72",
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", reqURL, err)
		}
	})

	t.Run("escaped_slash_rejected", func(t *testing.T) {
		const rawPath = "/%2fecho?key=api-key-72"
		_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodPost, rawPath, `{"message":"test body"}`, nil)
		if err != nil {
			t.Fatalf("utils.DoRawPathHTTP(POST, %q) failed: %v", rawPath, err)
		}
		if gotStatus != http.StatusNotFound {
			t.Errorf("utils.DoRawPathHTTP(POST, %q) status = %d, want %d (body: %q)", rawPath, gotStatus, http.StatusNotFound, gotBody)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPStatus: http.StatusNotFound,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", rawPath, err)
		}
	})
}

// TestExactAPIKeyNullBytesAndCRLF verifies null-byte (%00) and CRLF (%0d%0a)
// rejection and Service Control error reporting on an API-key-protected exact
// POST route (/echo).
func TestExactAPIKeyNullBytesAndCRLF(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	testCases := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "null_byte",
			path:       "/echo%00payload?key=api-key-73",
			body:       `{"message":"null byte body"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			// On an exact route (/echo) with no wildcard segment,
			// ESPv2 path_matcher does not percent-decode %0d%0a as
			// a path variable and treats /echo%0d%0a... as an
			// unmatched path (404 Not Found), whereas wildcard
			// routes return 400 Bad Request.
			name:       "crlf",
			path:       "/echo%0d%0aInjected-Header:%20bad?key=api-key-73",
			body:       `{"message":"crlf body"}`,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodPost, tc.path, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoRawPathHTTP(POST, %q) failed: %v", tc.path, err)
			}
			if gotStatus != tc.wantStatus {
				t.Errorf("utils.DoRawPathHTTP(POST, %q) status = %d, want %d (body: %q)", tc.path, gotStatus, tc.wantStatus, gotBody)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
			}
			wantSC := utils.ServiceControlExpectation{
				HTTPStatus: tc.wantStatus,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", tc.path, err)
			}
		})
	}
}

// TestExactAPIKeyPathTraversal verifies that raw (/../) and percent-encoded
// (%2E%2E) dot-segment traversals resolving to an API-key-protected exact POST
// route (/echo) pass Service Control Check and Report.
func TestExactAPIKeyPathTraversal(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	testCases := []struct {
		name       string
		path       string
		apiKey     string
		body       string
		wantInBody string
	}{
		{
			name:       "raw_traversal",
			path:       "/other/../echo?key=api-key-74-raw",
			apiKey:     "api-key-74-raw",
			body:       `{"message":"traversal message"}`,
			wantInBody: "traversal message",
		},
		{
			name:       "percent_encoded_traversal",
			path:       "/foo/bar/%2E%2E/%2E%2E/echo?key=api-key-74-percent",
			apiKey:     "api-key-74-percent",
			body:       `{"message":"percent traversal message"}`,
			wantInBody: "percent traversal message",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodPost, tc.path, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoRawPathHTTP(POST, %q) failed: %v", tc.path, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoRawPathHTTP(POST, %q) status = %d, want %d (body: %q)", tc.path, gotStatus, http.StatusOK, gotBody)
			}
			if !strings.Contains(gotBody, tc.wantInBody) {
				t.Errorf("utils.DoRawPathHTTP(POST, %q) body = %q, want substring %q", tc.path, gotBody, tc.wantInBody)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(2)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: echoOperation,
				ConsumerID:    "api_key:" + tc.apiKey,
				HTTPStatus:    http.StatusOK,
			}
			if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
				t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", tc.path, err)
			}
		})
	}
}

// TestExactAPIKeyMatrixParams verifies that matrix parameters (;param=val) on
// an API-key-protected exact POST route (/echo) fail route matching with HTTP
// 404 Not Found and report the status to Service Control.
func TestExactAPIKeyMatrixParams(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	t.Run("canonical", func(t *testing.T) {
		reqURL := s.ListenerURL("/echo?key=api-key-75")
		resp, err := client.DoWithHeaders(reqURL, http.MethodPost, "matrix canonical message", nil)
		if err != nil {
			t.Fatalf("client.DoWithHeaders(%q, POST) failed: %v", reqURL, err)
		}
		if !strings.Contains(string(resp), "matrix canonical message") {
			t.Errorf("client.DoWithHeaders(%q, POST) = %q, want substring %q", reqURL, string(resp), "matrix canonical message")
		}

		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			OperationName: echoOperation,
			ConsumerID:    "api_key:api-key-75",
			HTTPStatus:    http.StatusOK,
		}
		if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
			t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", reqURL, err)
		}
	})

	t.Run("matrix_param", func(t *testing.T) {
		const rawPath = "/echo;param=val?key=api-key-75"
		_, gotStatus, gotBody, err := utils.DoRawPathHTTP(t.Context(), s.ListenerAddr(), http.MethodPost, rawPath, `{"message":"matrix body"}`, nil)
		if err != nil {
			t.Fatalf("utils.DoRawPathHTTP(POST, %q) failed: %v", rawPath, err)
		}
		if gotStatus != http.StatusNotFound {
			t.Errorf("utils.DoRawPathHTTP(POST, %q) status = %d, want %d (body: %q)", rawPath, gotStatus, http.StatusNotFound, gotBody)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1)
		if err != nil {
			t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
		}
		wantSC := utils.ServiceControlExpectation{
			HTTPStatus: http.StatusNotFound,
		}
		if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
			t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", rawPath, err)
		}
	})
}

// TestExactAPIKeyDoubleAndTrailingSlashes verifies that redundant leading and
// trailing slashes on an API-key-protected exact POST route (/echo) are
// normalized and pass Service Control Check and Report.
func TestExactAPIKeyDoubleAndTrailingSlashes(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	testCases := []struct {
		name   string
		path   string
		apiKey string
		body   string
	}{
		{
			name:   "double_slash",
			path:   "//echo?key=api-key-76-double",
			apiKey: "api-key-76-double",
			body:   "double slash message",
		},
		{
			name:   "triple_slash",
			path:   "///echo?key=api-key-76-triple",
			apiKey: "api-key-76-triple",
			body:   "triple slash message",
		},
		{
			name:   "trailing_slash",
			path:   "/echo/?key=api-key-76-trailing",
			apiKey: "api-key-76-trailing",
			body:   "trailing slash message",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := s.ListenerURL(tc.path)
			resp, err := client.DoWithHeaders(reqURL, http.MethodPost, tc.body, nil)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, POST) failed: %v", reqURL, err)
			}
			if !strings.Contains(string(resp), tc.body) {
				t.Errorf("client.DoWithHeaders(%q, POST) = %q, want substring %q", reqURL, string(resp), tc.body)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(2)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
			}
			wantSC := utils.ServiceControlExpectation{
				OperationName: echoOperation,
				ConsumerID:    "api_key:" + tc.apiKey,
				HTTPStatus:    http.StatusOK,
			}
			if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
				t.Errorf("utils.VerifyCheckAndReportRequests(POST %q) failed: %v", reqURL, err)
			}
		})
	}
}

// TestExactAPIKeyMalformedEncoding verifies that malformed percent-encodings
// (%ZZ and %2) on an API-key-protected exact POST route (/echo) return HTTP
// 404 Not Found and emit a Service Control report.
func TestExactAPIKeyMalformedEncoding(t *testing.T) {
	s := setupExactAPIKeyNormEnv(t)

	testCases := []struct {
		name   string
		rawReq string
	}{
		{
			name:   "invalid_hex",
			rawReq: "POST /echo%ZZ?key=api-key-77 HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
		},
		{
			name:   "incomplete_percent",
			rawReq: "POST /echo%2?key=api-key-77 HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statusLine, err := utils.SendRawTCPStatusLine(t.Context(), s.ListenerAddr(), tc.rawReq, 5*time.Second)
			if err != nil {
				t.Fatalf("utils.SendRawTCPStatusLine(%q) failed: %v", tc.name, err)
			}
			// On an exact route (/echo) with no wildcard segment,
			// ESPv2 path_matcher treats /echo%ZZ and /echo%2 as
			// unmatched paths and returns 404 Not Found with a
			// Service Control ReportRequest status of 404.
			if !strings.HasPrefix(statusLine, "HTTP/1.1 404") {
				t.Errorf("utils.SendRawTCPStatusLine(%q) = %q, want prefix %q", tc.name, statusLine, "HTTP/1.1 404")
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
			}
			wantSC := utils.ServiceControlExpectation{
				HTTPStatus: http.StatusNotFound,
			}
			if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", tc.name, err)
			}
		})
	}
}

func setupExactAPIKeyNormEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWServiceControlNorm, platform.EchoSidecar)
	t.Cleanup(func() { s.TearDown(t) })
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{Selector: echoOperation},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}
	return s
}
