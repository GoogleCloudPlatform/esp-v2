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
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tcpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/trace_context"
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
	// The C++ wrappers will only be injected (loaded) into the Envoy Listener/Route configuration if the customer's flags explicitly request legacy formats (x-cloud-trace-context or grpc-trace-bin).
	formats := parseLegacyFormats(opts.TracingOptions.OutgoingContext)
	
	if len(formats) == 0 {
		return nil, nil // Do not inject the C++ filter if legacy formats are not requested
	}

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
	formats := parseLegacyFormats(incoming)
	if len(formats) == 0 {
		return nil, nil // Do not inject if no legacy formats requested
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
