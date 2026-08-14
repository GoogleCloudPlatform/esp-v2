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
			Desc: "No-op when outgoing context is not legacy format",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "traceparent",
					},
				},
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "No-op when outgoing context is empty",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						OutgoingContext: "",
					},
				},
			},
			WantFilterConfigs: nil,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergen.NewTraceContextFilterGensFromOPConfig)
	}
}
