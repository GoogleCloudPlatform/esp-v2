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
	"time"

	bt "cloudesf.googlesource.com/gcpproxy/src/go/bootstrap"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	boot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

// CreateBoostrapConfig outputs envoy bootstrap config for xDS.
func CreateBootstrapConfig(ads_connect_timeout *time.Duration) *boot.Bootstrap {
	return &boot.Bootstrap{
		// Node info
		Node: bt.CreateNode(),

		// admin
		Admin: bt.CreateAdmin(),

		// Dynamic resource
		DynamicResources: &boot.Bootstrap_DynamicResources{
			LdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			CdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			AdsConfig: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
							ClusterName: "ads_cluster",
						},
					},
				}},
			},
		},

		// Static resource
		StaticResources: &boot.Bootstrap_StaticResources{
			Clusters: []v2.Cluster{
				v2.Cluster{
					Name:           "ads_cluster",
					LbPolicy:       v2.Cluster_ROUND_ROBIN,
					ConnectTimeout: *ads_connect_timeout,
					ClusterDiscoveryType: &v2.Cluster_Type{
						Type: v2.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &core.Http2ProtocolOptions{},
					LoadAssignment:       ut.CreateLoadAssignment("127.0.0.1", 8790),
				},
			},
		},
	}
}
