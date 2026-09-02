// Copyright 2023 Google LLC
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

package filtergen_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen/filtergentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func TestNewTraceContextFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testdata := []filtergentest.SuccessOPTestCase{
		{
			Desc: "Generate trace context for cloud trace",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "x-cloud-trace-context",
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.trace_context",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextForwardedConfig",
      "outgoingContexts":[
         "CLOUD_TRACE_CONTEXT"
      ]
   }
}
`,
			},
		},
		{
			Desc: "Generate trace context for grpc trace bin",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "grpc-trace-bin",
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.trace_context",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextForwardedConfig",
      "outgoingContexts":[
         "GRPC_TRACE_BIN"
      ]
   }
}
`,
			},
		},
		{
			Desc: "Generate trace context for both",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "grpc-trace-bin,x-cloud-trace-context",
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.trace_context",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextForwardedConfig",
      "outgoingContexts":[
         "GRPC_TRACE_BIN",
         "CLOUD_TRACE_CONTEXT"
      ]
   }
}
`,
			},
		},
		{
			Desc: "Outputs traceparent config when traceparent is explicitly requested",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "traceparent",
					},
				},
			},
			WantFilterConfigs: []string{
				`{"name":"com.google.espv2.filters.http.trace_context","typedConfig":{"@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextForwardedConfig","outgoingContexts":["TRACE_CONTEXT"]}}`,
			},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergen.NewTraceContextFilterGensFromOPConfig)
	}
}

func TestGenEarlyHeaderMutationConfig(t *testing.T) {
	testcases := []struct {
		desc     string
		incoming string
		want     string
	}{
		{
			desc:     "Generate for cloud trace",
			incoming: "x-cloud-trace-context",
			want:     `{"name":"com.google.espv2.filters.http.early_header_mutation.trace_context","typedConfig":{"@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextTranslatorConfig","incomingContexts":["CLOUD_TRACE_CONTEXT"]}}`,
		},
		{
			desc:     "Generate for grpc trace bin",
			incoming: "grpc-trace-bin",
			want:     `{"name":"com.google.espv2.filters.http.early_header_mutation.trace_context","typedConfig":{"@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextTranslatorConfig","incomingContexts":["GRPC_TRACE_BIN"]}}`,
		},
		{
			desc:     "Outputs traceparent config when explicitly requested",
			incoming: "traceparent",
			want:     `{"name":"com.google.espv2.filters.http.early_header_mutation.trace_context","typedConfig":{"@type":"type.googleapis.com/envoy.v12.http.trace_context.TraceContextTranslatorConfig","incomingContexts":["TRACE_CONTEXT"]}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := filtergen.GenEarlyHeaderMutationConfig(tc.incoming)
			if err != nil {
				t.Fatalf("GenEarlyHeaderMutationConfig(%s) failed: %v", tc.incoming, err)
			}
			if tc.want == "" {
				if got != nil {
					t.Errorf("GenEarlyHeaderMutationConfig(%s) = %v, want nil", tc.incoming, got)
				}
				return
			}

			gotJSON, err := util.ProtoToJson(got)
			if err != nil {
				t.Fatalf("Failed to marshal got proto to json: %v", err)
			}

			if err := util.JsonEqual(tc.want, gotJSON); err != nil {
				t.Errorf("GenEarlyHeaderMutationConfig(%s) mismatch: %v", tc.incoming, err)
			}
		})
	}
}
