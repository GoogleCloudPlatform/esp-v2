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

	"github.com/golang/protobuf/ptypes"

	opencensuspb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	tracepb "github.com/envoyproxy/data-plane-api/api/trace"
)

var (
	TracingStackdriverAddress = flag.String("tracing_stackdriver_address", "", "By default, the Stackdriver exporter will connect to production Stackdriver. If this is non-empty, it will connect to this address. It must be in the gRPC format.")
	TracingSamplingRate       = flag.Float64("tracing_sample_rate", 0.001, "tracing sampling rate from 0.0 to 1.0")
	TracingIncomingContext    = flag.String("tracing_incoming_context", "", "comma separated incoming trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
	TracingOutgoingContext    = flag.String("tracing_outgoing_context", "", "comma separated outgoing trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
)

func createTraceContexts(ctx_str string) ([]tracepb.OpenCensusConfig_TraceContext, error) {
	out := []tracepb.OpenCensusConfig_TraceContext{}

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

// CreateTracing outputs envoy tracing config
func CreateTracing(tracingProjectId string) (*tracepb.Tracing, error) {

	cfg := &tracepb.OpenCensusConfig{
		TraceConfig:                &opencensuspb.TraceConfig{},
		StackdriverExporterEnabled: true,
		StackdriverProjectId:       tracingProjectId,
	}

	if *TracingStackdriverAddress != "" {
		cfg.StackdriverAddress = *TracingStackdriverAddress
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
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensuspb.ConstantSampler{
				Decision: opencensuspb.ConstantSampler_ALWAYS_ON,
			},
		}
	} else if *TracingSamplingRate == 0.0 {
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ConstantSampler{
			ConstantSampler: &opencensuspb.ConstantSampler{
				Decision: opencensuspb.ConstantSampler_ALWAYS_PARENT,
			},
		}
	} else {
		if *TracingSamplingRate < 0.0 || *TracingSamplingRate > 1.0 {
			return nil, fmt.Errorf("Invalid trace sampling rate: %v. It must be >= 0.0 and <= 1.0", *TracingSamplingRate)
		}
		cfg.TraceConfig.Sampler = &opencensuspb.TraceConfig_ProbabilitySampler{
			ProbabilitySampler: &opencensuspb.ProbabilitySampler{
				SamplingProbability: *TracingSamplingRate,
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
