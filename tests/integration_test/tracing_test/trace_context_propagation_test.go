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
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

type propagationTestCase struct {
	desc           string
	requestHeaders map[string]string
	assertFn       func(t *testing.T, respHeaders http.Header)
}

func runPropagationConfigGroup(t *testing.T, testId uint16, confArgs []string, testCases []propagationTestCase) {
	s := env.NewTestEnv(testId, platform.EchoRemote)
	s.SetupFakeTraceServer(1)
	defer func() {
		drainSpans(s)
		s.TearDown(t)
	}()
	if err := s.Setup(confArgs); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader")
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			respHeaders, _, err := utils.DoWithHeaders(url, util.GET, "", tc.requestHeaders)
			if err != nil {
				t.Fatalf("fail to make call to backend: %v", err)
			}
			tc.assertFn(t, respHeaders)
		})
	}
}

// TestTraceContextPropagationMatrix implements all 20 matrix permutations
// organized into 10 Proxy Configuration Groups per Section 8 & Section 9.3 of the Master Test Plan.
func TestTraceContextPropagationMatrix(t *testing.T) {
	t.Parallel()

	tpTraceId := "0af7651916cd43dd8448eb211c80319c"
	tpSpanId := "b7ad6b7169203331"
	ctTraceId := "105445aa7843bc8bf206b12000100000"
	ctSpanId := "12345"
	gtbTraceId := "33333333333333333333333333333333"
	gtbSpanId := "4444444444444444"

	incomingTP := createTraceparent(tpTraceId, tpSpanId, true)
	incomingCT := createCloudTraceContext(ctTraceId, ctSpanId, true)
	incomingGTB, err := createGrpcTraceBin(gtbTraceId, gtbSpanId, true)
	if err != nil {
		t.Fatalf("failed to create incoming GTB: %v", err)
	}

	// -------------------------------------------------------------------------
	// Group 1: Default Flags (traceparent,x-cloud-trace-context in & out)
	// Cases 1, 2, 3, 4
	// -------------------------------------------------------------------------
	t.Run("Group1_DefaultFlags", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent,x-cloud-trace-context",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 220, confArgs, []propagationTestCase{
			{
				desc: "Case 1: TP Only propagates trace ID and generates child span ID and Cloud Trace header",
				requestHeaders: map[string]string{
					"traceparent": incomingTP,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) does not match expected prefix (00-%s-)", gotTP, tpTraceId)
					}
					if gotTP == incomingTP {
						t.Errorf("Echo-Traceparent (%s) must have a new child span ID, should not equal incoming", gotTP)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, tpTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match expected prefix (%s/)", gotCT, tpTraceId)
					}
					if !strings.HasSuffix(gotCT, ";o=1") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not have sampled flag ;o=1", gotCT)
					}
				},
			},
			{
				desc: "Case 2: XCTC Only translates to traceparent and preserves trace ID",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+ctTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) does not match expected prefix (00-%s-)", gotTP, ctTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match expected prefix (%s/)", gotCT, ctTraceId)
					}
					if gotCT == incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must have a new child span ID", gotCT)
					}
				},
			},
			{
				desc: "Case 3: TP + XCTC favors traceparent by default and discards XCTC trace ID",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) does not match expected prefix (00-%s-)", gotTP, tpTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, tpTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) should be rooted in TP trace ID (%s)", gotCT, tpTraceId)
					}
					if strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must not contain discarded XCTC trace ID (%s)", gotCT, ctTraceId)
					}
				},
			},
			{
				desc:           "Case 4: No headers generates matching new root trace ID in both headers",
				requestHeaders: nil,
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-") || len(gotTP) != 55 {
						t.Fatalf("Echo-Traceparent (%s) is not a valid fresh traceparent header", gotTP)
					}
					parts := strings.Split(gotTP, "-")
					if len(parts) != 4 {
						t.Fatalf("malformed Echo-Traceparent: %s", gotTP)
					}
					newTraceId := parts[1]
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, newTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match generated root trace ID (%s)", gotCT, newTraceId)
					}
				},
			},
		})
	})

	// -------------------------------------------------------------------------
	// Group 2: Sequential Flag Order Precedence & Reversal
	// Cases 5, 6, 7, 8
	// -------------------------------------------------------------------------
	t.Run("Group2a_Case5_CloudTraceFirst", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=x-cloud-trace-context,traceparent",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 222, confArgs, []propagationTestCase{
			{
				desc: "Case 5: XCTC listed first in incoming context takes precedence over TP",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+ctTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) must be rooted in XCTC trace ID (%s)", gotTP, ctTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must be rooted in XCTC trace ID (%s)", gotCT, ctTraceId)
					}
				},
			},
		})
	})

	t.Run("Group2b_Case6_GrpcTraceBinFirst", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=grpc-trace-bin,traceparent",
			"--tracing_outgoing_context=traceparent,grpc-trace-bin",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 224, confArgs, []propagationTestCase{
			{
				desc: "Case 6: grpc-trace-bin listed first in incoming context takes precedence over TP",
				requestHeaders: map[string]string{
					"traceparent":    incomingTP,
					"grpc-trace-bin": incomingGTB,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+gtbTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) must be rooted in GTB trace ID (%s)", gotTP, gtbTraceId)
					}
					gotGTB := h.Get("Echo-Grpc-Trace-Bin")
					gtbPrefix, _ := createGrpcTraceBinPrefix(gtbTraceId)
					if !strings.HasPrefix(gotGTB, gtbPrefix) {
						t.Errorf("Echo-Grpc-Trace-Bin (%s) does not have expected prefix (%s)", gotGTB, gtbPrefix)
					}
				},
			},
		})
	})

	t.Run("Group2c_Case7_CloudTraceVsGrpcTraceBin_CloudTraceFirst", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=x-cloud-trace-context,grpc-trace-bin",
			"--tracing_outgoing_context=x-cloud-trace-context",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 226, confArgs, []propagationTestCase{
			{
				desc: "Case 7: XCTC listed first takes precedence over GTB; unconfigured GTB is preserved",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
					"grpc-trace-bin":        incomingGTB,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must be rooted in XCTC trace ID (%s)", gotCT, ctTraceId)
					}
					gotGTB := h.Get("Echo-Grpc-Trace-Bin")
					if gotGTB != incomingGTB {
						t.Errorf("Echo-Grpc-Trace-Bin (%s) was not preserved untouched matching incoming (%s)", gotGTB, incomingGTB)
					}
				},
			},
		})
	})

	t.Run("Group2d_Case8_GrpcTraceBinVsCloudTrace_GrpcTraceBinFirst", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=grpc-trace-bin,x-cloud-trace-context",
			"--tracing_outgoing_context=grpc-trace-bin",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 228, confArgs, []propagationTestCase{
			{
				desc: "Case 8: GTB listed first takes precedence over XCTC; unconfigured XCTC is preserved",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
					"grpc-trace-bin":        incomingGTB,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotGTB := h.Get("Echo-Grpc-Trace-Bin")
					gtbPrefix, _ := createGrpcTraceBinPrefix(gtbTraceId)
					if !strings.HasPrefix(gotGTB, gtbPrefix) {
						t.Errorf("Echo-Grpc-Trace-Bin (%s) must be rooted in GTB trace ID (%s)", gotGTB, gtbTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) was not preserved untouched matching incoming (%s)", gotCT, incomingCT)
					}
				},
			},
		})
	})

	// -------------------------------------------------------------------------
	// Group 3: Single-Format Configurations
	// Cases 9, 10, 11 (W3C Only) and Cases 12, 13, 14 (Cloud Trace Only)
	// -------------------------------------------------------------------------
	t.Run("Group3a_W3COnly", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent",
			"--tracing_outgoing_context=traceparent",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 230, confArgs, []propagationTestCase{
			{
				desc: "Case 9: TP Only with W3C-only configuration propagates trace ID",
				requestHeaders: map[string]string{
					"traceparent": incomingTP,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) does not match expected prefix", gotTP)
					}
					if h.Get("Echo-X-Cloud-Trace-Context") != "" {
						t.Errorf("Echo-X-Cloud-Trace-Context should not exist in W3C-only mode")
					}
				},
			},
			{
				desc: "Case 10: XCTC Only with W3C-only configuration generates fresh TP root and preserves XCTC",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-") || strings.Contains(gotTP, ctTraceId) {
						t.Errorf("Echo-Traceparent (%s) should be a fresh root, not containing unconfigured XCTC (%s)", gotTP, ctTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) should be preserved untouched matching incoming (%s)", gotCT, incomingCT)
					}
				},
			},
			{
				desc: "Case 11: TP + XCTC with W3C-only configuration propagates TP and preserves XCTC",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) does not match expected prefix (00-%s-)", gotTP, tpTraceId)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) should be preserved untouched matching incoming (%s)", gotCT, incomingCT)
					}
				},
			},
		})
	})

	t.Run("Group3b_CloudTraceOnly", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=x-cloud-trace-context",
			"--tracing_outgoing_context=x-cloud-trace-context",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 232, confArgs, []propagationTestCase{
			{
				desc: "Case 12: TP Only with XCTC-only configuration generates new XCTC root and restores original TP via stash",
				requestHeaders: map[string]string{
					"traceparent": incomingTP,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT == "" || strings.Contains(gotCT, tpTraceId) {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must be a fresh root not matching TP trace ID", gotCT)
					}
					gotTP := h.Get("Echo-Traceparent")
					if gotTP != incomingTP {
						t.Errorf("Echo-Traceparent (%s) was not restored to original incoming TP (%s)", gotTP, incomingTP)
					}
				},
			},
			{
				desc: "Case 13: XCTC Only with XCTC-only configuration propagates trace ID",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match expected prefix (%s/)", gotCT, ctTraceId)
					}
					if h.Get("Echo-Traceparent") != "" {
						t.Errorf("Echo-Traceparent should not exist in XCTC-only mode when not provided in request")
					}
				},
			},
			{
				desc: "Case 14: TP + XCTC with XCTC-only configuration propagates XCTC and restores original TP",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, ctTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must be rooted in XCTC trace ID (%s)", gotCT, ctTraceId)
					}
					gotTP := h.Get("Echo-Traceparent")
					if gotTP != incomingTP {
						t.Errorf("Echo-Traceparent (%s) was not restored to original incoming TP (%s)", gotTP, incomingTP)
					}
				},
			},
		})
	})

	// -------------------------------------------------------------------------
	// Group 4: Disabled & Transparent Proxy Passthrough
	// Cases 15, 16, 17 (Fully Disabled) and Cases 18, 19 (Outgoing Disabled Only)
	// -------------------------------------------------------------------------
	t.Run("Group4a_FullyDisabled", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=",
			"--tracing_outgoing_context=",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 234, confArgs, []propagationTestCase{
			{
				desc: "Case 15: TP Only with tracing fully disabled restores original TP untouched",
				requestHeaders: map[string]string{
					"traceparent": incomingTP,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if gotTP != incomingTP {
						t.Errorf("Echo-Traceparent (%s) does not match original incoming (%s)", gotTP, incomingTP)
					}
				},
			},
			{
				desc: "Case 16: XCTC Only with tracing fully disabled passes through XCTC untouched",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match original incoming (%s)", gotCT, incomingCT)
					}
				},
			},
			{
				desc: "Case 17: TP + XCTC with tracing fully disabled passes both through untouched",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if gotTP != incomingTP {
						t.Errorf("Echo-Traceparent (%s) does not match original incoming (%s)", gotTP, incomingTP)
					}
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match original incoming (%s)", gotCT, incomingCT)
					}
				},
			},
		})
	})

	t.Run("Group4b_OutgoingDisabledOnly", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent,x-cloud-trace-context",
			"--tracing_outgoing_context=",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 236, confArgs, []propagationTestCase{
			{
				desc: "Case 18: TP Only with outgoing tracing disabled strips child span and restores original TP",
				requestHeaders: map[string]string{
					"traceparent": incomingTP,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if gotTP != incomingTP {
						t.Errorf("Echo-Traceparent (%s) was not restored to original incoming (%s)", gotTP, incomingTP)
					}
				},
			},
			{
				desc: "Case 19: XCTC Only with outgoing tracing disabled passes through original XCTC without generating TP",
				requestHeaders: map[string]string{
					"X-Cloud-Trace-Context": incomingCT,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if gotCT != incomingCT {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) does not match original incoming (%s)", gotCT, incomingCT)
					}
					if h.Get("Echo-Traceparent") != "" {
						t.Errorf("Echo-Traceparent should not be emitted when outgoing tracing is disabled")
					}
				},
			},
		})
	})

	// -------------------------------------------------------------------------
	// Group 5: All Formats Simultaneous
	// Case 20 (All Three Formats)
	// -------------------------------------------------------------------------
	t.Run("Group5_AllFormatsSimultaneous", func(t *testing.T) {
		confArgs := append([]string{
			"--tracing_incoming_context=traceparent,x-cloud-trace-context,grpc-trace-bin",
			"--tracing_outgoing_context=traceparent,x-cloud-trace-context,grpc-trace-bin",
		}, utils.CommonArgs()...)

		runPropagationConfigGroup(t, 238, confArgs, []propagationTestCase{
			{
				desc: "Case 20: All three formats simultaneously translated, sharing same trace ID and child span ID",
				requestHeaders: map[string]string{
					"traceparent":           incomingTP,
					"X-Cloud-Trace-Context": incomingCT,
					"grpc-trace-bin":        incomingGTB,
				},
				assertFn: func(t *testing.T, h http.Header) {
					gotTP := h.Get("Echo-Traceparent")
					if !strings.HasPrefix(gotTP, "00-"+tpTraceId+"-") {
						t.Errorf("Echo-Traceparent (%s) must be rooted in TP trace ID (%s)", gotTP, tpTraceId)
					}
					tpParts := strings.Split(gotTP, "-")
					if len(tpParts) != 4 {
						t.Fatalf("malformed Echo-Traceparent: %s", gotTP)
					}
					childSpanId := tpParts[2]
					if childSpanId == tpSpanId {
						t.Errorf("Echo-Traceparent (%s) child span ID must not equal incoming span ID (%s)", childSpanId, tpSpanId)
					}

					gotCT := h.Get("Echo-X-Cloud-Trace-Context")
					if !strings.HasPrefix(gotCT, tpTraceId+"/") {
						t.Errorf("Echo-X-Cloud-Trace-Context (%s) must be rooted in TP trace ID (%s)", gotCT, tpTraceId)
					}

					gotGTB := h.Get("Echo-Grpc-Trace-Bin")
					if gotGTB == "" {
						t.Fatalf("Echo-Grpc-Trace-Bin header missing")
					}
					parsedTraceId, parsedSpanId, parsedSampled, err := parseGrpcTraceBin(gotGTB)
					if err != nil {
						t.Fatalf("failed to parse Echo-Grpc-Trace-Bin (%s): %v", gotGTB, err)
					}
					if parsedTraceId != tpTraceId {
						t.Errorf("Echo-Grpc-Trace-Bin trace ID (%s) does not match TP trace ID (%s)", parsedTraceId, tpTraceId)
					}
					// Both legacy headers (XCTC and GTB) are generated by ESPv2's TraceContextFilter
					// from active_span.getSpanId(), so they must share the exact same span ID.
					ctParts := strings.Split(gotCT, "/")
					if len(ctParts) >= 2 {
						ctSpanAndFlags := strings.Split(ctParts[1], ";")
						ctSpanDec, parseErr := strconv.ParseUint(ctSpanAndFlags[0], 10, 64)
						if parseErr != nil {
							t.Fatalf("failed to parse span ID from Echo-X-Cloud-Trace-Context (%s): %v", gotCT, parseErr)
						}
						expectedLegacySpanHex := fmt.Sprintf("%016x", ctSpanDec)
						if parsedSpanId != expectedLegacySpanHex {
							t.Errorf("Echo-Grpc-Trace-Bin span ID (%s) does not match Echo-X-Cloud-Trace-Context span ID (%s)", parsedSpanId, expectedLegacySpanHex)
						}
					}
					if !parsedSampled {
						t.Errorf("Echo-Grpc-Trace-Bin options bit should reflect sampled flag true")
					}
				},
			},
		})
	})
}
