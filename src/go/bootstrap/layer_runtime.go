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

package bootstrap

import (
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateLayeredRuntime outputs LayeredRuntime struct for bootstrap config
func CreateLayeredRuntime() *bootstrappb.LayeredRuntime {

	return &bootstrappb.LayeredRuntime{
		Layers: []*bootstrappb.RuntimeLayer{
			//
			{
				Name: "static-runtime",
				LayerSpecifier: &bootstrappb.RuntimeLayer_StaticLayer{
					StaticLayer: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"re2.max_program_size.error_level": {
								Kind: &structpb.Value_NumberValue{
									NumberValue: 1000,
								},
							},
							// Enable an Envoy vulnerability mitigation. For details, please see b/299661830.
							"http.max_requests_per_io_cycle": {
								Kind: &structpb.Value_NumberValue{
									NumberValue: 1,
								},
							},
						},
					},
				},
			},
		},
	}
}
