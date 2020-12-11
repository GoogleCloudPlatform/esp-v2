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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/proto"

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

	defaultOpts := options.DefaultCommonOptions()

	testData := []struct {
		desc                       string
		tracingProjectId           string
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
			desc:                       "Success with custom stackdriver address",
			tracingProjectId:           fakeOptsProjectId,
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
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeOptsProjectId,
				StackdriverAddress:         fakeStackdriverAddress,
			},
		},
		{
			desc:                       "Success with custom max number of attributes/annotations/message_events/links",
			tracingProjectId:           fakeOptsProjectId,
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
		opts.TracingIncomingContext = tc.tracingIncomingContext
		opts.TracingOutgoingContext = tc.tracingOutgoingContext
		opts.TracingStackdriverAddress = tc.tracingStackdriverAddress
		opts.TracingMaxNumAttributes = tc.tracingMaxNumAttributes
		opts.TracingMaxNumAnnotations = tc.tracingMaxNumAnnotations
		opts.TracingMaxNumMessageEvents = tc.tracingMaxNumMessageEvents
		opts.TracingMaxNumLinks = tc.tracingMaxNumLinks

		got, err := createOpenCensusConfig(opts)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		}

		if tc.wantResult != nil {
			if got == nil {
				t.Errorf("Test (%s): failed, expected result should not be nil", tc.desc)
			}

			if !proto.Equal(got, tc.wantResult) {
				t.Errorf("Test (%s): failed, got : %v, want: %v", tc.desc, got, tc.wantResult)
			}
		}
	}
}

// Ensures that the project-id is automatically populated in the tracing config on GCP deployments
func TestGcpOpenCensusConfig(t *testing.T) {

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
				},
				StackdriverExporterEnabled: true,
				StackdriverProjectId:       fakeMetadataProjectId,
				IncomingTraceContext: []tracepb.OpenCensusConfig_TraceContext{
					tracepb.OpenCensusConfig_TRACE_CONTEXT,
					tracepb.OpenCensusConfig_CLOUD_TRACE_CONTEXT,
				},
				OutgoingTraceContext: []tracepb.OpenCensusConfig_TraceContext{
					tracepb.OpenCensusConfig_TRACE_CONTEXT,
					tracepb.OpenCensusConfig_CLOUD_TRACE_CONTEXT,
				},
			},
		},
	}

	for _, tc := range testData {

		runTest(t, true, func() {

			opts := options.DefaultCommonOptions()
			got, err := createOpenCensusConfig(opts)
			if err != nil {
				t.Fatalf("Test (%s): failed, got err: %v, want no err", tc.desc, err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Errorf("Test (%s): failed, expected result should not be nil", tc.desc)
				}

				if !proto.Equal(got, tc.wantResult) {
					t.Errorf("Test (%s): failed, got : %v, want: %v", tc.desc, got, tc.wantResult)
				}
			}

		})

	}
}

// Tests the sample rate is correctly populated in the HCM tracing config.
func TestHcmTracingSampleRate(t *testing.T) {

	testData := []struct {
		desc              string
		tracingSampleRate float64
		wantResult        *hcmpb.HttpConnectionManager_Tracing
		wantError         string
	}{
		{
			desc:              "Default sampling rate works",
			tracingSampleRate: options.DefaultCommonOptions().TracingSamplingRate,
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
			desc:              "Custom sampling rate works",
			tracingSampleRate: 0.275,
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
			desc:              "Sample rate of 1 works",
			tracingSampleRate: 1,
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
			desc:              "Sample rate of 0 works",
			tracingSampleRate: 0,
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
			desc:              "Sample rate rounded at 6 decimal points",
			tracingSampleRate: 0.123456789,
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
			desc:              "Invalid sampling rate has error",
			tracingSampleRate: 1.3,
			wantError:         "invalid trace sampling rate",
		},
	}

	for _, tc := range testData {

		runTest(t, true, func() {

			opts := options.DefaultCommonOptions()
			opts.TracingSamplingRate = tc.tracingSampleRate

			got, err := CreateTracing(opts)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Errorf("Test (%s): failed, expected result should not be nil", tc.desc)
				}

				got.Provider.ConfigType = nil
				if !proto.Equal(got, tc.wantResult) {
					t.Errorf("Test (%s): failed, got : %v, want: %v", tc.desc, got, tc.wantResult)
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
