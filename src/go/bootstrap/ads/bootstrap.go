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

package ads

import (
	"github.com/golang/protobuf/ptypes/duration"

	bt "cloudesf.googlesource.com/gcpproxy/src/go/bootstrap"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	bootstrappb "github.com/envoyproxy/data-plane-api/api/bootstrap"
	cdspb "github.com/envoyproxy/data-plane-api/api/cds"
	configsourcepb "github.com/envoyproxy/data-plane-api/api/config_source"
	grpcservicepb "github.com/envoyproxy/data-plane-api/api/grpc_service"
	protocolpb "github.com/envoyproxy/data-plane-api/api/protocol"
)

// CreateBoostrapConfig outputs envoy bootstrap config for xDS.
func CreateBootstrapConfig(ads_connect_timeout *duration.Duration) *bootstrappb.Bootstrap {
	return &bootstrappb.Bootstrap{
		// Node info
		Node: bt.CreateNode(),

		// admin
		Admin: bt.CreateAdmin(),

		// Dynamic resource
		DynamicResources: &bootstrappb.Bootstrap_DynamicResources{
			LdsConfig: &configsourcepb.ConfigSource{
				ConfigSourceSpecifier: &configsourcepb.ConfigSource_Ads{
					Ads: &configsourcepb.AggregatedConfigSource{},
				},
			},
			CdsConfig: &configsourcepb.ConfigSource{
				ConfigSourceSpecifier: &configsourcepb.ConfigSource_Ads{
					Ads: &configsourcepb.AggregatedConfigSource{},
				},
			},
			AdsConfig: &configsourcepb.ApiConfigSource{
				ApiType: configsourcepb.ApiConfigSource_GRPC,
				GrpcServices: []*grpcservicepb.GrpcService{{
					TargetSpecifier: &grpcservicepb.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &grpcservicepb.GrpcService_EnvoyGrpc{
							ClusterName: "ads_cluster",
						},
					},
				}},
			},
		},

		// Static resource
		StaticResources: &bootstrappb.Bootstrap_StaticResources{
			Clusters: []*cdspb.Cluster{
				{
					Name:           "ads_cluster",
					LbPolicy:       cdspb.Cluster_ROUND_ROBIN,
					ConnectTimeout: ads_connect_timeout,
					ClusterDiscoveryType: &cdspb.Cluster_Type{
						Type: cdspb.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &protocolpb.Http2ProtocolOptions{},
					LoadAssignment:       ut.CreateLoadAssignment("127.0.0.1", 8790),
				},
			},
		},
	}
}
