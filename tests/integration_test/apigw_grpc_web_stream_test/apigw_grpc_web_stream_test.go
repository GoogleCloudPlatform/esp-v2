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

// Package apigw_grpc_web_stream_test verifies API Gateway gRPC-Web client and
// server streaming with CORS as well as native gRPC bidirectional streaming.
package apigw_grpc_web_stream_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/protowire"
)

const streamCORSOrigin = "http://cloud.google.com"

var (
	sharedInteropCORSEnv *env.TestEnv
	sharedNativeBidiEnv  *env.TestEnv
)

// TestMain runs the test suite and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedInteropCORSEnv != nil {
		sharedInteropCORSEnv.TearDown(nil)
	}
	if sharedNativeBidiEnv != nil {
		sharedNativeBidiEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestGRPCWebClientStreamingAutoCORS verifies client-streaming gRPC-Web calls
// with auto-CORS headers enabled.
func TestGRPCWebClientStreamingAutoCORS(t *testing.T) {
	s := setupGRPCWebStreamCORSEnv(t)
	addr := s.ListenerAddr()
	const rpcPath = "/grpc.testing.TestService/StreamingInputCall"
	// Payloads of 3, 4, and 5 bytes sum to 12 bytes total.
	frames := [][]byte{
		encodeStreamingInputCallRequestWire([]byte("abc")),
		encodeStreamingInputCallRequestWire([]byte("defg")),
		encodeStreamingInputCallRequestWire([]byte("hijkl")),
	}

	headers, status, respFrames, trailer, err := sendGRPCWebFrames(t.Context(), addr, rpcPath, streamCORSOrigin, frames)
	if err != nil {
		t.Fatalf("sendGRPCWebFrames(%q, %q) failed: %v", addr, rpcPath, err)
	}
	if status != http.StatusOK {
		t.Fatalf("sendGRPCWebFrames(%q, %q) status = %d, want %d", addr, rpcPath, status, http.StatusOK)
	}
	if got := headers.Get("Access-Control-Allow-Origin"); got != streamCORSOrigin {
		t.Errorf("sendGRPCWebFrames(%q, %q) Access-Control-Allow-Origin = %q, want %q", addr, rpcPath, got, streamCORSOrigin)
	}
	if len(respFrames) != 1 {
		t.Errorf("sendGRPCWebFrames(%q, %q) response frame count = %d, want 1", addr, rpcPath, len(respFrames))
	} else {
		// StreamingInputCallResponse field 1 (aggregated_payload_size = 12) ->
		// wire bytes [0x08, 12].
		wantWire := []byte{0x08, 12}
		if !bytes.Equal(respFrames[0], wantWire) {
			t.Errorf("sendGRPCWebFrames(%q, %q) wire bytes = %v, want %v (aggregated_payload_size=12)", addr, rpcPath, respFrames[0], wantWire)
		}
	}
	if got, want := trailer["grpc-status"], "0"; got != want {
		t.Errorf("sendGRPCWebFrames(%q, %q) grpc-status = %q, want %q", addr, rpcPath, got, want)
	}
}

// TestGRPCWebServerStreamingAutoCORS verifies server-streaming gRPC-Web calls
// with auto-CORS headers enabled.
func TestGRPCWebServerStreamingAutoCORS(t *testing.T) {
	s := setupGRPCWebStreamCORSEnv(t)
	addr := s.ListenerAddr()
	const rpcPath = "/grpc.testing.TestService/StreamingOutputCall"
	frames := [][]byte{
		encodeStreamingOutputCallRequestWire([]int{3, 4, 5, 6}),
	}

	headers, status, respFrames, trailer, err := sendGRPCWebFrames(t.Context(), addr, rpcPath, streamCORSOrigin, frames)
	if err != nil {
		t.Fatalf("sendGRPCWebFrames(%q, %q) failed: %v", addr, rpcPath, err)
	}
	if status != http.StatusOK {
		t.Fatalf("sendGRPCWebFrames(%q, %q) status = %d, want %d", addr, rpcPath, status, http.StatusOK)
	}
	if got := headers.Get("Access-Control-Allow-Origin"); got != streamCORSOrigin {
		t.Errorf("sendGRPCWebFrames(%q, %q) Access-Control-Allow-Origin = %q, want %q", addr, rpcPath, got, streamCORSOrigin)
	}
	if len(respFrames) != 4 {
		t.Errorf("sendGRPCWebFrames(%q, %q) message count = %d, want 4", addr, rpcPath, len(respFrames))
	}
	if got, want := trailer["grpc-status"], "0"; got != want {
		t.Errorf("sendGRPCWebFrames(%q, %q) grpc-status = %q, want %q", addr, rpcPath, got, want)
	}
}

// TestNativeGRPCBidiStreaming verifies native gRPC bidirectional streaming
// without CORS enabled.
func TestNativeGRPCBidiStreaming(t *testing.T) {
	s := setupNativeBidiNoCORSEnv(t)
	testPlans := `
plans {
  echo_stream {
    request {
      text: "Hello, bidi stream!"
    }
    count: 5
  }
}`
	wantResult := `
results {
  echo_stream {
    count: 5
  }
}
`
	port := s.Ports().ListenerPort
	result, err := client.RunGRPCEchoTest(testPlans, port)
	if err != nil {
		t.Fatalf("client.RunGRPCEchoTest(echo_stream, port=%d) failed: %v", port, err)
	}
	if !strings.Contains(result, wantResult) {
		t.Errorf("client.RunGRPCEchoTest(echo_stream, port=%d) =\n%s\nwant substring:\n%s", port, result, wantResult)
	}
}

func setupGRPCWebStreamCORSEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedNativeBidiEnv != nil {
		sharedNativeBidiEnv.TearDown(t)
		sharedNativeBidiEnv = nil
	}
	if sharedInteropCORSEnv != nil {
		return sharedInteropCORSEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWGRPCWebStream, platform.GrpcInteropSidecar)
	args := append([]string{
		"--cors_preset=basic",
		"--cors_allow_origin=" + streamCORSOrigin,
		"--cors_allow_methods=GET, POST, OPTIONS",
		"--cors_allow_headers=content-type,x-grpc-web,x-user-agent",
		"--cors_expose_headers=grpc-status,grpc-message",
	}, utils.CommonArgs()...)

	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) for GrpcInteropSidecar CORS env failed: %v", args, err)
	}
	sharedInteropCORSEnv = s
	return sharedInteropCORSEnv
}

func setupNativeBidiNoCORSEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedInteropCORSEnv != nil {
		sharedInteropCORSEnv.TearDown(t)
		sharedInteropCORSEnv = nil
	}
	if sharedNativeBidiEnv != nil {
		return sharedNativeBidiEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWGRPCWebStream, platform.GrpcEchoSidecar)
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               "test.grpc.Test.EchoStream",
				AllowUnregisteredCalls: true,
			},
		},
	})
	args := utils.CommonArgs()
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) for GrpcEchoSidecar no-CORS env failed: %v", args, err)
	}
	sharedNativeBidiEnv = s
	return sharedNativeBidiEnv
}

// encodeStreamingInputCallRequestWire encodes
// grpc.testing.StreamingInputCallRequest{payload: Payload{body: body}}.
// StreamingInputCallRequest field 1 (payload, bytes) -> Payload field 2
// (body, bytes).
func encodeStreamingInputCallRequestWire(body []byte) []byte {
	var payload []byte
	payload = protowire.AppendTag(payload, 2, protowire.BytesType)
	payload = protowire.AppendBytes(payload, body)

	var req []byte
	req = protowire.AppendTag(req, 1, protowire.BytesType)
	req = protowire.AppendBytes(req, payload)
	return req
}

// encodeStreamingOutputCallRequestWire encodes a
// grpc.testing.StreamingOutputCallRequest with response_parameters sizes.
// StreamingOutputCallRequest field 2 (response_parameters, bytes) ->
// ResponseParameters field 1 (size, varint).
func encodeStreamingOutputCallRequestWire(sizes []int) []byte {
	var out []byte
	for _, size := range sizes {
		var param []byte
		param = protowire.AppendTag(param, 1, protowire.VarintType)
		param = protowire.AppendVarint(param, uint64(size))

		out = protowire.AppendTag(out, 2, protowire.BytesType)
		out = protowire.AppendBytes(out, param)
	}
	return out
}

