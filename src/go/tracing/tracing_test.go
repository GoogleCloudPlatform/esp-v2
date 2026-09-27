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
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	fakeOptsProjectId     = "fake-opts-project-id"
	fakeMetadataProjectId = "fake-metadata-project-id"
)

func TestNormalizeOtlpEndpoint(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{
			input: "http://127.0.0.1:4317",
			want:  "127.0.0.1:4317",
		},
		{
			input: "https://custom-collector:4317",
			want:  "custom-collector:4317",
		},
		{
			input: "custom-collector:4317",
			want:  "custom-collector:4317",
		},
		{
			input: "  http://custom-collector:4317  ",
			want:  "custom-collector:4317",
		},
		{
			input: "  https://custom-collector:4317  ",
			want:  "custom-collector:4317",
		},
		{
			input: "dns:custom-collector:4317",
			want:  "dns:custom-collector:4317",
		},
		{
			input: "",
			want:  "",
		},
	}

	for _, tc := range testCases {
		if got := normalizeOtlpEndpoint(tc.input); got != tc.want {
			t.Errorf("normalizeOtlpEndpoint(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// Tests the various combinations of tracing flags and environment variables for OTLP exporter resolution.
func TestOpenTelemetryConfig(t *testing.T) {
	testData := []struct {
		desc        string
		envEndpoint string
		opts        *options.TracingOptions
		wantError   string
		wantResult  *tracepb.OpenTelemetryConfig
	}{
		{
			desc: "Success with default tracing",
			opts: &options.TracingOptions{
				ProjectId: fakeOptsProjectId,
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "telemetry.googleapis.com",
							StatPrefix: "opentelemetry",
							ChannelCredentials: &corev3.GrpcService_GoogleGrpc_ChannelCredentials{
								CredentialSpecifier: &corev3.GrpcService_GoogleGrpc_ChannelCredentials_GoogleDefault{
									GoogleDefault: &emptypb.Empty{},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:        "OTEL_EXPORTER_OTLP_ENDPOINT with http scheme is normalized",
			envEndpoint: "http://custom-collector:4317",
			opts: &options.TracingOptions{
				ProjectId: fakeOptsProjectId,
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "custom-collector:4317",
							StatPrefix: "opentelemetry",
						},
					},
				},
			},
		},
		{
			desc:        "OTEL_EXPORTER_OTLP_ENDPOINT with https scheme is normalized",
			envEndpoint: "https://custom-collector:4317",
			opts: &options.TracingOptions{
				ProjectId: fakeOptsProjectId,
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "custom-collector:4317",
							StatPrefix: "opentelemetry",
						},
					},
				},
			},
		},
		{
			desc:        "OTEL_EXPORTER_OTLP_ENDPOINT overrides legacy --tracing_stackdriver_address flag",
			envEndpoint: "http://env-collector:4317",
			opts: &options.TracingOptions{
				ProjectId:          fakeOptsProjectId,
				StackdriverAddress: "flag-collector:4317",
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "env-collector:4317",
							StatPrefix: "opentelemetry",
						},
					},
				},
			},
		},
		{
			desc: "Fallback to legacy --tracing_stackdriver_address when env var is unset",
			opts: &options.TracingOptions{
				ProjectId:          fakeOptsProjectId,
				StackdriverAddress: "flag-collector:4317",
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "flag-collector:4317",
							StatPrefix: "opentelemetry",
						},
					},
				},
			},
		},
		{
			desc:        "OTEL_EXPORTER_OTLP_ENDPOINT with whitespace is trimmed",
			envEndpoint: "   custom-collector:4317   ",
			opts: &options.TracingOptions{
				ProjectId: fakeOptsProjectId,
			},
			wantResult: &tracepb.OpenTelemetryConfig{
				ServiceName: "espv2",
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
						GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
							TargetUri:  "custom-collector:4317",
							StatPrefix: "opentelemetry",
						},
					},
				},
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tc.envEndpoint)

			got, err := createOpenTelemetryConfig(*tc.opts)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("failed, expected err: %v, got: %v", tc.wantError, err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Errorf("failed, expected result should not be nil")
				}

				// The environment resource detector is serialized into unknown fields
				// (field 4) for compatibility with Envoy v1.38 without updating go-control-plane.
				// We attach the same unknown field to wantResult for protobuf comparison.
				envDetector := &corev3.TypedExtensionConfig{
					Name: "envoy.tracers.opentelemetry.resource_detectors.environment",
					TypedConfig: &anypb.Any{
						TypeUrl: "type.googleapis.com/envoy.extensions.tracers.opentelemetry.resource_detectors.v3.EnvironmentResourceDetectorConfig",
					},
				}
				detectorBytes, err := proto.Marshal(envDetector)
				if err != nil {
					t.Fatalf("failed to marshal envDetector: %v", err)
				}
				var wantUnknown []byte
				wantUnknown = protowire.AppendTag(wantUnknown, 4, protowire.BytesType)
				wantUnknown = protowire.AppendBytes(wantUnknown, detectorBytes)
				tc.wantResult.ProtoReflect().SetUnknown(wantUnknown)

				if diff := cmp.Diff(tc.wantResult, got, protocmp.Transform()); diff != "" {
					t.Errorf("createOpenTelemetryConfig(%v) diff (-want +got):\n%s", tc.opts, diff)
				}
			}
		})
	}
}

