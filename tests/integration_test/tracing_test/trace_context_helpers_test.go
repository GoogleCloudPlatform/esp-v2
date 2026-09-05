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

package tracing_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// createTraceparent creates a W3C traceparent header: 00-{traceId}-{spanId}-{flags}.
func createTraceparent(traceId, spanId string, sampled bool) string {
	flag := "00"
	if sampled {
		flag = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", traceId, spanId, flag)
}

// createTraceparentPrefix returns the prefix of a W3C traceparent header matching the trace ID.
func createTraceparentPrefix(traceId string) string {
	return "00-" + traceId + "-"
}

// createTraceparentContext creates a sampled W3C traceparent header (backward-compatibility alias).
func createTraceparentContext(traceId, spanId string) string {
	return createTraceparent(traceId, spanId, true)
}

// createTraceparentContextPrefix returns the prefix of a W3C traceparent header (backward-compatibility alias).
func createTraceparentContextPrefix(traceId string) string {
	return createTraceparentPrefix(traceId)
}

// createCloudTraceContext creates an X-Cloud-Trace-Context header: {traceId}/{spanId};o={sampled}.
func createCloudTraceContext(traceId, spanId string, sampled bool) string {
	flag := "0"
	if sampled {
		flag = "1"
	}
	return fmt.Sprintf("%s/%s;o=%s", traceId, spanId, flag)
}

// createCloudTraceContextPrefix returns the prefix of an X-Cloud-Trace-Context header matching the trace ID.
func createCloudTraceContextPrefix(traceId string) string {
	return traceId + "/"
}

// createGrpcTraceBin encodes a 128-bit trace ID and 64-bit span ID into base64 grpc-trace-bin format (29 bytes).
// Byte layout:
// - Byte 0: Version (0x00)
// - Byte 1: Trace ID field ID (0x00)
// - Bytes 2-17: 16 bytes (128-bit) Trace ID
// - Byte 18: Span ID field ID (0x01)
// - Bytes 19-26: 8 bytes (64-bit) Span ID
// - Byte 27: Trace Options field ID (0x02)
// - Byte 28: 1 byte Options (0x01 if sampled, 0x00 if unsampled)
func createGrpcTraceBin(traceIdHex, spanIdHex string, sampled bool) (string, error) {
	traceBytes, err := hex.DecodeString(traceIdHex)
	if err != nil || len(traceBytes) != 16 {
		return "", fmt.Errorf("invalid traceId: %v", err)
	}
	spanBytes, err := hex.DecodeString(spanIdHex)
	if err != nil || len(spanBytes) != 8 {
		return "", fmt.Errorf("invalid spanId: %v", err)
	}

	buf := make([]byte, 29)
	buf[0] = 0 // version
	buf[1] = 0 // field 0 (trace id)
	copy(buf[2:18], traceBytes)
	buf[18] = 1 // field 1 (span id)
	copy(buf[19:27], spanBytes)
	buf[27] = 2 // field 2 (options)
	if sampled {
		buf[28] = 1
	} else {
		buf[28] = 0
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// parseGrpcTraceBin decodes base64 grpc-trace-bin and extracts trace ID, span ID, and sampled flag.
func parseGrpcTraceBin(b64 string) (traceId, spanId string, sampled bool, err error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", "", false, err
	}
	if len(raw) < 29 || raw[0] != 0 || raw[1] != 0 || raw[18] != 1 || raw[27] != 2 {
		return "", "", false, fmt.Errorf("invalid grpc-trace-bin layout")
	}
	traceId = hex.EncodeToString(raw[2:18])
	spanId = hex.EncodeToString(raw[19:27])
	sampled = (raw[28] & 1) != 0
	return traceId, spanId, sampled, nil
}

// createGrpcTraceBinPrefix generates the first 24 base64 characters of grpc-trace-bin.
// Due to 3-byte base64 boundary alignment (18 bytes = 6 * 3 bytes -> 24 base64 chars),
// the first 24 chars depend exclusively on the trace ID.
func createGrpcTraceBinPrefix(traceIdHex string) (string, error) {
	traceBytes, err := hex.DecodeString(traceIdHex)
	if err != nil || len(traceBytes) != 16 {
		return "", fmt.Errorf("invalid traceId: %v", err)
	}
	buf := make([]byte, 18)
	buf[0] = 0 // version
	buf[1] = 0 // field 0 (trace id)
	copy(buf[2:18], traceBytes)
	return base64.StdEncoding.EncodeToString(buf), nil
}

func TestTraceContextHelpersCodec(t *testing.T) {
	traceId := "0af7651916cd43dd8448eb211c80319c"
	spanId := "b7ad6b7169203331"

	// 1. Test W3C traceparent formatting
	tpSampled := createTraceparent(traceId, spanId, true)
	if tpSampled != "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01" {
		t.Errorf("unexpected createTraceparent sampled: %s", tpSampled)
	}
	tpUnsampled := createTraceparent(traceId, spanId, false)
	if tpUnsampled != "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00" {
		t.Errorf("unexpected createTraceparent unsampled: %s", tpUnsampled)
	}
	tpPrefix := createTraceparentPrefix(traceId)
	if tpPrefix != "00-0af7651916cd43dd8448eb211c80319c-" {
		t.Errorf("unexpected createTraceparentPrefix: %s", tpPrefix)
	}
	if !strings.HasPrefix(tpSampled, tpPrefix) {
		t.Errorf("tpSampled does not start with tpPrefix")
	}

	// 2. Test Cloud Trace formatting
	ctSampled := createCloudTraceContext(traceId, spanId, true)
	if ctSampled != "0af7651916cd43dd8448eb211c80319c/b7ad6b7169203331;o=1" {
		t.Errorf("unexpected createCloudTraceContext sampled: %s", ctSampled)
	}
	ctUnsampled := createCloudTraceContext(traceId, spanId, false)
	if ctUnsampled != "0af7651916cd43dd8448eb211c80319c/b7ad6b7169203331;o=0" {
		t.Errorf("unexpected createCloudTraceContext unsampled: %s", ctUnsampled)
	}
	ctPrefix := createCloudTraceContextPrefix(traceId)
	if ctPrefix != "0af7651916cd43dd8448eb211c80319c/" {
		t.Errorf("unexpected createCloudTraceContextPrefix: %s", ctPrefix)
	}
	if !strings.HasPrefix(ctSampled, ctPrefix) {
		t.Errorf("ctSampled does not start with ctPrefix")
	}

	// 3. Test grpc-trace-bin encoder and decoder round-trip (sampled)
	gtbSampled, err := createGrpcTraceBin(traceId, spanId, true)
	if err != nil {
		t.Fatalf("createGrpcTraceBin failed: %v", err)
	}
	gotTraceId, gotSpanId, gotSampled, err := parseGrpcTraceBin(gtbSampled)
	if err != nil {
		t.Fatalf("parseGrpcTraceBin failed: %v", err)
	}
	if gotTraceId != traceId || gotSpanId != spanId || gotSampled != true {
		t.Errorf("round-trip mismatch: got (%s, %s, %v), want (%s, %s, true)", gotTraceId, gotSpanId, gotSampled, traceId, spanId)
	}

	// 4. Test grpc-trace-bin encoder and decoder round-trip (unsampled)
	gtbUnsampled, err := createGrpcTraceBin(traceId, spanId, false)
	if err != nil {
		t.Fatalf("createGrpcTraceBin unsampled failed: %v", err)
	}
	gotTraceId, gotSpanId, gotSampled, err = parseGrpcTraceBin(gtbUnsampled)
	if err != nil {
		t.Fatalf("parseGrpcTraceBin unsampled failed: %v", err)
	}
	if gotTraceId != traceId || gotSpanId != spanId || gotSampled != false {
		t.Errorf("round-trip mismatch: got (%s, %s, %v), want (%s, %s, false)", gotTraceId, gotSpanId, gotSampled, traceId, spanId)
	}

	// 5. Test mathematical alignment prefix property
	gtbPrefix, err := createGrpcTraceBinPrefix(traceId)
	if err != nil {
		t.Fatalf("createGrpcTraceBinPrefix failed: %v", err)
	}
	if len(gtbPrefix) != 24 {
		t.Errorf("expected 24-char prefix, got len %d (%s)", len(gtbPrefix), gtbPrefix)
	}
	if !strings.HasPrefix(gtbSampled, gtbPrefix) {
		t.Errorf("gtbSampled (%s) does not have prefix (%s)", gtbSampled, gtbPrefix)
	}
	if !strings.HasPrefix(gtbUnsampled, gtbPrefix) {
		t.Errorf("gtbUnsampled (%s) does not have prefix (%s)", gtbUnsampled, gtbPrefix)
	}

	// 6. Test error cases
	if _, err := createGrpcTraceBin("invalid_hex", spanId, true); err == nil {
		t.Errorf("expected error for invalid traceId hex")
	}
	if _, err := createGrpcTraceBin(traceId, "short", true); err == nil {
		t.Errorf("expected error for short spanId")
	}
	if _, _, _, err := parseGrpcTraceBin("invalid_base64!!!"); err == nil {
		t.Errorf("expected error for invalid base64")
	}
	if _, _, _, err := parseGrpcTraceBin("AAAA"); err == nil {
		t.Errorf("expected error for truncated buffer")
	}
}
