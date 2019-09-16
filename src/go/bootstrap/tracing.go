// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"fmt"
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/metadata"
	"cloudesf.googlesource.com/gcpproxy/src/go/options"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
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

// CreateTracing outputs envoy tracing config
func CreateTracing(opts options.CommonOptions) (*tracepb.Tracing, error) {

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

	if opts.TracingSamplingRate == 1.0 {
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensuspb.ConstantSampler{
				Decision: opencensuspb.ConstantSampler_ALWAYS_ON,
			},
		}
	} else if opts.TracingSamplingRate == 0.0 {
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensuspb.ConstantSampler{
				Decision: opencensuspb.ConstantSampler_ALWAYS_PARENT,
			},
		}
	} else {
		if opts.TracingSamplingRate < 0.0 || opts.TracingSamplingRate > 1.0 {
			return nil, fmt.Errorf("Invalid trace sampling rate: %v. It must be >= 0.0 and <= 1.0", opts.TracingSamplingRate)
		}
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ProbabilitySampler{
			ProbabilitySampler: &opencensuspb.ProbabilitySampler{
				SamplingProbability: opts.TracingSamplingRate,
			},
		}
	}

	v, _ := ptypes.MarshalAny(cfg)
	return &tracepb.Tracing{
		Http: &tracepb.Tracing_Http{
			Name:       "envoy.tracers.opencensus",
			ConfigType: &tracepb.Tracing_Http_TypedConfig{TypedConfig: v},
		},
	}, nil
}
