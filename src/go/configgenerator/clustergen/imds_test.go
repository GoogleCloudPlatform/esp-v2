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
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestIMDSClusterFromOPConfig_GenConfig(t *testing.T) {
	testData := []SuccessOPTestCase{
		{
			Desc: "Success with http metadata url and custom options",
			OptsIn: options.ConfigGeneratorOptions{
				ClusterConnectTimeout: time.Second * 36,
				CommonOptions: options.CommonOptions{
					MetadataURL: "http://metadata.server.com",
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "metadata-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 36),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STRICT_DNS,
					},
					LoadAssignment: util.CreateLoadAssignment("metadata.server.com", 80),
				},
			},
		},
		{
			Desc: "Success with https metadata url",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					MetadataURL: "https://metadata.server.com",
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "metadata-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 20),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STRICT_DNS,
					},
					LoadAssignment:  util.CreateLoadAssignment("metadata.server.com", 443),
					TransportSocket: CreateDefaultTLS(t, "metadata.server.com", false),
				},
			},
		},
		{
			Desc: "Success with default options and custom DNS resolver",
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "127.0.0.1:1087",
				CommonOptions: options.CommonOptions{
					MetadataURL: "http://169.254.169.254",
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "metadata-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 20),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STRICT_DNS,
					},
					LoadAssignment: util.CreateLoadAssignment("169.254.169.254", 80),
					DnsResolvers: []*corepb.Address{
						{
							Address: &corepb.Address_SocketAddress{
								SocketAddress: &corepb.SocketAddress{
									Address: "127.0.0.1",
									PortSpecifier: &corepb.SocketAddress_PortValue{
										PortValue: 1087,
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
		tc.RunTest(t, clustergen.NewIMDSClustersFromOPConfig)
	}
}

func TestIMDSClusterFromOPConfig_Disabled(t *testing.T) {
	testData := []SuccessOPTestCase{
		{
			Desc: "Disabled on non-GCP",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					NonGCP: true,
				},
			},
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewIMDSClustersFromOPConfig)
	}
}
