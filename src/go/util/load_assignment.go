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

package util

import (
	addresspb "github.com/envoyproxy/data-plane-api/api/address"
	edspb "github.com/envoyproxy/data-plane-api/api/eds"
	endpointpb "github.com/envoyproxy/data-plane-api/api/endpoint"
)

// CreateLoadAssignment creates a ClusterLoadAssignment
func CreateLoadAssignment(hostname string, port uint32) *edspb.ClusterLoadAssignment {
	return &edspb.ClusterLoadAssignment{
		ClusterName: hostname,
		Endpoints: []*endpointpb.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointpb.LbEndpoint{
					{
						HostIdentifier: &endpointpb.LbEndpoint_Endpoint{
							Endpoint: &endpointpb.Endpoint{
								Address: &addresspb.Address{
									Address: &addresspb.Address_SocketAddress{
										SocketAddress: &addresspb.SocketAddress{
											Address: hostname,
											PortSpecifier: &addresspb.SocketAddress_PortValue{
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
