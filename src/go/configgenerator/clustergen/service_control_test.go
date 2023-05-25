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
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNewServiceControlClusterFromOPConfig_GenConfig(t *testing.T) {
	testData := []clustergentest.SuccessOPTestCase{
		{
			Desc: "Success with http address",
			OptsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "http://127.0.0.1:8912",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "service-control-cluster",
					ConnectTimeout:       durationpb.New(5 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 8912),
				},
			},
		},
		{
			Desc: "Success with https address",
			OptsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "https://servicecontrol.googleapis.com",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "service-control-cluster",
					ConnectTimeout:       durationpb.New(5 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("servicecontrol.googleapis.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "servicecontrol.googleapis.com", false),
				},
			},
		},
		{
			Desc: "Success for custom DNS resolver",
			OptsIn: options.ConfigGeneratorOptions{
				ServiceControlURL:    "https://servicecontrol.googleapis.com",
				DnsResolverAddresses: "8.8.8.8",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "service-control-cluster",
					ConnectTimeout:       durationpb.New(5 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("servicecontrol.googleapis.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "servicecontrol.googleapis.com", false),
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
		tc.RunTest(t, clustergen.NewServiceControlClustersFromOPConfig)
	}
}

func TestNewServiceControlClusterFromOPConfig_BadInputFactory(t *testing.T) {
	testData := []clustergentest.FactoryErrorOPTestCase{
		{
			Desc: "Could not parse Service Control URL",
			OptsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "https://invalid^url:googleapis/com",
			},
			WantFactoryError: "failed to parse uri",
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewServiceControlClustersFromOPConfig)
	}
}
