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
							// Our service control filter may call route() in log time
							// but it is possible that the route isn't set with early local reply,
							// which triggers an ENVOY_BUG, so we use this flag to workaround.
							// For more context, see https://github.com/envoyproxy/envoy/issues/28626.
							"envoy.reloadable_features.prohibit_route_refresh_after_response_headers_sent": {
								Kind: &structpb.Value_BoolValue{
									BoolValue: false,
								},
							},
						},
					},
				},
			},
		},
	}
}
