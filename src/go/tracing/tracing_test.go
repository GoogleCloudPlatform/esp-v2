// Copyright 2019 Google LLC
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

package tracing

import (
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

const (
	fakeOptsProjectId      = "fake-opts-project-id"
	fakeMetadataProjectId  = "fake-metadata-project-id"
	fakeStackdriverAddress = "dns:non-existent-address:2840"
)

// Tests the various combination of tracing flags on a non-GCP deployment
func TestNonGcpOpenCensusConfig(t *testing.T) {
	testData := []struct {
		desc       string
		opts       *options.TracingOptions
		wantError  string
		wantResult *tracepb.OpenCensusConfig
	}{
		{
			desc: "Success with default tracing",
			opts: &options.TracingOptions{
				ProjectId: fakeOptsProjectId,
			},
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig:                &opencensuspb.TraceConfig{},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
			},
		},
		{
			desc: "Failed with invalid tracing_incoming_context",
			opts: &options.TracingOptions{
				ProjectId:       fakeOptsProjectId,
				IncomingContext: "aaa",
			},
			wantError: "Invalid trace context: aaa",
		},
		{
			desc: "Failed with invalid tracing_outgoing_context",
			opts: &options.TracingOptions{
				ProjectId:       fakeOptsProjectId,
				OutgoingContext: "bbb",
			},
			wantError: "Invalid trace context: bbb",
		},
		{
			desc: "Success with some tracing contexts",
			opts: &options.TracingOptions{
				ProjectId:       fakeOptsProjectId,
				IncomingContext: "traceparent,grpc-trace-bin",
				OutgoingContext: "x-cloud-trace-context",
			},
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig:                &opencensuspb.TraceConfig{},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				IncomingTraceContext: []tracepb.OpenCensusConfig_TraceContext{
					tracepb.OpenCensusConfig_TRACE_CONTEXT,
					tracepb.OpenCensusConfig_GRPC_TRACE_BIN,
				},
				OutgoingTraceContext: []tracepb.OpenCensusConfig_TraceContext{
					tracepb.OpenCensusConfig_CLOUD_TRACE_CONTEXT,
				},
			},
		},
		{
			desc: "Success with custom stackdriver address",
			opts: &options.TracingOptions{
				ProjectId:          fakeOptsProjectId,
				StackdriverAddress: fakeStackdriverAddress,
			},
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig:                &opencensuspb.TraceConfig{},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				StackdriverAddress:         fakeStackdriverAddress,
			},
		},
		{
			desc: "Success with custom max number of attributes/annotations/message_events/links",
			opts: &options.TracingOptions{
				ProjectId:           fakeOptsProjectId,
				StackdriverAddress:  fakeStackdriverAddress,
				MaxNumAttributes:    1,
				MaxNumAnnotations:   2,
				MaxNumMessageEvents: 3,
				MaxNumLinks:         4,
			},
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    1,
					MaxNumberOfAnnotations:   2,
					MaxNumberOfMessageEvents: 3,
					MaxNumberOfLinks:         4,
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				StackdriverAddress:         fakeStackdriverAddress,
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := createOpenCensusConfig(*tc.opts)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("failed, expected err: %v, got: %v", tc.wantError, err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Errorf("failed, expected result should not be nil")
				}

				if diff := cmp.Diff(tc.wantResult, got, protocmp.Transform()); diff != "" {
					t.Errorf("createOpenCensusConfig(%v) diff (-want +got):\n%s", tc.opts, diff)
				}
			}
		})
	}
}

// Ensures that the project-id is automatically populated in the tracing config on GCP deployments
func TestShouldFetchTracingProjectID(t *testing.T) {
	testData := []struct {
		desc string
		opts options.CommonOptions
		want bool
	}{
		{
			desc: "No fetch when project ID is specified",
			opts: options.CommonOptions{
				TracingOptions: &options.TracingOptions{
					ProjectId: fakeOptsProjectId,
				},
			},
			want: false,
		},
		{
			desc: "No fetch when non-GCP",
			opts: options.CommonOptions{
				NonGCP:         true,
				TracingOptions: &options.TracingOptions{},
			},
			want: false,
		},
		{
			desc: "No fetch when tracing is disabled",
			opts: options.CommonOptions{
				TracingOptions: &options.TracingOptions{
					DisableTracing: true,
				},
			},
			want: false,
		},
		{
			desc: "Fetch by default",
			opts: options.CommonOptions{
				TracingOptions: &options.TracingOptions{},
			},
			want: true,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			got := ShouldFetchTracingProjectID(tc.opts)

			if got != tc.want {
				t.Fatalf("ShouldFetchTracingProjectID() got %v, want %v", got, tc.want)
			}
		})
	}
}

