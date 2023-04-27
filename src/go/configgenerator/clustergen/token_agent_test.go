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

func TestNewTokenAgentClusterFromOPConfig_GenConfig(t *testing.T) {
	testData := []SuccessOPTestCase{
		{
			Desc: "Success with default options on NonGCP",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					NonGCP: true,
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "token-agent-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 20),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STATIC,
					},
					LoadAssignment: util.CreateLoadAssignment("127.0.0.1", 8791),
				},
			},
		},
		{
			Desc: "Success with custom options on GCP with service control key",
			OptsIn: options.ConfigGeneratorOptions{
				ClusterConnectTimeout: time.Second * 36,
				TokenAgentPort:        9203,
				ServiceAccountKey:     "/path/to/key.json",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "token-agent-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 36),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STATIC,
					},
					LoadAssignment: util.CreateLoadAssignment("127.0.0.1", 9203),
				},
			},
		},
		{
			Desc: "Success with default options on NonGCP and custom DNS resolver",
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "127.0.0.1:1087",
				CommonOptions: options.CommonOptions{
					NonGCP: true,
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:           "token-agent-cluster",
					LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
					ConnectTimeout: durationpb.New(time.Second * 20),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{
						Type: clusterpb.Cluster_STATIC,
					},
					LoadAssignment: util.CreateLoadAssignment("127.0.0.1", 8791),
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
		tc.RunTest(t, clustergen.NewTokenAgentClustersFromOPConfig)
	}
}

func TestNewTokenAgentClusterFromOPConfig_Disabled(t *testing.T) {
	testData := []DisabledOPTestCase{
		{
			Desc: "Disabled on GCP",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					NonGCP: false,
				},
			},
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewTokenAgentClustersFromOPConfig)
	}
}
