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
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
)

func createTraceContexts(ctx_str string) ([]tracepb.OpenCensusConfig_TraceContext, error) {
	var out []tracepb.OpenCensusConfig_TraceContext

	if ctx_str == "" {
		return out, nil
	}

	for _, ctx := range strings.Split(ctx_str, ",") {
		switch ctx {
		case "traceparent":
			out = append(out, tracepb.OpenCensusConfig_TRACE_CONTEXT)
		case "grpc-trace-bin":
			out = append(out, tracepb.OpenCensusConfig_GRPC_TRACE_BIN)
		case "x-cloud-trace-context":
			out = append(out, tracepb.OpenCensusConfig_CLOUD_TRACE_CONTEXT)
		default:
			return out, fmt.Errorf("Invalid trace context: %v. It must be one of (traceparent|grpc-trace-bin|x-cloud-trace-context)", ctx)
		}
	}

	return out, nil
}

func getTracingProjectId(opts options.CommonOptions) (string, error) {

	// If user specified a project-id, use that
	projectId := opts.TracingProjectId
	if projectId != "" {
		return projectId, nil
	}

	// Otherwise determine project-id automatically
	glog.Infof("tracing_project_id was not specified, attempting to fetch it from GCP Metadata server")
	if opts.NonGCP {
		return "", fmt.Errorf("tracing_project_id was not specified and can not be fetched from GCP Metadata server on non-GCP runtime")
	}

	return metadata.NewMetadataFetcher(opts).FetchProjectId()
}

func createOpenCensusConfig(opts options.CommonOptions) (*tracepb.OpenCensusConfig, error) {
	projectId, err := getTracingProjectId(opts)
	if err != nil {
		return nil, err
	}

	cfg := &tracepb.OpenCensusConfig{
		TraceConfig: &opencensuspb.TraceConfig{
			MaxNumberOfAttributes:    opts.TracingMaxNumAttributes,
			MaxNumberOfAnnotations:   opts.TracingMaxNumAnnotations,
			MaxNumberOfMessageEvents: opts.TracingMaxNumMessageEvents,
			MaxNumberOfLinks:         opts.TracingMaxNumLinks,
		},
		StackdriverExporterEnabled: true,
		StackdriverProjectId:       projectId,
	}

	if opts.TracingStackdriverAddress != "" {
		cfg.StackdriverAddress = opts.TracingStackdriverAddress
	}

	if ctx, err := createTraceContexts(opts.TracingIncomingContext); err == nil {
		cfg.IncomingTraceContext = ctx
	} else {
		return nil, err
	}

	if ctx, err := createTraceContexts(opts.TracingOutgoingContext); err == nil {
		cfg.OutgoingTraceContext = ctx
	} else {
		return nil, err
	}

	// Tracing sample rate in OpenCensusConfig is not used at all by Envoy.
	// No need to set it.

	return cfg, nil
}

// CreateTracing outputs envoy HCM tracing config.
func CreateTracing(opts options.CommonOptions) (*hcmpb.HttpConnectionManager_Tracing, error) {

	openCensusConfig, err := createOpenCensusConfig(opts)
	if err != nil {
		return nil, err
	}

	typedConfig, err := ptypes.MarshalAny(openCensusConfig)
	if err != nil {
		return nil, err
	}

	if opts.TracingSamplingRate < 0.0 || opts.TracingSamplingRate > 1.0 {
		return nil, fmt.Errorf("invalid trace sampling rate: %v. It must be >= 0.0 and <= 1.0", opts.TracingSamplingRate)
	}

	// This results in precision errors. Round percentage to 4 decimal points.
	percentSampleRate := opts.TracingSamplingRate * 100
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
			Name:       "envoy.tracers.opencensus",
			ConfigType: &tracepb.Tracing_Http_TypedConfig{TypedConfig: typedConfig},
		},
		Verbose: opts.TracingEnableVerboseAnnotations,
	}, nil
}