// Tests the sample rate is correctly populated in the HCM tracing config.
func TestHcmTracingSampleRate(t *testing.T) {

	testData := []struct {
		desc       string
		opts       options.TracingOptions
		wantResult *hcmpb.HttpConnectionManager_Tracing
		wantError  string
	}{
		{
			desc: "Default sampling rate works",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: options.DefaultCommonOptions().TracingOptions.SamplingRate,
			},
			wantResult: &hcmpb.HttpConnectionManager_Tracing{
				ClientSampling: &typepb.Percent{
					Value: 0,
				},
				RandomSampling: &typepb.Percent{
					Value: 0.1,
				},
				OverallSampling: &typepb.Percent{
					Value: 0.1,
				},
				Provider: &tracepb.Tracing_Http{
					Name: "envoy.tracers.opencensus",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Custom sampling rate works",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: 0.275,
			},
			wantResult: &hcmpb.HttpConnectionManager_Tracing{
				ClientSampling: &typepb.Percent{
					Value: 0,
				},
				RandomSampling: &typepb.Percent{
					Value: 27.5,
				},
				OverallSampling: &typepb.Percent{
					Value: 27.5,
				},
				Provider: &tracepb.Tracing_Http{
					Name: "envoy.tracers.opencensus",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Sample rate of 1 works",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: 1,
			},
			wantResult: &hcmpb.HttpConnectionManager_Tracing{
				ClientSampling: &typepb.Percent{
					Value: 0,
				},
				RandomSampling: &typepb.Percent{
					Value: 100,
				},
				OverallSampling: &typepb.Percent{
					Value: 100,
				},
				Provider: &tracepb.Tracing_Http{
					Name: "envoy.tracers.opencensus",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Sample rate of 0 works",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: 0,
			},
			wantResult: &hcmpb.HttpConnectionManager_Tracing{
				ClientSampling: &typepb.Percent{
					Value: 0,
				},
				RandomSampling: &typepb.Percent{
					Value: 0,
				},
				OverallSampling: &typepb.Percent{
					Value: 0,
				},
				Provider: &tracepb.Tracing_Http{
					Name: "envoy.tracers.opencensus",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Sample rate rounded at 6 decimal points",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: 0.123456789,
			},
			wantResult: &hcmpb.HttpConnectionManager_Tracing{
				ClientSampling: &typepb.Percent{
					Value: 0,
				},
				RandomSampling: &typepb.Percent{
					Value: 12.3457,
				},
				OverallSampling: &typepb.Percent{
					Value: 12.3457,
				},
				Provider: &tracepb.Tracing_Http{
					Name: "envoy.tracers.opencensus",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Invalid sampling rate has error",
			opts: options.TracingOptions{
				ProjectId:    "test-project",
				SamplingRate: 1.3,
			},
			wantError: "invalid trace sampling rate",
		},
		{
			desc: "Empty config when project ID is not specified",
			opts: options.TracingOptions{
				SamplingRate: options.DefaultCommonOptions().TracingOptions.SamplingRate,
			},
			wantResult: nil,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			runTest(t, true, func() {
				got, err := CreateTracing(tc.opts)

				if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
					t.Fatalf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
				}

				if tc.wantResult != nil {
					if got == nil {
						t.Fatalf("Test (%s): failed, expected result should not be nil", tc.desc)
					}

					// Not checking inner config, tested by other tests in this file.
					got.Provider.ConfigType = nil

					if diff := cmp.Diff(tc.wantResult, got, protocmp.Transform()); diff != "" {
						t.Errorf("CreateTracing() diff (-want +got):\n%s", diff)
					}
				}
			})
		})
	}
}

func runTest(_ *testing.T, shouldRunServer bool, f func()) {

	if shouldRunServer {
		// Run a mock server and point injected client to mock server
		mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
			util.ProjectIDPath: fakeMetadataProjectId,
		})
		defer mockMetadataServer.Close()
		metadata.SetMockMetadataFetcher(mockMetadataServer.URL, time.Now())
	} else {
		// Point injected client to non-existent url
		metadata.SetMockMetadataFetcher("non-existent-url-39874983", time.Now())
	}

	f()
}
