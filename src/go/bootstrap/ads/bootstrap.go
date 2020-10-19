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

package ads

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/ptypes"

	bt "github.com/GoogleCloudPlatform/esp-v2/src/go/bootstrap"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// CreateBootstrapConfig outputs envoy bootstrap config for xDS.
func CreateBootstrapConfig(opts options.AdsBootstrapperOptions) (string, error) {
	apiVersion := corepb.ApiVersion_V3

	// Parse ADS connect timeout
	connectTimeoutProto := ptypes.DurationProto(opts.AdsConnectTimeout)

	bt := &bootstrappb.Bootstrap{
		// Node info
		Node: bt.CreateNode(opts.CommonOptions),

		// admin
		Admin: bt.CreateAdmin(opts.CommonOptions),

		// layer runtime
		LayeredRuntime: bt.CreateLayeredRuntime(),

		// Dynamic resource
		DynamicResources: &bootstrappb.Bootstrap_DynamicResources{
			LdsConfig: &corepb.ConfigSource{
				ConfigSourceSpecifier: &corepb.ConfigSource_Ads{
					Ads: &corepb.AggregatedConfigSource{},
				},
				ResourceApiVersion: apiVersion,
			},
			CdsConfig: &corepb.ConfigSource{
				ConfigSourceSpecifier: &corepb.ConfigSource_Ads{
					Ads: &corepb.AggregatedConfigSource{},
				},
				ResourceApiVersion: apiVersion,
			},
			AdsConfig: &corepb.ApiConfigSource{
				ApiType:             corepb.ApiConfigSource_GRPC,
				TransportApiVersion: apiVersion,
				GrpcServices: []*corepb.GrpcService{{
					TargetSpecifier: &corepb.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &corepb.GrpcService_EnvoyGrpc{
							ClusterName: opts.AdsNamedPipe,
						},
					},
				}},
			},
		},

		// Static resource
		StaticResources: &bootstrappb.Bootstrap_StaticResources{
			Clusters: []*clusterpb.Cluster{
				{
					Name:           opts.AdsNamedPipe,
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: connectTimeoutProto,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STATIC,
					},
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
					LoadAssignment:       util.CreateUdsLoadAssignment(opts.AdsNamedPipe),
				},
			},
		},
	}

	jsonStr, err := util.ProtoToJson(bt)
	if err != nil {
		return "", fmt.Errorf("failed to MarshalToString, error: %v", err)
	}
	return jsonStr, nil
}
