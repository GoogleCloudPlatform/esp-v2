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

package bootstrap

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
)

const (
	fakeOptsProjectId      = "fake-opts-project-id"
	fakeMetadataProjectId  = "fake-metadata-project-id"
	fakeStackdriverAddress = "dns:non-existent-address:2840"
)

// Tests the various combination of tracing flags on a non-GCP deployment
func TestNonGCPTracingConfig(t *testing.T) {

	defaultOpts := options.DefaultCommonOptions()

	testData := []struct {
		desc                       string
		tracingProjectId           string
		tracingSampleRate          float64
		tracingIncomingContext     string
		tracingOutgoingContext     string
		tracingStackdriverAddress  string
		tracingMaxNumAttributes    int64
		tracingMaxNumAnnotations   int64
		tracingMaxNumMessageEvents int64
		tracingMaxNumLinks         int64
		wantError                  string
		wantResult                 *tracepb.OpenCensusConfig
	}{
		{
			desc:                       "Success with default tracing",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          defaultOpts.TracingSamplingRate,
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: defaultOpts.TracingSamplingRate,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
			},
		},
		{
			desc:                   "Failed with invalid tracing_incoming_context",
			tracingProjectId:       fakeOptsProjectId,
			tracingIncomingContext: "aaa",
			wantError:              "Invalid trace context: aaa",
		},
		{
			desc:                   "Failed with invalid tracing_outgoing_context",
			tracingProjectId:       fakeOptsProjectId,
			tracingOutgoingContext: "bbb",
			wantError:              "Invalid trace context: bbb",
		},
		{
			desc:                       "Success with some tracing contexts",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          defaultOpts.TracingSamplingRate,
			tracingIncomingContext:     "traceparent,grpc-trace-bin",
			tracingOutgoingContext:     "x-cloud-trace-context",
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: defaultOpts.TracingSamplingRate,
						},
					},
				},
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
			desc:              "Failed with invalid sampling rate",
			tracingProjectId:  fakeOptsProjectId,
			tracingSampleRate: 2.1,
			wantError:         "Invalid trace sampling rate: 2.1",
		},
		{
			desc:                       "Success with sample rate 0.0",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          0.0,
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ConstantSampler{
						ConstantSampler: &opencensuspb.ConstantSampler{
							Decision: opencensuspb.ConstantSampler_ALWAYS_PARENT,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
			},
		},
		{
			desc:                       "Success with sample rate 1.0",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          1.0,
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ConstantSampler{
						ConstantSampler: &opencensuspb.ConstantSampler{
							Decision: opencensuspb.ConstantSampler_ALWAYS_ON,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
			},
		},
		{
			desc:                       "Success with sample rate 0.27",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          0.27,
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: 0.27,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
			},
		},
		{
			desc:                       "Success with custom stackdriver address",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          defaultOpts.TracingSamplingRate,
			tracingStackdriverAddress:  fakeStackdriverAddress,
			tracingMaxNumAttributes:    defaultOpts.TracingMaxNumAttributes,
			tracingMaxNumAnnotations:   defaultOpts.TracingMaxNumAnnotations,
			tracingMaxNumMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
			tracingMaxNumLinks:         defaultOpts.TracingMaxNumLinks,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: defaultOpts.TracingSamplingRate,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				StackdriverAddress:         fakeStackdriverAddress,
			},
		},
		{
			desc:                       "Success with custom max number of attributes/annotations/message_events/links",
			tracingProjectId:           fakeOptsProjectId,
			tracingSampleRate:          defaultOpts.TracingSamplingRate,
			tracingStackdriverAddress:  fakeStackdriverAddress,
			tracingMaxNumAttributes:    1,
			tracingMaxNumAnnotations:   2,
			tracingMaxNumMessageEvents: 3,
			tracingMaxNumLinks:         4,
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    1,
					MaxNumberOfAnnotations:   2,
					MaxNumberOfMessageEvents: 3,
					MaxNumberOfLinks:         4,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: defaultOpts.TracingSamplingRate,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				StackdriverAddress:         fakeStackdriverAddress,
			},
		},
	}

	for _, tc := range testData {

		opts := options.DefaultCommonOptions()
		opts.NonGCP = true
		opts.TracingProjectId = tc.tracingProjectId
		opts.TracingSamplingRate = tc.tracingSampleRate
		opts.TracingIncomingContext = tc.tracingIncomingContext
		opts.TracingOutgoingContext = tc.tracingOutgoingContext
		opts.TracingStackdriverAddress = tc.tracingStackdriverAddress
		opts.TracingMaxNumAttributes = tc.tracingMaxNumAttributes
		opts.TracingMaxNumAnnotations = tc.tracingMaxNumAnnotations
		opts.TracingMaxNumMessageEvents = tc.tracingMaxNumMessageEvents
		opts.TracingMaxNumLinks = tc.tracingMaxNumLinks

		got, err := CreateTracing(opts)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		}

		if tc.wantResult != nil {
			if got == nil {
				t.Errorf("Test (%s): failed, expected result should not be nil", tc.desc)
			}
			if got.Http.Name != "envoy.tracers.opencensus" {
				t.Errorf("Test (%s): failed, expected config name is wrong", tc.desc)
			}

			gotCfg := &tracepb.OpenCensusConfig{}
			if err := ptypes.UnmarshalAny(got.Http.GetTypedConfig(), gotCfg); err != nil {
				t.Errorf("Test (%s): failed, failed to unmarshall any", tc.desc)
			}
			if !proto.Equal(gotCfg, tc.wantResult) {
				t.Errorf("Test (%s): failed, got : %v, want: %v", tc.desc, gotCfg, tc.wantResult)
			}
		}
	}
}

