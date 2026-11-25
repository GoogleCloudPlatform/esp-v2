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

// Package apigw_dataplane_payloads_test verifies API Gateway data-plane
// request body handling, including malformed JSON, oversized payloads,
// truncated chunked transfers, content-type mismatches, and gzip bodies.
package apigw_dataplane_payloads_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
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

// TestMalformedJSONPayload verifies that malformed JSON bodies are rejected
// with HTTP 400 while valid JSON succeeds with HTTP 200.
func TestMalformedJSONPayload(t *testing.T) {
	s := setupDataplanePayloadsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves")

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
		wantJSON   string
	}{
		{
			name:       "malformed_unclosed_json",
			reqBody:    `{"theme": "Kids"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid_json",
			reqBody:    `{"id": 901, "theme": "Kids"}`,
			wantStatus: http.StatusOK,
			wantJSON:   `{"id":"901","theme":"Kids"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, tc.reqBody, map[string]string{
				"Content-Type": "application/json",
			})
			if err != nil || status != tc.wantStatus {
				t.Fatalf("POST /v1/shelves (%s) status = %d, err = %v, want %d (body: %q)", tc.name, status, err, tc.wantStatus, body)
			}
			if tc.wantJSON != "" {
				if err := util.JsonEqual(tc.wantJSON, body); err != nil {
					t.Errorf("POST /v1/shelves (%s) JSON mismatch (got %q, want %q):\n%v", tc.name, body, tc.wantJSON, err)
				}
			}
		})
	}
}

// TestPayloadExceedingLimits verifies handling of payloads exceeding 30MB.
func TestPayloadExceedingLimits(t *testing.T) {
	s := setupDataplanePayloadsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves")

	const (
		payloadSize = 35 * 1024 * 1024
		prefix      = `{"theme":"`
		suffix      = `"}`
	)
	var b strings.Builder
	b.Grow(payloadSize)
	b.WriteString(prefix)
	padLen := payloadSize - len(prefix) - len(suffix)
	chunk := strings.Repeat("a", 64*1024)
	for padLen >= len(chunk) {
		b.WriteString(chunk)
		padLen -= len(chunk)
	}
	if padLen > 0 {
		b.WriteString(chunk[:padLen])
	}
	b.WriteString(suffix)
	largePayload := b.String()

	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, largePayload, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		t.Fatalf("POST /v1/shelves (35MB payload) failed: %v", err)
	}
	// A 35MB body either triggers Envoy's grpc_json_transcoder internal
	// buffer limit (HTTP 413 Payload Too Large) or streams to
	// GrpcBookstoreSidecar, which rejects it at the 4MB gRPC max receive
	// message size limit with codes.ResourceExhausted mapped to HTTP 429
	// Too Many Requests.
	if status != http.StatusTooManyRequests && status != http.StatusRequestEntityTooLarge {
		t.Errorf("POST /v1/shelves (35MB payload) status = %d, want %d or %d (body: %q)", status, http.StatusTooManyRequests, http.StatusRequestEntityTooLarge, body)
	}
}

// TestTruncatedChunkedBody verifies that an aborted chunked request does not
// impair proxy health for subsequent requests.
func TestTruncatedChunkedBody(t *testing.T) {
	s := setupDataplanePayloadsEnv(t)
	addr := s.ListenerAddr()

	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("DialContext(tcp, %q) failed: %v", addr, err)
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("conn.SetWriteDeadline failed: %v", err)
	}
	partialChunkedReq := fmt.Sprintf("POST /v1/shelves HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Transfer-Encoding: chunked\r\n"+
		"Content-Type: application/json\r\n"+
		"\r\n"+
		"10\r\n"+
		"{\"partial_data\":\r\n", addr)
	if _, err := conn.Write([]byte(partialChunkedReq)); err != nil {
		t.Fatalf("conn.Write(partial chunked request) failed: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("conn.Close() failed: %v", err)
	}

	// Verify proxy remains healthy and processes subsequent requests
	// normally.
	targetURL := s.ListenerURL("/v1/shelves")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"id": 902, "theme": "Subsequent"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("subsequent POST /v1/shelves status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"id":"902","theme":"Subsequent"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("subsequent POST /v1/shelves JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestContentTypeMismatch verifies that raw binary payloads declared as
// application/json are rejected with HTTP 400.
func TestContentTypeMismatch(t *testing.T) {
	s := setupDataplanePayloadsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves")
	rawBinary := string([]byte{0x00, 0xFF, 0xFE, 0xFD, 0x12, 0x34, 0x56, 0x78})

	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, rawBinary, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusBadRequest {
		t.Errorf("POST /v1/shelves (raw binary with application/json) status = %d, err = %v, want %d (body: %q)", status, err, http.StatusBadRequest, body)
	}
}

// TestGzipCompressedRequestBody verifies that gzip-compressed JSON bodies sent
// to gRPC transcoding are rejected with HTTP 400.
func TestGzipCompressedRequestBody(t *testing.T) {
	s := setupDataplanePayloadsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves")

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(`{"theme": "Kids"}`)); err != nil {
		t.Fatalf("gzip.Writer.Write failed: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip.Writer.Close failed: %v", err)
	}

	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, buf.String(), map[string]string{
		"Content-Type":     "application/json",
		"Content-Encoding": "gzip",
	})
	if err != nil || status != http.StatusBadRequest {
		t.Errorf("POST /v1/shelves (gzip body) status = %d, err = %v, want %d (body: %q)", status, err, http.StatusBadRequest, body)
	}
}

func setupDataplanePayloadsEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWDataplanePayloads, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "endpoints.examples.bookstore.Bookstore.CreateShelf",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(dataplane payloads env) failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}
