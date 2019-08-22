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
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

// CreateBoostrapConfig outputs envoy bootstrap config for xDS.
func CreateBootstrapConfig(adsConnectTimeout *duration.Duration, adsHostname string, adsPort uint32, adminPort uint32) (*bootstrappb.Bootstrap, error) {

	return &bootstrappb.Bootstrap{
		// Node info
		Node: bt.CreateNode(),

		// admin
		Admin: bt.CreateAdmin(adminPort),

		// Dynamic resource
		DynamicResources: &bootstrappb.Bootstrap_DynamicResources{
			LdsConfig: &corepb.ConfigSource{
				ConfigSourceSpecifier: &corepb.ConfigSource_Ads{
					Ads: &corepb.AggregatedConfigSource{},
				},
			},
			CdsConfig: &corepb.ConfigSource{
				ConfigSourceSpecifier: &corepb.ConfigSource_Ads{
					Ads: &corepb.AggregatedConfigSource{},
				},
			},
			AdsConfig: &corepb.ApiConfigSource{
				ApiType: corepb.ApiConfigSource_GRPC,
				GrpcServices: []*corepb.GrpcService{{
					TargetSpecifier: &corepb.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &corepb.GrpcService_EnvoyGrpc{
							ClusterName: "ads_cluster",
						},
					},
				}},
			},
		},

		// Static resource
		StaticResources: &bootstrappb.Bootstrap_StaticResources{
			Clusters: []*v2pb.Cluster{
				{
					Name:           "ads_cluster",
					LbPolicy:       v2pb.Cluster_ROUND_ROBIN,
					ConnectTimeout: adsConnectTimeout,
					ClusterDiscoveryType: &v2pb.Cluster_Type{
						Type: v2pb.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
					LoadAssignment:       ut.CreateLoadAssignment(adsHostname, adsPort),
				},
			},
		},
	}, nil
}
