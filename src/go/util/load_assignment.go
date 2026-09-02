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
	"time"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	Http2KeepaliveInterval = 30 * time.Second
	Http2KeepaliveTimeout  = 10 * time.Second
)

// CreateUpstreamProtocolOptions creates a http2 protocol option as a typed upstream extension.
func CreateUpstreamProtocolOptions(upstreamHttpFilters []*hcmpb.HttpFilter) map[string]*anypb.Any {
	o := &httppb.HttpProtocolOptions{
		UpstreamProtocolOptions: &httppb.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httppb.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httppb.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{
						MaxConcurrentStreams:        &wrapperspb.UInt32Value{Value: 2147483647},
						InitialStreamWindowSize:     &wrapperspb.UInt32Value{Value: 268435456},
						InitialConnectionWindowSize: &wrapperspb.UInt32Value{Value: 268435456},
						ConnectionKeepalive: &corepb.KeepaliveSettings{
							Interval: durationpb.New(Http2KeepaliveInterval),
							Timeout:  durationpb.New(Http2KeepaliveTimeout),
						},
					},
				},
			},
		},
		HttpFilters: upstreamHttpFilters,
	}
	a, _ := anypb.New(o)

	return map[string]*anypb.Any{
		UpstreamProtocolOptions: a,
	}
}

// CreateHttp1UpstreamProtocolOptions creates an HTTP/1 protocol option as a typed upstream extension.
func CreateHttp1UpstreamProtocolOptions(upstreamHttpFilters []*hcmpb.HttpFilter) map[string]*anypb.Any {
	o := &httppb.HttpProtocolOptions{
		UpstreamProtocolOptions: &httppb.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httppb.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httppb.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
					HttpProtocolOptions: &corepb.Http1ProtocolOptions{},
				},
			},
		},
		HttpFilters: upstreamHttpFilters,
	}
	a, _ := anypb.New(o)

	return map[string]*anypb.Any{
		UpstreamProtocolOptions: a,
	}
}

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
