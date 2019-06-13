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

package bootstrap

import (
	"flag"
	"time"

	"github.com/gogo/protobuf/jsonpb"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	boot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"

	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
)

var (
	AdsConnectTimeout = flag.Duration("ads_connect_imeout", 10*time.Second, "ads connect timeout in seconds")
)

// CreateBoostrapConfig outputs envoy bootstrap static config
func CreateBootstrapConfig() string {
	boot := &boot.Bootstrap{
		// Node info
		Node: &core.Node{
			Id:      "api_proxy",
			Cluster: "api_proxy_cluster",
		},

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
					ConnectTimeout: *AdsConnectTimeout,
					ClusterDiscoveryType: &v2.Cluster_Type{
						Type: v2.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &core.Http2ProtocolOptions{},
					LoadAssignment:       ut.CreateLoadAssignment("127.0.0.1", 8790),
				},
			},
		},

		// admin
		Admin: &boot.Admin{
			AccessLogPath: "/dev/null",
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address: "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 8001,
						},
					},
				},
			},
		},
	}

	marshaler := &jsonpb.Marshaler{
		Indent: "  ",
	}
	json_str, _ := marshaler.MarshalToString(boot)
	return json_str
}
