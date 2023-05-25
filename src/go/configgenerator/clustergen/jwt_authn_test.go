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
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNewJWTProviderClustersFromOPConfig_GenConfig(t *testing.T) {
	testData := []clustergentest.SuccessOPTestCase{
		{
			Desc: "Use https jwksUri and http jwksUri",
			ServiceConfigIn: &confpb.Service{
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider_0",
							Issuer:  "issuer_0",
							JwksUri: "https://metadata.com/pkey",
						},
						{
							Id:      "auth_provider_1",
							Issuer:  "issuer_1",
							JwksUri: "http://metadata.com/pkey",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "jwt-provider-cluster-metadata.com:443",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "metadata.com", false),
				},
				{
					Name:                 "jwt-provider-cluster-metadata.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 80),
				},
			},
		},
		{
			Desc: "De-deduplicate auth provider with same host",
			ServiceConfigIn: &confpb.Service{
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider_0",
							Issuer:  "issuer_0",
							JwksUri: "https://metadata.com/pkey",
						},
						{
							Id:      "auth_provider_1",
							Issuer:  "issuer_1",
							JwksUri: "https://metadata.com/pkey",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "jwt-provider-cluster-metadata.com:443",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "metadata.com", false),
				},
			},
		},
		{
			Desc: "Use custom DNS resolver",
			ServiceConfigIn: &confpb.Service{
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider_0",
							Issuer:  "issuer_0",
							JwksUri: "http://metadata.com/pkey",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "8.8.8.8",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "jwt-provider-cluster-metadata.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 80),
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
		tc.RunTest(t, clustergen.NewJWTProviderClustersFromOPConfig)
	}
}

func TestNewJWTProviderClustersFromOPConfig_BadInputFactory(t *testing.T) {
	testData := []clustergentest.FactoryErrorOPTestCase{
		{
			Desc: "Could not parse JWKS URI",
			ServiceConfigIn: &confpb.Service{
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider_0",
							Issuer:  "issuer_0",
							JwksUri: "https://invalid^url:googleapis:com/test",
						},
					},
				},
			},
			WantFactoryError: "Fail to parse uri",
		},
		{
			Desc: "OICD is disabled but needed",
			ServiceConfigIn: &confpb.Service{
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider_0",
							Issuer: "issuer_0",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				DisableOidcDiscovery: true,
			},
			WantFactoryError: "error processing authentication provider",
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewJWTProviderClustersFromOPConfig)
	}
}