// Tests that the Environment resource detector is properly configured in OpenTelemetryConfig.
func TestOpenTelemetryResourceDetectors(t *testing.T) {
	opts := options.TracingOptions{
		ProjectId: fakeOptsProjectId,
	}

	cfg, err := createOpenTelemetryConfig(opts)
	if err != nil {
		t.Fatalf("createOpenTelemetryConfig() failed: %v", err)
	}

	unknown := cfg.ProtoReflect().GetUnknown()
	if len(unknown) == 0 {
		t.Fatalf("expected unknown fields containing resource detectors, got empty")
	}

	// Parse unknown fields to verify field 4 contains the environment detector
	var foundDetector bool
	for len(unknown) > 0 {
		num, typ, n := protowire.ConsumeTag(unknown)
		if n < 0 {
			t.Fatalf("invalid tag in unknown fields")
		}
		unknown = unknown[n:]
		if num == 4 && typ == protowire.BytesType {
			bytesVal, m := protowire.ConsumeBytes(unknown)
			if m < 0 {
				t.Fatalf("invalid bytes in field 4")
			}
			unknown = unknown[m:]

			detector := &corev3.TypedExtensionConfig{}
			if err := proto.Unmarshal(bytesVal, detector); err != nil {
				t.Fatalf("failed to unmarshal field 4 into TypedExtensionConfig: %v", err)
			}

			const wantName = "envoy.tracers.opentelemetry.resource_detectors.environment"
			if detector.Name != wantName {
				t.Errorf("detector.Name = %q, want %q", detector.Name, wantName)
			}

			const wantTypeUrl = "type.googleapis.com/envoy.extensions.tracers.opentelemetry.resource_detectors.v3.EnvironmentResourceDetectorConfig"
			if detector.TypedConfig == nil || detector.TypedConfig.TypeUrl != wantTypeUrl {
				t.Errorf("detector.TypedConfig.TypeUrl = %v, want %q", detector.TypedConfig, wantTypeUrl)
			}
			foundDetector = true
		} else {
			m := protowire.ConsumeFieldValue(num, typ, unknown)
			if m < 0 {
				t.Fatalf("invalid field value in unknown fields")
			}
			unknown = unknown[m:]
		}
	}

	if !foundDetector {
		t.Errorf("field 4 (resource_detectors) not found in OpenTelemetryConfig unknown fields")
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
					Name: "envoy.tracers.opentelemetry",
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
					Name: "envoy.tracers.opentelemetry",
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
					Name: "envoy.tracers.opentelemetry",
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
					Name: "envoy.tracers.opentelemetry",
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
					Name: "envoy.tracers.opentelemetry",
					// Typed config is already tested, so strip it out.
					ConfigType: nil,
				},
			},
		},
		{
			desc: "Invalid sampling rate has error",
			opts: options.TracingOptions{
				SamplingRate: 1.3,
			},
			wantError: "invalid trace sampling rate",
		},
		{
			desc: "Empty config when tracing is disabled",
			opts: options.TracingOptions{
				DisableTracing: true,
				SamplingRate:   options.DefaultCommonOptions().TracingOptions.SamplingRate,
			},
			wantResult: nil,
		},
		{
			desc: "Generate OpenTelemetry tracer when StackdriverAddress is specified",
			opts: options.TracingOptions{
				ProjectId:          "test-project",
				SamplingRate:       1.0,
				StackdriverAddress: "127.0.0.1:9990",
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
					Name:       "envoy.tracers.opentelemetry",
					ConfigType: nil,
				},
			},
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
				} else {
					if got != nil {
						t.Fatalf("Test (%s): failed, expected nil result, got: %v", tc.desc, got)
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

func TestParseResourceAttributes(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  map[string]string
	}{
		{
			desc:  "Single attribute",
			input: "gcp.project.id=my-project",
			want: map[string]string{
				"gcp.project.id": "my-project",
			},
		},
		{
			desc:  "Multiple attributes with whitespace",
			input: "  service.name=my-svc , gcp.project.id=my-project , service.version=1.0  ",
			want: map[string]string{
				"service.name":    "my-svc",
				"gcp.project.id":  "my-project",
				"service.version": "1.0",
			},
		},
		{
			desc:  "Quoted attribute values",
			input: `gcp.project.id="my-project",service.name='my-svc'`,
			want: map[string]string{
				"gcp.project.id": "my-project",
				"service.name":   "my-svc",
			},
		},
		{
			desc:  "Empty string",
			input: "",
			want:  map[string]string{},
		},
		{
			desc:  "Malformed entries ignored",
			input: "keyonly,valid=value,=valonly",
			want: map[string]string{
				"valid": "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := parseResourceAttributes(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseResourceAttributes(%q) diff (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestResolveTracingProjectId(t *testing.T) {
	testCases := []struct {
		desc            string
		envResourceAttr string
		optsProjectId   string
		wantProjectId   string
	}{
		{
			desc:            "Project ID resolved from OTEL_RESOURCE_ATTRIBUTES with gcp.project_id",
			envResourceAttr: "gcp.project_id=otel-project-underscore",
			optsProjectId:   "",
			wantProjectId:   "otel-project-underscore",
		},
		{
			desc:            "Project ID resolved from OTEL_RESOURCE_ATTRIBUTES with legacy gcp.project.id",
			envResourceAttr: "gcp.project.id=otel-project-dot",
			optsProjectId:   "",
			wantProjectId:   "otel-project-dot",
		},
		{
			desc:            "gcp.project_id takes precedence over gcp.project.id",
			envResourceAttr: "gcp.project.id=dot-project,gcp.project_id=underscore-project",
			optsProjectId:   "flag-project",
			wantProjectId:   "underscore-project",
		},
		{
			desc:            "OTEL_RESOURCE_ATTRIBUTES takes precedence over opts.ProjectId",
			envResourceAttr: "gcp.project.id=otel-project",
			optsProjectId:   "flag-project",
			wantProjectId:   "otel-project",
		},
		{
			desc:            "Fallback to opts.ProjectId when OTEL_RESOURCE_ATTRIBUTES is unset",
			envResourceAttr: "",
			optsProjectId:   "flag-project",
			wantProjectId:   "flag-project",
		},
		{
			desc:            "Fallback to opts.ProjectId when OTEL_RESOURCE_ATTRIBUTES lacks gcp.project.id",
			envResourceAttr: "service.name=my-svc,service.version=1.0",
			optsProjectId:   "flag-project",
			wantProjectId:   "flag-project",
		},
		{
			desc:            "Fallback to empty string when both env var and opts.ProjectId are empty",
			envResourceAttr: "",
			optsProjectId:   "",
			wantProjectId:   "",
		},
		{
			desc:            "Project ID resolved among multiple attributes with whitespace and quotes",
			envResourceAttr: `service.name=my-svc, gcp.project.id="quoted-project", environment=prod`,
			optsProjectId:   "flag-project",
			wantProjectId:   "quoted-project",
		},
		{
			desc:            "Fallback to opts.ProjectId when gcp.project.id attribute is empty",
			envResourceAttr: "gcp.project.id=,service.name=my-svc",
			optsProjectId:   "flag-project",
			wantProjectId:   "flag-project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", tc.envResourceAttr)

			opts := options.TracingOptions{
				ProjectId: tc.optsProjectId,
			}
			got := ResolveTracingProjectId(opts)
			if got != tc.wantProjectId {
				t.Errorf("ResolveTracingProjectId(%v) = %q, want %q", opts, got, tc.wantProjectId)
			}
		})
	}
}
