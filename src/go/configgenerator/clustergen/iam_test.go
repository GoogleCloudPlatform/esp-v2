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

func TestNewIAMClusterFromOPConfig_GenConfig(t *testing.T) {
	testData := []SuccessOPTestCase{
		{
			Desc: "Success, generate iam cluster when backendAuthIamCredential is set",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					BackendAuthCredentials: &options.IAMCredentialsOptions{
						ServiceAccountEmail: "service-account@google.com",
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "iam-cluster",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_STRICT_DNS},
					LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
					TransportSocket:      CreateDefaultTLS(t, "iamcredentials.googleapis.com", false),
				},
			},
		},
		{
			Desc: "Success, generate iam cluster when serviceControlIamCredential is set",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					ServiceControlCredentials: &options.IAMCredentialsOptions{
						ServiceAccountEmail: "service-account@google.com",
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "iam-cluster",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_STRICT_DNS},
					LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
					TransportSocket:      CreateDefaultTLS(t, "iamcredentials.googleapis.com", false),
				},
			},
		},
		{
			Desc: "Success, generate iam cluster with custom http IAM URL",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					IamURL: "http://insecure-iam.googleapis.com:8080",
					BackendAuthCredentials: &options.IAMCredentialsOptions{
						ServiceAccountEmail: "service-account@google.com",
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "iam-cluster",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_STRICT_DNS},
					LoadAssignment:       util.CreateLoadAssignment("insecure-iam.googleapis.com", 8080),
				},
			},
		},
		{
			Desc: "Success, generate iam cluster with custom DNS resolver",
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "127.0.0.1:1087",
				CommonOptions: options.CommonOptions{
					ServiceControlCredentials: &options.IAMCredentialsOptions{
						ServiceAccountEmail: "service-account@google.com",
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "iam-cluster",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_STRICT_DNS},
					LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
					TransportSocket:      CreateDefaultTLS(t, "iamcredentials.googleapis.com", false),
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
		tc.RunTest(t, clustergen.NewIAMClustersFromOPConfig)
	}
}

func TestNewIAMClusterFromOPConfig_Disabled(t *testing.T) {
	testData := []SuccessOPTestCase{
		{
			Desc: "Disabled when no Backend Auth or SC creds are provided",
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "127.0.0.1:1087",
				CommonOptions: options.CommonOptions{
					BackendAuthCredentials:    nil,
					ServiceControlCredentials: nil,
				},
			},
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewIAMClustersFromOPConfig)
	}
}
