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

func TestNewRemoteBackendClustersFromOPConfig_GenConfig(t *testing.T) {
	testData := []clustergentest.SuccessOPTestCase{
		{
			Desc: "Success for HTTPS backend rules, de-duplicated",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "https://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
						{
							Address:  "https://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-mybackend.com:443",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "mybackend.com", false),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for single HTTP backend rule",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "http://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-mybackend.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for mixed http, https backends",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "http://mybackend_http.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
						{
							Address:  "https://mybackend_https.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-mybackend_http.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_http.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                 "backend-cluster-mybackend_https.com:443",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_https.com", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "mybackend_https.com", false),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpcs backend, de-duplicated",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpcs://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
						{
							Address:  "grpcs://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend.com:443",
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
			Desc: "Success for grpc backend",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for mixed grpc and grpcs backends",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend_http.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
						{
							Address:  "grpcs://mybackend_https.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend_http.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend_http.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                          "backend-cluster-mybackend_https.com:443",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend_https.com", 443),
					TransportSocket:               clustergentest.CreateDefaultTLS(t, "mybackend_https.com", true),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success, providing correct backend_dns_lookup_family flag",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "https://mybackend.run.app",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendDnsLookupFamily: "v4only",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-mybackend.run.app:443",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.run.app", 443),
					TransportSocket:      clustergentest.CreateDefaultTLS(t, "mybackend.run.app", false),
				},
			},
		},
		{
			Desc: "Success for single HTTP backend rule with custom DNS",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "http://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				DnsResolverAddresses: "8.8.8.8",
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                 "backend-cluster-mybackend.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 80),
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
		{
			Desc: "Success for grpc backend with non-OpenAPI HTTP backend rule",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                 "backend-cluster-http.abc.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("http.abc.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Success for grpc backend with non-OpenAPI HTTP backend rule, with de-duplication of inner rule",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend-1.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
						{
							Address:  "grpc://mybackend-2.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend-1.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend-1.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                 "backend-cluster-http.abc.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("http.abc.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                          "backend-cluster-mybackend-2.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend-2.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "Backend rule is skipped when it doesn't have address",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend-1.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									// Missing address
								},
							},
						},
						{
							// Missing address
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend-1.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend-1.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
		{
			Desc: "All backend rules are skipped when BackendAddressOverride is enabled",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend-1.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
						{
							Address:  "http://mybackend-2.com",
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				EnableBackendAddressOverride: true,
			},
			WantClusters: nil,
		},
		{
			Desc: "Discovery APIs are skipped",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend-1.com",
							Selector: "google.discovery.GetDiscoveryRest",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
						{
							Address:  "http://mybackend-2.com",
							Selector: "google.discovery.GetDiscovery",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				AllowDiscoveryAPIs: false,
			},
			WantClusters: nil,
		},
		{
			Desc: "Discovery APIs are disallowed",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://mybackend-1.com",
							Selector: "google.discovery.GetDiscoveryRest",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "http://http.abc.com/api/",
								},
							},
						},
						{
							Address:  "http://mybackend-2.com",
							Selector: "google.discovery.GetDiscovery",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				AllowDiscoveryAPIs: true,
			},
			WantClusters: []*clusterpb.Cluster{
				{
					Name:                          "backend-cluster-mybackend-1.com:80",
					ConnectTimeout:                durationpb.New(20 * time.Second),
					ClusterDiscoveryType:          &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:                util.CreateLoadAssignment("mybackend-1.com", 80),
					TypedExtensionProtocolOptions: util.CreateUpstreamProtocolOptions(),
					DnsLookupFamily:               clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                 "backend-cluster-http.abc.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("http.abc.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
				{
					Name:                 "backend-cluster-mybackend-2.com:80",
					ConnectTimeout:       durationpb.New(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend-2.com", 80),
					DnsLookupFamily:      clusterpb.Cluster_V4_PREFERRED,
				},
			},
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewRemoteBackendClustersFromOPConfig)
	}
}

func TestNewRemoteBackendClustersFromOPConfig_BadInputFactory(t *testing.T) {
	testData := []clustergentest.FactoryErrorOPTestCase{
		{
			Desc: "Could not parse backend url",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "http://some^invalid:url:googleapis/com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			WantFactoryError: "error parsing remote backend rule's address",
		},
		{
			Desc: "non-OpenAPI HTTP backend rule can NOT be grpc address",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "https://test.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Address: "grpc://abc.com/api/",
								},
							},
						},
					},
				},
			},
			WantFactoryError: "gRPC protocol conflicted with http backend",
		},
	}

	for _, tc := range testData {
		tc.RunTest(t, clustergen.NewRemoteBackendClustersFromOPConfig)
	}
}
