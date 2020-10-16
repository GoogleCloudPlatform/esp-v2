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

package util

import (
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

// CreateLoadAssignment creates a cluster for a TCP/IP port.
func CreateLoadAssignment(hostname string, port uint32) *endpointpb.ClusterLoadAssignment {
	return &endpointpb.ClusterLoadAssignment{
		ClusterName: hostname,
		Endpoints: []*endpointpb.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointpb.LbEndpoint{
					{
						HostIdentifier: &endpointpb.LbEndpoint_Endpoint{
							Endpoint: &endpointpb.Endpoint{
								Address: &corepb.Address{
									Address: &corepb.Address_SocketAddress{
										SocketAddress: &corepb.SocketAddress{
											Address: hostname,
											PortSpecifier: &corepb.SocketAddress_PortValue{
												PortValue: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateUdsLoadAssignment creates a cluster for a unix domain socket.
func CreateUdsLoadAssignment(clusterName string) *endpointpb.ClusterLoadAssignment {
	return &endpointpb.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpointpb.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointpb.LbEndpoint{
					{
						HostIdentifier: &endpointpb.LbEndpoint_Endpoint{
							Endpoint: &endpointpb.Endpoint{
								Address: &corepb.Address{
									Address: &corepb.Address_Pipe{
										Pipe: &corepb.Pipe{
											Path: clusterName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
