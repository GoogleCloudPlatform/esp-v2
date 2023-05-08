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
)

func TestNewCompressorFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testdata := []SuccessOPTestCase{
		{
			Desc: "Generate with compression enabled",
			OptsIn: options.ConfigGeneratorOptions{
				EnableResponseCompression: true,
			},
			WantFilterConfigs: []string{
				`
{
   "name":"envoy.filters.http.compressor",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.compressor.v3.Compressor",
      "compressorLibrary":{
         "name":"envoy.compression.gzip.compressor",
         "typedConfig":{
            "@type":"type.googleapis.com/envoy.extensions.compression.gzip.compressor.v3.Gzip"
         }
      }
   }
}
`,
				`
{
   "name":"envoy.filters.http.compressor",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.compressor.v3.Compressor",
      "compressorLibrary":{
         "name":"envoy.compression.brotli.compressor",
         "typedConfig":{
            "@type":"type.googleapis.com/envoy.extensions.compression.brotli.compressor.v3.Brotli"
         }
      }
   }
}
`,
			},
		},
		{
			Desc: "No-op when opt is disabled",
			OptsIn: options.ConfigGeneratorOptions{
				EnableResponseCompression: false,
			},
			WantFilterConfigs: nil,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergentemp.NewCompressorFilterGensFromOPConfig)
	}
}