func sendGRPCWebFrames(ctx context.Context, addr, rpcPath, origin string, frames [][]byte) (http.Header, int, [][]byte, bsclient.GRPCWebTrailer, error) {
	var rawBuf bytes.Buffer
	for _, msgBytes := range frames {
		var header [5]byte
		binary.BigEndian.PutUint32(header[1:], uint32(len(msgBytes)))
		rawBuf.Write(header[:])
		rawBuf.Write(msgBytes)
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(rawBuf.Len()))
	base64.StdEncoding.Encode(encoded, rawBuf.Bytes())

	reqURL := fmt.Sprintf("http://%s%s", addr, rpcPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, nil, nil, fmt.Errorf("creating gRPC-Web request for %q: %v", reqURL, err)
	}
	req.Header.Set("X-User-Agent", "grpc-web-javascript/0.1")
	req.Header.Set("Content-Type", "application/grpc-web-text")
	req.Header.Set("Accept", "application/grpc-web-text")
	req.Header.Set("X-Grpc-Web", "1")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, nil, fmt.Errorf("sending gRPC-Web request to %q: %v", reqURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, resp.StatusCode, nil, nil, fmt.Errorf("reading gRPC-Web response body from %q: %v", reqURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp.Header, resp.StatusCode, nil, nil, nil
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(respBytes)))
	nRead, nWritten := 0, 0
	for nRead < len(respBytes) {
		n, err := base64.StdEncoding.Decode(decoded[nWritten:], respBytes[nRead:])
		nWritten += n
		if err == nil {
			break
		}
		var corruptErr base64.CorruptInputError
		if !errors.As(err, &corruptErr) || n == 0 || int(corruptErr) <= 0 || int(corruptErr) > len(respBytes)-nRead {
			return resp.Header, resp.StatusCode, nil, nil, fmt.Errorf("decoding gRPC-Web base64 response at offset %d: %v", nRead, err)
		}
		// application/grpc-web-text concatenates padded base64 chunks; Go's
		// base64.Decode reports CorruptInputError at the start of the next
		// chunk immediately following a '=' padding boundary.
		chunk := bytes.TrimRight(respBytes[nRead:nRead+int(corruptErr)], "\r\n")
		if len(chunk) == 0 || chunk[len(chunk)-1] != '=' {
			return resp.Header, resp.StatusCode, nil, nil, fmt.Errorf("decoding gRPC-Web base64 response at offset %d: %v", nRead+int(corruptErr), err)
		}
		nRead += int(corruptErr)
	}

	reader := bytes.NewReader(decoded[:nWritten])
	var respFrames [][]byte
	trailer := make(bsclient.GRPCWebTrailer)
	for {
		var header [5]byte
		if _, err := io.ReadFull(reader, header[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return resp.Header, resp.StatusCode, nil, nil, fmt.Errorf("reading gRPC-Web frame header: %v", err)
		}
		isTrailer := (header[0] & 0x80) != 0
		length := binary.BigEndian.Uint32(header[1:])
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return resp.Header, resp.StatusCode, nil, nil, fmt.Errorf("reading gRPC-Web frame payload (%d bytes): %v", length, err)
		}
		if isTrailer {
			for _, line := range strings.Split(string(payload), "\r\n") {
				key, val, ok := strings.Cut(strings.TrimSpace(line), ":")
				if ok {
					trailer[strings.TrimSpace(key)] = strings.TrimSpace(val)
				}
			}
			break
		}
		respFrames = append(respFrames, payload)
	}
	return resp.Header, resp.StatusCode, respFrames, trailer, nil
}
