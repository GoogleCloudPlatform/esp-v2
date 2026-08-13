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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

func createOpenTelemetryConfig(opts options.TracingOptions) (*tracepb.OpenTelemetryConfig, error) {
	// Stackdriver Export via OTLP directly accesses the Google Cloud Telemetry API.
	targetUri := "telemetry.googleapis.com"

	cfg := &tracepb.OpenTelemetryConfig{
		ServiceName: "espv2", // Provide a default service name.
		GrpcService: &corev3.GrpcService{
			TargetSpecifier: &corev3.GrpcService_GoogleGrpc_{
				GoogleGrpc: &corev3.GrpcService_GoogleGrpc{
					TargetUri:  targetUri,
					StatPrefix: "opentelemetry",
				},
			},
		},
	}

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
