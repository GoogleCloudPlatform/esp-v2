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
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"

	gen "cloudesf.googlesource.com/gcpproxy/src/go/configgenerator"
	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	boot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// CreateBoostrapConfig outputs envoy bootstrap config for xDS.
func CreateBootstrapConfig(ads_connect_timeout *time.Duration) string {
	boot := &boot.Bootstrap{
		// Node info
		Node: createNode(),

		// admin
		Admin: createAdmin(),

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

	marshaler := &jsonpb.Marshaler{
		Indent: "  ",
	}
	json_str, _ := marshaler.MarshalToString(boot)
	return json_str
}

// ServiceToBoostrapConfig outputs envoy bootstrap config from service config.
// id is the service configuration ID. It is generated when deploying
// service config to ServiceManagement Server, example: 2017-02-13r0.
func ServiceToBoostrapConfig(serviceConfig *conf.Service, id string) (*boot.Bootstrap, error) {
	bootstrap := &boot.Bootstrap{
		Node:  createNode(),
		Admin: createAdmin(),
	}

	serviceInfo, err := sc.NewServiceInfoFromServiceConfig(serviceConfig, id)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	listener, err := gen.MakeListeners(serviceInfo)
	if err != nil {
		return nil, err
	}
	clusters, err := gen.MakeClusters(serviceInfo)
	if err != nil {
		return nil, err
	}

	bootstrap.StaticResources = &boot.Bootstrap_StaticResources{
		Listeners: []v2.Listener{*listener},
		Clusters:  clusters,
	}
	return bootstrap, nil
}

func createNode() *core.Node {
	return &core.Node{
		Id:      "api_proxy",
		Cluster: "api_proxy_cluster",
	}
}

func createAdmin() *boot.Admin {
	return &boot.Admin{
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
	}
}
