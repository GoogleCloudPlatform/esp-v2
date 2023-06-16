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
	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/glog"
	"google.golang.org/protobuf/types/known/anypb"
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

// MaybeFetchTracingProjectID fetches the tenant project ID from IMDS if it is
// not specified by the user.
func MaybeFetchTracingProjectID(opts options.CommonOptions, imdsFetcher *metadata.MetadataFetcher) (string, error) {

	// If user specified a project-id, use that
	projectId := opts.TracingOptions.ProjectId
	if projectId != "" {
		return projectId, nil
	}

	// Otherwise determine project-id automatically
	glog.Infof("tracing_project_id was not specified, attempting to fetch it from GCP Metadata server")
	if opts.NonGCP {
		return "", fmt.Errorf("tracing_project_id was not specified and can not be fetched from GCP Metadata server on non-GCP runtime")
	}

	return imdsFetcher.FetchProjectId()
}

func createOpenCensusConfig(opts options.TracingOptions) (*tracepb.OpenCensusConfig, error) {
	cfg := &tracepb.OpenCensusConfig{
		TraceConfig: &opencensuspb.TraceConfig{
			MaxNumberOfAttributes:    opts.MaxNumAttributes,
			MaxNumberOfAnnotations:   opts.MaxNumAnnotations,
			MaxNumberOfMessageEvents: opts.MaxNumMessageEvents,
			MaxNumberOfLinks:         opts.MaxNumLinks,
		},
		StackdriverExporterEnabled: true,
		StackdriverProjectId:       opts.ProjectId,
	}

	if opts.StackdriverAddress != "" {
		cfg.StackdriverAddress = opts.StackdriverAddress
	}

	if ctx, err := createTraceContexts(opts.IncomingContext); err == nil {
		cfg.IncomingTraceContext = ctx
	} else {
		return nil, err
	}

	if ctx, err := createTraceContexts(opts.OutgoingContext); err == nil {
		cfg.OutgoingTraceContext = ctx
	} else {
		return nil, err
	}

	// Tracing sample rate in OpenCensusConfig is not used at all by Envoy.
	// No need to set it.

	return cfg, nil
}

// CreateTracing outputs envoy HCM tracing config.
func CreateTracing(opts options.TracingOptions) (*hcmpb.HttpConnectionManager_Tracing, error) {
	if opts.ProjectId == "" {
		return nil, fmt.Errorf("cannot create tracing config because tracing project ID is empty")
	}

	openCensusConfig, err := createOpenCensusConfig(opts)
	if err != nil {
		return nil, err
	}

	typedConfig, err := anypb.New(openCensusConfig)
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
			Name:       "envoy.tracers.opencensus",
			ConfigType: &tracepb.Tracing_Http_TypedConfig{TypedConfig: typedConfig},
		},
		Verbose: opts.EnableVerboseAnnotations,
	}, nil
}
