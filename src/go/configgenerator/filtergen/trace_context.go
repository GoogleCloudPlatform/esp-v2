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

package filtergen

import (
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	tcpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/trace_context"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// TraceContextFilterName is the specific outgoing filter name mapped dynamically to the backend HTTP pipeline.
	TraceContextFilterName = "com.google.espv2.filters.http.trace_context"
)

type TraceContextGenerator struct {
	OutgoingContexts []tcpb.TraceContextFormat
	NoopFilterGenerator
}

func parseLegacyFormats(outgoing string) []tcpb.TraceContextFormat {
	var formats []tcpb.TraceContextFormat
	for _, p := range strings.Split(outgoing, ",") {
		switch strings.TrimSpace(p) {
		case "traceparent":
			formats = append(formats, tcpb.TraceContextFormat_TRACE_CONTEXT)
		case "x-cloud-trace-context":
			formats = append(formats, tcpb.TraceContextFormat_CLOUD_TRACE_CONTEXT)
		case "grpc-trace-bin":
			formats = append(formats, tcpb.TraceContextFormat_GRPC_TRACE_BIN)
		}
	}
	return formats
}

// NewTraceContextFilterGensFromOPConfig acts as a FilterGeneratorOPFactory.
func NewTraceContextFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	// The C++ wrappers will always be injected to handle x-cloud-trace-context or grpc-trace-bin.
	// It is also needed even when outgoing contexts are empty, to aggressively strip traceparent overrides from OpenTelemetry.
	if opts.TracingOptions.DisableTracing {
		return nil, nil
	}
	formats := parseLegacyFormats(opts.TracingOptions.OutgoingContext)

	return []FilterGenerator{
		&TraceContextGenerator{
			OutgoingContexts: formats,
		},
	}, nil
}

func (g *TraceContextGenerator) FilterName() string {
	return TraceContextFilterName
}

func (g *TraceContextGenerator) GenFilterConfig() (proto.Message, error) {
	return &tcpb.TraceContextForwardedConfig{
		OutgoingContexts: g.OutgoingContexts,
	}, nil
}

// GenEarlyHeaderMutationConfig generates the early header mutation extension for incoming trace contexts.
func GenEarlyHeaderMutationConfig(incoming string) (*corepb.TypedExtensionConfig, error) {
	var formats []tcpb.TraceContextFormat
	if incoming != "" {
		formats = parseLegacyFormats(incoming)
	}

	config := &tcpb.TraceContextTranslatorConfig{
		IncomingContexts: formats,
	}
	serialized, err := anypb.New(config)
	if err != nil {
		return nil, err
	}

	return &corepb.TypedExtensionConfig{
		Name:        "com.google.espv2.filters.http.early_header_mutation.trace_context",
		TypedConfig: serialized,
	}, nil
}
