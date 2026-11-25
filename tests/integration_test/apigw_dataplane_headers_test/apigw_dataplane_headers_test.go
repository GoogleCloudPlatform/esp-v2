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

// Package apigw_dataplane_headers_test verifies API Gateway data-plane HTTP
// header handling, including spoofed user-info stripping, header size limits,
// request smuggling, hop-by-hop headers, UTF-8, and empty values.
package apigw_dataplane_headers_test

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
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

// TestSpoofedEndpointAPIUserInfo verifies that forged X-Endpoint-API-UserInfo
// headers are stripped for unauthenticated callers and overwritten for
// authenticated callers.
func TestSpoofedEndpointAPIUserInfo(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	targetURL := s.ListenerURL("/echoHeader")
	const maliciousUserInfo = `{"email":"malicious@attacker.com"}`

	tests := []struct {
		name         string
		headers      map[string]string
		wantUserInfo []string
	}{
		{
			name: "unauthenticated_strips_forged_header",
			headers: map[string]string{
				"X-Endpoint-API-UserInfo": maliciousUserInfo,
			},
		},
		{
			name: "authenticated_overwrites_forged_header",
			headers: map[string]string{
				"Authorization":           "Bearer " + testdata.Es256Token,
				"X-Endpoint-API-UserInfo": maliciousUserInfo,
			},
			wantUserInfo: []string{testdata.Es256TokenPayloadBase64},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			respHeaders, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", tc.headers)
			if err != nil || status != http.StatusOK {
				t.Fatalf("GET /echoHeader (%s) status = %d, err = %v, want %d (body: %q)", tc.name, status, err, http.StatusOK, body)
			}
			got := respHeaders.Values("Echo-X-Endpoint-Api-Userinfo")
			if !slices.Equal(got, tc.wantUserInfo) {
				t.Errorf("GET /echoHeader (%s) Echo-X-Endpoint-Api-Userinfo = %q, want %q", tc.name, got, tc.wantUserInfo)
			}
		})
	}
}

// TestLargeHTTPHeader verifies that normal headers succeed while headers
// exceeding 60KB are rejected with HTTP 431.
func TestLargeHTTPHeader(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	targetURL := s.ListenerURL("/echoHeader")

	tests := []struct {
		name       string
		headerName string
		headerSize int
		wantStatus int
	}{
		{
			name:       "within_limit_2kb",
			headerName: "X-Custom-Header",
			headerSize: 2 * 1024,
			wantStatus: http.StatusOK,
		},
		{
			name:       "exceeds_limit_62kb",
			headerName: "X-Large-Header",
			headerSize: 62 * 1024,
			wantStatus: http.StatusRequestHeaderFieldsTooLarge,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", map[string]string{
				tc.headerName: strings.Repeat("a", tc.headerSize),
			})
			if err != nil || status != tc.wantStatus {
				t.Errorf("GET /echoHeader (%s, %d bytes) status = %d, err = %v, want %d (body: %q)", tc.name, tc.headerSize, status, err, tc.wantStatus, body)
			}
		})
	}
}

// TestRequestSmugglingCLTE verifies that conflicting Content-Length and
// Transfer-Encoding headers are rejected with HTTP 400.
func TestRequestSmugglingCLTE(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	addr := s.ListenerAddr()
	rawReq := fmt.Sprintf("POST /echoHeader HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Content-Length: 5\r\n"+
		"Transfer-Encoding: chunked\r\n"+
		"\r\n"+
		"0\r\n\r\n", addr)

	resp, err := utils.SendRawTCP(t.Context(), addr, rawReq, 5*time.Second)
	if err != nil {
		t.Fatalf("utils.SendRawTCP(POST /echoHeader CL-TE) failed: %v", err)
	}
	const wantPrefix = "HTTP/1.1 400 Bad Request"
	if !strings.HasPrefix(resp, wantPrefix) {
		t.Errorf("utils.SendRawTCP(POST /echoHeader CL-TE) = %q, want prefix %q", resp, wantPrefix)
	}
}

// TestHopByHopHeaderStripping verifies that Connection and Connection-listed
// hop-by-hop headers are stripped while end-to-end headers are preserved.
func TestHopByHopHeaderStripping(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	addr := s.ListenerAddr()
	rawReq := fmt.Sprintf("GET /echoHeader HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Connection: close, X-Custom-Hop-By-Hop\r\n"+
		"X-Custom-Hop-By-Hop: strip-this-header\r\n"+
		"X-End-To-End-Header: preserve-this-header\r\n"+
		"\r\n", addr)

	resp, err := utils.SendRawTCP(t.Context(), addr, rawReq, 5*time.Second)
	if err != nil {
		t.Fatalf("utils.SendRawTCP(GET /echoHeader hop-by-hop) failed: %v", err)
	}
	const wantPrefix = "HTTP/1.1 200 OK"
	if !strings.HasPrefix(resp, wantPrefix) {
		t.Fatalf("utils.SendRawTCP(GET /echoHeader hop-by-hop) = %q, want prefix %q", resp, wantPrefix)
	}

	lowerResp := strings.ToLower(resp)
	if !strings.Contains(lowerResp, "echo-x-end-to-end-header: preserve-this-header") {
		t.Errorf("GET /echoHeader response = %q, want substring %q", resp, "echo-x-end-to-end-header: preserve-this-header")
	}
	if strings.Contains(lowerResp, "echo-connection") {
		t.Errorf("GET /echoHeader response = %q, want no %q header", resp, "echo-connection")
	}
	if strings.Contains(lowerResp, "echo-x-custom-hop-by-hop") {
		t.Errorf("GET /echoHeader response = %q, want no %q header", resp, "echo-x-custom-hop-by-hop")
	}
}

// TestMultiByteUTF8Headers verifies that multi-byte UTF-8 header values are
// forwarded unmodified.
func TestMultiByteUTF8Headers(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	targetURL := s.ListenerURL("/echoHeader")
	const utf8Value = "café-日本語-🚀-üöä"

	respHeaders, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", map[string]string{
		"X-UTF8-Header": utf8Value,
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET /echoHeader (UTF-8) status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	if got := respHeaders.Get("Echo-X-Utf8-Header"); got != utf8Value {
		t.Errorf("GET /echoHeader Echo-X-Utf8-Header = %q, want %q", got, utf8Value)
	}
}

// TestEmptyHeaderValues verifies that empty header values are forwarded
// alongside normal headers.
func TestEmptyHeaderValues(t *testing.T) {
	s := setupDataplaneHeadersEnv(t)
	targetURL := s.ListenerURL("/echoHeader")

	respHeaders, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", map[string]string{
		"X-Empty-Tag":  "",
		"X-Normal-Tag": "present",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET /echoHeader (empty header) status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	if got := respHeaders.Get("Echo-X-Normal-Tag"); got != "present" {
		t.Errorf("GET /echoHeader Echo-X-Normal-Tag = %q, want %q", got, "present")
	}
	vals := respHeaders.Values("Echo-X-Empty-Tag")
	if len(vals) != 1 || vals[0] != "" {
		t.Errorf("GET /echoHeader Echo-X-Empty-Tag = %q, want [%q]", vals, "")
	}
}

func setupDataplaneHeadersEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWDataplaneHeaders, platform.EchoSidecar)
	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector:               svc + ".EchoHeader",
				AllowWithoutCredential: true,
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               svc + ".EchoHeader",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(dataplane headers env) failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}
