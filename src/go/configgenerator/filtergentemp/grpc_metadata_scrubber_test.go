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

package filtergentemp_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergentemp"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/imdario/mergo"
)

func TestNewGRPCMetadataScrubberFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testdata := []SuccessOPTestCase{
		{
			Desc: "Generate when gRPC for HTTP/1.x is enabled",
			OptsIn: options.ConfigGeneratorOptions{
				EnableGrpcForHttp1: true,
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.grpc_metadata_scrubber",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.grpc_metadata_scrubber.FilterConfig"
   }
}
`,
			},
		},
		{
			Desc: "No-op when gRPC for HTTP/1.x is disabled",
			OptsIn: options.ConfigGeneratorOptions{
				EnableGrpcForHttp1: false,
			},
			OptsMergeBehavior: mergo.WithOverwriteWithEmptyValue,
			WantFilterConfigs: nil,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergentemp.NewGRPCMetadataScrubberFilterGensFromOPConfig)
	}
}
