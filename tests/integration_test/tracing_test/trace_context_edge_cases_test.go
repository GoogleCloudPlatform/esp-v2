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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestTraceContextEdgeCases(t *testing.T) {
	t.Parallel()

	tpTraceId := "0af7651916cd43dd8448eb211c80319c"
	tpSpanId := "b7ad6b7169203331"
	validTP := createTraceparent(tpTraceId, tpSpanId, true)

	// -------------------------------------------------------------------------
	// Edge 1: Anti-Spoofing Stash Protection
	// -------------------------------------------------------------------------
	t.Run("Edge1_AntiSpoofingStashProtection", func(t *testing.T) {
		spoofedStash := "00-spoofed0000000000000000000000000-1111111111111111-01"

		// 1A: Active tracing enabled - client attempts to inject fake stash header
		s1 := env.NewTestEnv(240, platform.EchoRemote)
		s1.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s1)
			s1.TearDown(t)
		}()
		confArgs1 := append([]string{
			"--tracing_incoming_context=traceparent,x-cloud-trace-context",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context",
		}, utils.CommonArgs()...)
		if err := s1.Setup(confArgs1); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url1 := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s1.Ports().ListenerPort, "/echoHeader")
		headers1, _, err := utils.DoWithHeaders(url1, util.GET, "", map[string]string{
			"traceparent":                  validTP,
			"x-espv2-original-traceparent": spoofedStash,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		if headers1.Get("Echo-X-Espv2-Original-Traceparent") != "" {
			t.Errorf("Echo-X-Espv2-Original-Traceparent must not be echoed to backend")
		}
		gotTP1 := headers1.Get("Echo-Traceparent")
		if !strings.HasPrefix(gotTP1, "00-"+tpTraceId+"-") {
			t.Errorf("Echo-Traceparent (%s) should be rooted in authentic TP trace ID (%s), not spoofed", gotTP1, tpTraceId)
		}

		// 1B: Outgoing tracing disabled - client sends ONLY spoofed stash without traceparent
		s2 := env.NewTestEnv(241, platform.EchoRemote)
		s2.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s2)
			s2.TearDown(t)
		}()
		confArgs2 := append([]string{
			"--tracing_incoming_context=",
			"--tracing_outgoing_context=",
		}, utils.CommonArgs()...)
		if err := s2.Setup(confArgs2); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url2 := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s2.Ports().ListenerPort, "/echoHeader")
		headers2, _, err := utils.DoWithHeaders(url2, util.GET, "", map[string]string{
			"x-espv2-original-traceparent": spoofedStash,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		gotTP2 := headers2.Get("Echo-Traceparent")
		if strings.Contains(gotTP2, "spoofed") {
			t.Errorf("Echo-Traceparent (%s) must not contain spoofed trace ID", gotTP2)
		}
		if headers2.Get("Echo-X-Espv2-Original-Traceparent") != "" {
			t.Errorf("Echo-X-Espv2-Original-Traceparent must be stripped by proxy")
		}
	})

	// -------------------------------------------------------------------------
	// Edge 2: Anti-Leak Stash Cleanup
	// -------------------------------------------------------------------------
	t.Run("Edge2_AntiLeakStashCleanup", func(t *testing.T) {
		s := env.NewTestEnv(242, platform.EchoRemote)
		s.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s)
			s.TearDown(t)
		}()
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent",
			"--tracing_outgoing_context=traceparent",
		}, utils.CommonArgs()...)
		if err := s.Setup(confArgs); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader")
		headers, _, err := utils.DoWithHeaders(url, util.GET, "", map[string]string{
			"traceparent": validTP,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		if headers.Get("Echo-X-Espv2-Original-Traceparent") != "" {
			t.Errorf("internal stash header x-espv2-original-traceparent leaked to backend")
		}
	})

	// -------------------------------------------------------------------------
	// Edge 3: Malformed Header Fallbacks
	// -------------------------------------------------------------------------
	t.Run("Edge3a_MalformedCloudTraceFallback", func(t *testing.T) {
		s := env.NewTestEnv(244, platform.EchoRemote)
		s.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s)
			s.TearDown(t)
		}()
		confArgs := append([]string{
			"--tracing_incoming_context=x-cloud-trace-context,traceparent",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context",
		}, utils.CommonArgs()...)
		if err := s.Setup(confArgs); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader")
		headers, _, err := utils.DoWithHeaders(url, util.GET, "", map[string]string{
			"X-Cloud-Trace-Context": "invalid-garbage-format!@#$",
			"traceparent":           validTP,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		gotTP := headers.Get("Echo-Traceparent")
		if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
			t.Errorf("Echo-Traceparent (%s) did not fallback to valid TP trace ID (%s)", gotTP, tpTraceId)
		}
		gotCT := headers.Get("Echo-X-Cloud-Trace-Context")
		if !strings.HasPrefix(gotCT, tpTraceId+"/") {
			t.Errorf("Echo-X-Cloud-Trace-Context (%s) did not match fallback TP trace ID (%s)", gotCT, tpTraceId)
		}
	})

	t.Run("Edge3b_MalformedGrpcTraceBinFallback", func(t *testing.T) {
		s := env.NewTestEnv(246, platform.EchoRemote)
		s.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s)
			s.TearDown(t)
		}()
		confArgs := append([]string{
			"--tracing_incoming_context=grpc-trace-bin,traceparent",
			"--tracing_outgoing_context=traceparent,grpc-trace-bin",
		}, utils.CommonArgs()...)
		if err := s.Setup(confArgs); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader")
		headers, _, err := utils.DoWithHeaders(url, util.GET, "", map[string]string{
			"grpc-trace-bin": "!not-valid-base64!",
			"traceparent":    validTP,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		gotTP := headers.Get("Echo-Traceparent")
		if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
			t.Errorf("Echo-Traceparent (%s) did not fallback to valid TP trace ID (%s)", gotTP, tpTraceId)
		}
		gotGTB := headers.Get("Echo-Grpc-Trace-Bin")
		gtbPrefix, _ := createGrpcTraceBinPrefix(tpTraceId)
		if !strings.HasPrefix(gotGTB, gtbPrefix) {
			t.Errorf("Echo-Grpc-Trace-Bin (%s) did not match fallback TP trace ID (%s)", gotGTB, tpTraceId)
		}
	})

	// -------------------------------------------------------------------------
	// Edge 4: Sampling Flag Preservation (Sampled vs Unsampled)
	// -------------------------------------------------------------------------
	t.Run("Edge4_SamplingFlagPreservation", func(t *testing.T) {
		s := env.NewTestEnv(248, platform.EchoRemote)
		s.SetupFakeTraceServer(1)
		defer func() {
			drainSpans(s)
			s.TearDown(t)
		}()
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent,x-cloud-trace-context,grpc-trace-bin",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context,grpc-trace-bin",
		}, utils.CommonArgs()...)
		if err := s.Setup(confArgs); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader")

		// 4A: Sampled = false (flag 00)
		unsampledTP := createTraceparent(tpTraceId, tpSpanId, false)
		headersUnsampled, _, err := utils.DoWithHeaders(url, util.GET, "", map[string]string{
			"traceparent": unsampledTP,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		gotTPUnsampled := headersUnsampled.Get("Echo-Traceparent")
		if !strings.HasSuffix(gotTPUnsampled, "-00") {
			t.Errorf("Echo-Traceparent (%s) did not preserve unsampled flag 00", gotTPUnsampled)
		}
		gotCTUnsampled := headersUnsampled.Get("Echo-X-Cloud-Trace-Context")
		if !strings.HasPrefix(gotCTUnsampled, tpTraceId+"/") {
			t.Errorf("Echo-X-Cloud-Trace-Context (%s) did not match trace ID (%s)", gotCTUnsampled, tpTraceId)
		}
		gotGTBUnsampled := headersUnsampled.Get("Echo-Grpc-Trace-Bin")
		gtbPrefix, _ := createGrpcTraceBinPrefix(tpTraceId)
		if !strings.HasPrefix(gotGTBUnsampled, gtbPrefix) {
			t.Errorf("Echo-Grpc-Trace-Bin (%s) did not match trace ID (%s)", gotGTBUnsampled, tpTraceId)
		}

		// 4B: Sampled = true (flag 01)
		sampledTP := createTraceparent(tpTraceId, tpSpanId, true)
		headersSampled, _, err := utils.DoWithHeaders(url, util.GET, "", map[string]string{
			"traceparent": sampledTP,
		})
		if err != nil {
			t.Fatalf("fail to call backend: %v", err)
		}
		gotTPSampled := headersSampled.Get("Echo-Traceparent")
		if !strings.HasSuffix(gotTPSampled, "-01") {
			t.Errorf("Echo-Traceparent (%s) did not preserve sampled flag 01", gotTPSampled)
		}
		gotCTSampled := headersSampled.Get("Echo-X-Cloud-Trace-Context")
		if !strings.HasSuffix(gotCTSampled, ";o=1") {
			t.Errorf("Echo-X-Cloud-Trace-Context (%s) did not preserve sampled flag ;o=1", gotCTSampled)
		}
		gotGTBSampled := headersSampled.Get("Echo-Grpc-Trace-Bin")
		_, _, sampledSampled, err := parseGrpcTraceBin(gotGTBSampled)
		if err != nil {
			t.Fatalf("failed to parse Echo-Grpc-Trace-Bin: %v", err)
		}
		if !sampledSampled {
			t.Errorf("Echo-Grpc-Trace-Bin did not preserve sampled options bit")
		}
	})
}
