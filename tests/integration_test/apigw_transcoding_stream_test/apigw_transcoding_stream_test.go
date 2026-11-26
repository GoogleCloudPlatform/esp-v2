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

// Package apigw_transcoding_stream_test verifies API Gateway HTTP REST and
// gRPC transcoding streaming patterns, including client, server, and
// bidirectional streaming.
package apigw_transcoding_stream_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	sharedEchoEnv    *env.TestEnv
	sharedInteropEnv *env.TestEnv
)

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEchoEnv != nil {
		sharedEchoEnv.TearDown(nil)
	}
	if sharedInteropEnv != nil {
		sharedInteropEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestExactRESTClientStreaming verifies chunked HTTP REST client streaming on
// an exact route.
func TestExactRESTClientStreaming(t *testing.T) {
	s := setupEchoStreamEnv(t)
	targetURL := s.ListenerURL("/echo")
	status, body, err := doChunkedPostJSON(t.Context(), targetURL, []string{`{"message":"chunk1`, ` chunk2"}`})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /echo chunked status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"message":"chunk1 chunk2"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /echo chunked JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestExactRESTServerStreaming verifies server-sent events streaming on an
// exact HTTP REST route.
func TestExactRESTServerStreaming(t *testing.T) {
	s := setupEchoStreamEnv(t)
	targetURL := s.ListenerURL("/echostream")
	reqBody := `{"sse_chunks": [{"data": 1}, {"data": 2}]}`
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, targetURL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(%q) failed: %v", targetURL, err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /echostream failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading POST /echostream response failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /echostream status = %d, want %d (body: %q)", resp.StatusCode, http.StatusOK, string(body))
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("POST /echostream Content-Type = %q, want prefix %q", got, "text/event-stream")
	}
	wantBody := "data: {\"data\":1}\n\ndata: {\"data\":2}\n\n"
	if string(body) != wantBody {
		t.Errorf("POST /echostream SSE body = %q, want %q", string(body), wantBody)
	}
}

// TestMultiSegmentWildcardTranscodingClientStreaming verifies gRPC client
// streaming transcoding on a multi-segment wildcard route.
func TestMultiSegmentWildcardTranscodingClientStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/clientstream/a/b/c")
	// Base64("abc") = "YWJj" (3 bytes), Base64("defgh") = "ZGVmZ2g="
	// (5 bytes) -> aggregatedPayloadSize = 8.
	status, body, err := doChunkedPostJSON(t.Context(), targetURL, []string{
		`[{"payload":{"body":"YWJj"}},`,
		`{"payload":{"body":"ZGVmZ2g="}}]`,
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/clientstream/a/b/c status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"aggregatedPayloadSize":8}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /v1/clientstream/a/b/c JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestMultiSegmentWildcardTranscodingServerStreaming verifies gRPC server
// streaming transcoding on a multi-segment wildcard route.
func TestMultiSegmentWildcardTranscodingServerStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/serverstream/server/sub")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"responseParameters":[{"size":3},{"size":5},{"size":7}]}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/serverstream/server/sub status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	var msgs []map[string]any
	if err := json.Unmarshal([]byte(body), &msgs); err != nil {
		t.Fatalf("json.Unmarshal(POST /v1/serverstream/server/sub body %q) failed: %v", body, err)
	}
	if len(msgs) != 3 {
		t.Errorf("POST /v1/serverstream/server/sub message count = %d, want 3 (body: %q)", len(msgs), body)
	}
}

