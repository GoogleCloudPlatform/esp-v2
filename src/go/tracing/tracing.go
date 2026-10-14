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
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/glog"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// normalizeOtlpEndpoint trims leading/trailing whitespace and strips "http://" or
// "https://" prefixes since Envoy's GoogleGrpc.TargetUri expects a gRPC target
// string rather than an HTTP URL scheme.
func normalizeOtlpEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

// parseResourceAttributes parses a comma-separated key=value string (as specified in
// the OpenTelemetry specification for OTEL_RESOURCE_ATTRIBUTES) into a map.
func parseResourceAttributes(rawAttrs string) map[string]string {
	attrs := make(map[string]string)
	for _, pair := range strings.Split(rawAttrs, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if k != "" {
				attrs[k] = v
			}
		}
	}
	return attrs
}

// ResolveTracingProjectId resolves the GCP Project ID for tracing and Service Control
// using the following precedence order:
// 1. "gcp.project_id" (standard) or "gcp.project.id" attribute from the OTEL_RESOURCE_ATTRIBUTES environment variable.
// 2. opts.ProjectId (fallback for the deprecated --tracing_project_id flag).
// 3. Default: empty string "" (falls back to GCP metadata server / ADC resolution).
func ResolveTracingProjectId(opts options.TracingOptions) string {
	rawAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
	if rawAttrs != "" {
		attrs := parseResourceAttributes(rawAttrs)
		// Check standard "gcp.project_id" first, then legacy "gcp.project.id".
		projectID, ok := attrs["gcp.project_id"]
		if !ok || projectID == "" {
			projectID, ok = attrs["gcp.project.id"]
		}
		if ok && projectID != "" {
			if opts.ProjectId != "" {
				glog.Infof("Both OTEL_RESOURCE_ATTRIBUTES (project_id=%q) and --tracing_project_id (%q) are configured. Using OTEL_RESOURCE_ATTRIBUTES.", projectID, opts.ProjectId)
			}
			return projectID
		}
	}

	if opts.ProjectId != "" {
		return opts.ProjectId
	}

	return ""
}

func createOpenTelemetryConfig(opts options.TracingOptions) (*tracepb.OpenTelemetryConfig, error) {
	// Exporter destination precedence:
	// 1. OTEL_EXPORTER_OTLP_ENDPOINT environment variable.
	// 2. opts.StackdriverAddress (fallback for deprecated --tracing_stackdriver_address).
	// 3. Default: "telemetry.googleapis.com" (Google Cloud Trace).
	targetURI := "telemetry.googleapis.com"
	envEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if envEndpoint != "" {
		targetURI = normalizeOtlpEndpoint(envEndpoint)
		if opts.StackdriverAddress != "" {
			glog.Infof("Both OTEL_EXPORTER_OTLP_ENDPOINT (%q) and --tracing_stackdriver_address (%q) are configured. Using OTEL_EXPORTER_OTLP_ENDPOINT.", envEndpoint, opts.StackdriverAddress)
		}
	} else if opts.StackdriverAddress != "" {
		targetURI = opts.StackdriverAddress
	}

	googleGrpc := &corev3.GrpcService_GoogleGrpc{
		TargetUri:  targetURI,
		StatPrefix: "opentelemetry",
	}
	if targetURI == "telemetry.googleapis.com" || strings.HasPrefix(targetURI, "telemetry.googleapis.com:") {
		googleGrpc.ChannelCredentials = &corev3.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &corev3.GrpcService_GoogleGrpc_ChannelCredentials_GoogleDefault{
				GoogleDefault: &emptypb.Empty{},
			},
		}
	}

	cfg := &tracepb.OpenTelemetryConfig{
		ServiceName: "espv2", // Provide a default service name.
		GrpcService: &corev3.GrpcService{
			TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
				GoogleGrpc: googleGrpc,
			},
		},
	}

	// In the pinned go-control-plane version, tracepb.OpenTelemetryConfig lacks the
	// ResourceDetectors field (field number 4 in Envoy's OpenTelemetryConfig proto).
	// We serialize the environment resource detector into the message's unknown fields
	// so Envoy v1.38 unmarshals it and enables the environment detector.
	envDetector := &corev3.TypedExtensionConfig{
		Name: "envoy.tracers.opentelemetry.resource_detectors.environment",
		TypedConfig: &anypb.Any{
			TypeUrl: "type.googleapis.com/envoy.extensions.tracers.opentelemetry.resource_detectors.v3.EnvironmentResourceDetectorConfig",
		},
	}
	detectorBytes, err := proto.Marshal(envDetector)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment resource detector: %w", err)
	}

	// Field 4, WireType: BytesType (2) -> Tag = (4 << 3) | 2 = 34
	var unknown []byte
	unknown = protowire.AppendTag(unknown, 4, protowire.BytesType)
	unknown = protowire.AppendBytes(unknown, detectorBytes)
	cfg.ProtoReflect().SetUnknown(unknown)

	return cfg, nil
}

// CreateTracing outputs envoy HCM tracing config.
func CreateTracing(opts options.TracingOptions) (*hcmpb.HttpConnectionManager_Tracing, error) {
	if opts.DisableTracing {
		return nil, nil
	}

	openTelemetryConfig, err := createOpenTelemetryConfig(opts)
	if err != nil {
		return nil, err
	}

	typedConfig, err := anypb.New(openTelemetryConfig)
	if err != nil {
		return nil, err
	}

	if opts.SamplingRate < 0.0 || opts.SamplingRate > 1.0 {
		return nil, fmt.Errorf("invalid trace sampling rate: %v. It must be >= 0.0 and <= 1.0", opts.SamplingRate)
	}

	// This results in precision errors. Round percentage to 4 decimal points.
	percentSampleRate := opts.SamplingRate * 100
	percentSampleRate = math.Round(percentSampleRate*10000) / 10000

	return &hcmpb.HttpConnectionManager_Tracing{
		ClientSampling: &typepb.Percent{
			Value: 0,
		},
		RandomSampling: &typepb.Percent{
			Value: percentSampleRate,
		},
		OverallSampling: &typepb.Percent{
			Value: percentSampleRate,
		},
		Provider: &tracepb.Tracing_Http{
			Name:       "envoy.tracers.opentelemetry",
			ConfigType: &tracepb.Tracing_Http_TypedConfig{TypedConfig: typedConfig},
		},
		Verbose: opts.EnableVerboseAnnotations,
	}, nil
}
