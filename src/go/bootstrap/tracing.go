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
	"flag"
	"fmt"
	"strings"

	types "github.com/gogo/protobuf/types"

	trace "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
	opencensus "istio.io/gogo-genproto/opencensus/proto/trace/v1"
)

var (
	TracingSamplingRate    = flag.Float64("tracing_sample_rate", 0.001, "tracing sampling rate from 0.0 to 1.0")
	TracingIncomingContext = flag.String("tracing_incoming_context", "", "comma separated incoming trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
	TracingOutgoingContext = flag.String("tracing_outgoing_context", "", "comma separated outgoing trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")

	TracingProjectId = flag.String("tracing_project_id", "", "The Google project id required for Stack driver tracing")
)

func createTraceContexts(ctx_str string) ([]trace.OpenCensusConfig_TraceContext, error) {
	out := []trace.OpenCensusConfig_TraceContext{}

	if ctx_str == "" {
		return out, nil
	}

	for _, ctx := range strings.Split(ctx_str, ",") {
		switch ctx {
		case "traceparent":
			out = append(out, trace.OpenCensusConfig_trace_context)
		case "grpc-trace-bin":
			out = append(out, trace.OpenCensusConfig_grpc_trace_bin)
		case "x-cloud-trace-context":
			out = append(out, trace.OpenCensusConfig_cloud_trace_context)
		default:
			return out, fmt.Errorf("Invalid trace context: %v. It must be one of (traceparent|grpc-trace-bin|x-cloud-trace-context)", ctx)
		}
	}

	return out, nil
}

// CreateTracing outputs envoy tracing config
func CreateTracing() (*trace.Tracing, error) {
	if *TracingProjectId == "" {
		return nil, fmt.Errorf("tracing_project_id must be specified for StackDriver tracing")
	}

	cfg := &trace.OpenCensusConfig{
		TraceConfig:                &opencensus.TraceConfig{},
		StackdriverExporterEnabled: true,
		StackdriverProjectId:       *TracingProjectId,
	}

	if ctx, err := createTraceContexts(*TracingIncomingContext); err == nil {
		cfg.IncomingTraceContext = ctx
	} else {
		return nil, err
	}

	if ctx, err := createTraceContexts(*TracingOutgoingContext); err == nil {
		cfg.OutgoingTraceContext = ctx
	} else {
		return nil, err
	}

	if *TracingSamplingRate == 1.0 {
		cfg.TraceConfig.Sampler = &opencensus.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensus.ConstantSampler{
				Decision: opencensus.ConstantSampler_ALWAYS_ON,
			},
		}
	} else if *TracingSamplingRate == 0.0 {
		cfg.TraceConfig.Sampler = &opencensus.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensus.ConstantSampler{
				Decision: opencensus.ConstantSampler_ALWAYS_PARENT,
			},
		}
	} else {
		if *TracingSamplingRate < 0.0 || *TracingSamplingRate > 1.0 {
			return nil, fmt.Errorf("Invalid trace sampling rate: %v. It must be >= 0.0 and <= 1.0", *TracingSamplingRate)
		}
		cfg.TraceConfig.Sampler = &opencensus.TraceConfig_ProbabilitySampler{
			ProbabilitySampler: &opencensus.ProbabilitySampler{
				SamplingProbability: *TracingSamplingRate,
			},
		}
	}

	v, _ := types.MarshalAny(cfg)
	return &trace.Tracing{
		Http: &trace.Tracing_Http{
			Name:       "envoy.tracers.opencensus",
			ConfigType: &trace.Tracing_Http_TypedConfig{v},
		},
	}, nil
}