// Ensures that the project-id is automatically populated in the tracing config on GCP deployments
func TestGCPTracingConfig(t *testing.T) {

	defaultOpts := options.DefaultCommonOptions()

	testData := []struct {
		desc       string
		wantResult *tracepb.OpenCensusConfig
	}{
		{
			desc: "Success with default tracing, project id from metadata",
			wantResult: &tracepb.OpenCensusConfig{
				TraceConfig: &opencensuspb.TraceConfig{
					MaxNumberOfAttributes:    defaultOpts.TracingMaxNumAttributes,
					MaxNumberOfAnnotations:   defaultOpts.TracingMaxNumAnnotations,
					MaxNumberOfMessageEvents: defaultOpts.TracingMaxNumMessageEvents,
					MaxNumberOfLinks:         defaultOpts.TracingMaxNumLinks,
					Sampler: &opencensuspb.TraceConfig_ProbabilitySampler{
						ProbabilitySampler: &opencensuspb.ProbabilitySampler{
							SamplingProbability: defaultOpts.TracingSamplingRate,
						},
					},
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeMetadataProjectId,
			},
		},
	}

	for _, tc := range testData {

		runTest(t, true, func() {

			opts := options.DefaultCommonOptions()

			got, err := CreateTracing(opts)

			if err != nil {
				t.Fatalf("Test (%s): failed, got err: %v, want no err", tc.desc, err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Errorf("Test (%s): failed, expected result should not be nil", tc.desc)
				}
				if got.Http.Name != "envoy.tracers.opencensus" {
					t.Errorf("Test (%s): failed, expected config name is wrong", tc.desc)
				}

				gotCfg := &tracepb.OpenCensusConfig{}
				if err := ptypes.UnmarshalAny(got.Http.GetTypedConfig(), gotCfg); err != nil {
					t.Errorf("Test (%s): failed, failed to unmarshall any", tc.desc)
				}
				if !proto.Equal(gotCfg, tc.wantResult) {
					t.Errorf("Test (%s): failed, got : %v, want: %v", tc.desc, gotCfg, tc.wantResult)
				}
			}

		})

	}
}

// Tests the various cases for automatically determining the project-id in any environment
func TestDetermineProjectId(t *testing.T) {
	testData := []struct {
		desc             string
		nonGcp           bool
		tracingProjectId string
		runServer        bool
		wantError        string
		wantResult       string
	}{
		{
			desc:             "tracing_project_id not specified, but successfully discovered",
			nonGcp:           false,
			tracingProjectId: "",
			runServer:        true,
			wantResult:       fakeMetadataProjectId,
		},
		{
			desc:             "tracing_project_id not specified, and non GCP runtime",
			nonGcp:           true,
			tracingProjectId: "",
			runServer:        false,
			wantError:        "tracing_project_id was not specified and can not be fetched from GCP Metadata server on non-GCP runtime",
		},
		{
			desc:             "tracing_project_id not specified, and error fetching from metadata server",
			nonGcp:           false,
			tracingProjectId: "",
			runServer:        false,
			wantError:        " ", // Allow any error message, depends on underlying http client error
		},
		{
			desc:             "tracing_project_id specified, successfully used",
			nonGcp:           false,
			tracingProjectId: fakeOptsProjectId,
			wantResult:       fakeOptsProjectId,
		},
	}

	for _, tc := range testData {

		runTest(t, tc.runServer, func() {

			opts := options.DefaultCommonOptions()
			opts.NonGCP = tc.nonGcp
			opts.TracingProjectId = tc.tracingProjectId

			got, err := getTracingProjectId(opts)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, got err: %v, want err: %v", tc.desc, err, tc.wantError)
			}

			if tc.wantError == "" && err != nil {
				t.Errorf("Test (%s): failed, got err: %v, want no err", tc.desc, err)
			}

			if !reflect.DeepEqual(got, tc.wantResult) {
				t.Errorf("Test (%s): failed, got: %v, want: %v", tc.desc, got, tc.wantResult)
			}

		})

	}
}

func runTest(_ *testing.T, shouldRunServer bool, f func()) {

	if shouldRunServer {
		// Run a mock server and point injected client to mock server
		mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
			util.ProjectIDSuffix: fakeMetadataProjectId,
		})
		defer mockMetadataServer.Close()
		metadata.SetMockMetadataFetcher(mockMetadataServer.URL, time.Now())
	} else {
		// Point injected client to non-existent url
		metadata.SetMockMetadataFetcher("non-existent-url-39874983", time.Now())
	}

	f()
}