// TestMultiSegmentWildcardTranscodingBidiStreaming verifies gRPC bidirectional
// streaming transcoding on a multi-segment wildcard route.
func TestMultiSegmentWildcardTranscodingBidiStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/bidistream/bidi/x/y/z")
	status, body, err := doChunkedPostJSON(t.Context(), targetURL, []string{
		`[{"responseParameters":[{"size":4}]},`,
		`{"responseParameters":[{"size":6}]}]`,
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/bidistream/bidi/x/y/z status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	var msgs []map[string]any
	if err := json.Unmarshal([]byte(body), &msgs); err != nil {
		t.Fatalf("json.Unmarshal(POST /v1/bidistream/bidi/x/y/z body %q) failed: %v", body, err)
	}
	if len(msgs) != 2 {
		t.Errorf("POST /v1/bidistream/bidi/x/y/z message count = %d, want 2 (body: %q)", len(msgs), body)
	}
}

// TestExactTranscodingClientStreaming verifies gRPC client streaming
// transcoding on an exact route.
func TestExactTranscodingClientStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/clientstream")
	// Base64("test1") = "dGVzdDE=" (5 bytes), Base64("test2") = "dGVzdDI="
	// (5 bytes) -> aggregatedPayloadSize = 10.
	status, body, err := doChunkedPostJSON(t.Context(), targetURL, []string{
		`[{"payload":{"body":"dGVzdDE="}},`,
		`{"payload":{"body":"dGVzdDI="}}]`,
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/clientstream status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"aggregatedPayloadSize":10}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /v1/clientstream JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestExactTranscodingServerStreaming verifies gRPC server streaming
// transcoding on an exact route.
func TestExactTranscodingServerStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/serverstream")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"responseParameters":[{"size":2},{"size":4},{"size":6},{"size":8}]}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/serverstream status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	var msgs []map[string]any
	if err := json.Unmarshal([]byte(body), &msgs); err != nil {
		t.Fatalf("json.Unmarshal(POST /v1/serverstream body %q) failed: %v", body, err)
	}
	if len(msgs) != 4 {
		t.Errorf("POST /v1/serverstream message count = %d, want 4 (body: %q)", len(msgs), body)
	}
}

// TestExactTranscodingBidiStreaming verifies gRPC bidirectional streaming
// transcoding on an exact route.
func TestExactTranscodingBidiStreaming(t *testing.T) {
	s := setupInteropTranscodingStreamEnv(t)
	targetURL := s.ListenerURL("/v1/bidistream")
	status, body, err := doChunkedPostJSON(t.Context(), targetURL, []string{
		`[{"responseParameters":[{"size":3}]},`,
		`{"responseParameters":[{"size":5}]},`,
		`{"responseParameters":[{"size":7}]}]`,
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/bidistream status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	var msgs []map[string]any
	if err := json.Unmarshal([]byte(body), &msgs); err != nil {
		t.Fatalf("json.Unmarshal(POST /v1/bidistream body %q) failed: %v", body, err)
	}
	if len(msgs) != 3 {
		t.Errorf("POST /v1/bidistream message count = %d, want 3 (body: %q)", len(msgs), body)
	}
}

func setupEchoStreamEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedInteropEnv != nil {
		sharedInteropEnv.TearDown(t)
		sharedInteropEnv = nil
	}
	if sharedEchoEnv != nil {
		return sharedEchoEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWTranscodingStream, platform.EchoSidecar)
	s.AppendAPIs([]*apipb.Api{
		{
			Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
			Methods: []*apipb.Method{
				{Name: "EchoStream"},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoStream",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/echostream"},
			Body:     "*",
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.AppendUsageRules([]*confpb.UsageRule{
		{
			Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			AllowUnregisteredCalls: true,
		},
		{
			Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoStream",
			AllowUnregisteredCalls: true,
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(EchoSidecar stream env) failed: %v", err)
	}
	sharedEchoEnv = s
	return sharedEchoEnv
}

func setupInteropTranscodingStreamEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEchoEnv != nil {
		sharedEchoEnv.TearDown(t)
		sharedEchoEnv = nil
	}
	if sharedInteropEnv != nil {
		return sharedInteropEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWTranscodingStream, platform.GrpcInteropSidecar)
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "grpc.testing.TestService.StreamingInputCall",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/clientstream"},
			Body:     "*",
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Post{Post: "/v1/clientstream/**"},
					Body:    "*",
				},
			},
		},
		{
			Selector: "grpc.testing.TestService.StreamingOutputCall",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/serverstream"},
			Body:     "*",
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Post{Post: "/v1/serverstream/**"},
					Body:    "*",
				},
			},
		},
		{
			Selector: "grpc.testing.TestService.FullDuplexCall",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/bidistream"},
			Body:     "*",
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Post{Post: "/v1/bidistream/**"},
					Body:    "*",
				},
			},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(GrpcInteropSidecar stream env) failed: %v", err)
	}
	sharedInteropEnv = s
	return sharedInteropEnv
}

func doChunkedPostJSON(ctx context.Context, targetURL string, chunks []string) (int, string, error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, pr)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return 0, "", fmt.Errorf("creating chunked POST request for %q: %w", targetURL, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.TransferEncoding = []string{"chunked"}

	var wg sync.WaitGroup
	var writeErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, c := range chunks {
			if i > 0 {
				select {
				case <-ctx.Done():
					_ = pw.CloseWithError(ctx.Err())
					writeErr = ctx.Err()
					return
				case <-time.After(5 * time.Millisecond):
				}
			}
			if _, err := pw.Write([]byte(c)); err != nil {
				_ = pw.CloseWithError(err)
				writeErr = err
				return
			}
		}
		if err := pw.Close(); err != nil {
			writeErr = err
		}
	}()
	defer wg.Wait()
	defer pr.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("chunked POST %q: %w", targetURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	_ = pr.Close()
	wg.Wait()
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("reading chunked POST %q response: %w", targetURL, err)
	}
	if writeErr != nil {
		return resp.StatusCode, string(body), fmt.Errorf("writing chunked POST %q body: %w", targetURL, writeErr)
	}
	return resp.StatusCode, string(body), nil
}
