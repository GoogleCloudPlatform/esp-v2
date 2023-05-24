// Copyright 2023 Google LLC
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

package clustergen_test

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/clustergentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewLocalBackendClusterFromOPConfig_GenConfig(t *testing.T) {
	testData := []clustergentest.SuccessOPTestCase{
		{
			Desc: "Success for OpenAPI HTTP backend",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.1:80",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for OpenAPI HTTP backend with a backend cluster max requests",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:            "http://127.0.0.1:80",
				BackendClusterMaxRequests: 10240,
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
					CircuitBreakers: &clusterpb.CircuitBreakers{
						Thresholds: []*clusterpb.CircuitBreakers_Thresholds{
							{
								Priority:    corepb.RoutingPriority_DEFAULT,
								MaxRequests: &wrappers.UInt32Value{Value: 10240},
							},
							{
								Priority:    corepb.RoutingPriority_HIGH,
								MaxRequests: &wrappers.UInt32Value{Value: 10240},
							},
						},
					},
				},
			},
		},
		{
			Desc: "Success for OpenAPI HTTPS backend",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "https://mybackend.com:443",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "mybackend.com", false),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpc backend",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("127.0.0.1", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpcs backend",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpcs://mybackend.com:443",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:               clustergentest.CreateDefaultTLS(t, "mybackend.com", true),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpc backend with default grpc health check settings",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:         "grpc://127.0.0.1:80",
				HealthCheckGrpcBackend: true,
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("127.0.0.1", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					HealthChecks: []*corepb.HealthCheck{
						{
							Timeout:            durationpb.New(1 * time.Second),
							Interval:           durationpb.New(1 * time.Second),
							NoTrafficInterval:  durationpb.New(60 * time.Second),
							UnhealthyThreshold: &wrappers.UInt32Value{Value: 3},
							HealthyThreshold:   &wrappers.UInt32Value{Value: 3},
							HealthChecker: &corepb.HealthCheck_GrpcHealthCheck_{
								GrpcHealthCheck: &corepb.HealthCheck_GrpcHealthCheck{},
							},
						},
					},
					DnsLookupFamily: clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpc backend + grpc health check with custom interval and service",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:                          "grpcs://mybackend.com:443",
				HealthCheckGrpcBackend:                  true,
				HealthCheckGrpcBackendInterval:          10 * time.Second,
				HealthCheckGrpcBackendNoTrafficInterval: 30 * time.Second,
				HealthCheckGrpcBackendService:           "foo.bar.service",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:               clustergentest.CreateDefaultTLS(t, "mybackend.com", true),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					HealthChecks: []*corepb.HealthCheck{
						{
							Timeout:            durationpb.New(10 * time.Second),
							Interval:           durationpb.New(10 * time.Second),
							NoTrafficInterval:  durationpb.New(30 * time.Second),
							UnhealthyThreshold: &wrappers.UInt32Value{Value: 3},
							HealthyThreshold:   &wrappers.UInt32Value{Value: 3},
							HealthChecker: &corepb.HealthCheck_GrpcHealthCheck_{
								GrpcHealthCheck: &corepb.HealthCheck_GrpcHealthCheck{
									ServiceName: "foo.bar.service",
								},
							},
						},
					},
					DnsLookupFamily: clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for custom DNS resolver",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:       "http://127.0.0.1:80",
				DnsResolverAddresses: "8.8.8.8",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
					DnsResolvers: []*corepb.Address{
						{
							Address: &corepb.Address_SocketAddress{
								SocketAddress: &corepb.SocketAddress{
									Address: "8.8.8.8",
									PortSpecifier: &corepb.SocketAddress_PortValue{
										PortValue: 53,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewLocalBackendClustersFromOPConfig)
	}
}

func TestNewLocalBackendClusterFromOPConfig_BadInputFactory(t *testing.T) {
	testData := []clustergentest.FactoryErrorOPTestCase{
		{
			Desc: "HealthCheckGrpcBackend but backend protocol not grpc",
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:         "http://127.0.0.1:80",
				HealthCheckGrpcBackend: true,
			},
			WantFactoryError: "--health_check_grpc_backend",
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewLocalBackendClustersFromOPConfig)
	}
}
