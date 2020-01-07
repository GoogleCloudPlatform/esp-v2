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

package configgenerator

import (
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"

	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	testProjectName       = "bookstore.endpoints.project123.cloud.goog"
	testApiName           = "endpoints.examples.bookstore.Bookstore"
	testServiceControlEnv = "servicecontrol.googleapis.com"
	testConfigID          = "2019-03-02r0"
)

func createTransportSocket(hostname string) *corepb.TransportSocket {
	transportSocket, _ := util.CreateTransportSocket(hostname, util.DefaultRootCAPaths)
	return transportSocket
}

func TestMakeServiceControlCluster(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantedCluster     v2pb.Cluster
		backendProtocol   string
	}{
		{
			desc: "Success for gRPC backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Control: &confpb.Control{
					Environment: testServiceControlEnv,
				},
			},
			backendProtocol: "grpc",
			wantedCluster: v2pb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &v2pb.Cluster_Type{Type: v2pb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
				LoadAssignment:       util.CreateLoadAssignment(testServiceControlEnv, 443),
				TransportSocket:      createTransportSocket("servicecontrol.googleapis.com"),
			},
		},
		{
			desc: "Success for HTTP1 backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
					},
				},
				Control: &confpb.Control{
					Environment: "http://127.0.0.1:8000",
				},
			},
			backendProtocol: "http1",
			wantedCluster: v2pb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
				LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 8000),
			},
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		cluster, err := makeServiceControlCluster(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !proto.Equal(cluster, &tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeServiceControlCluster\ngot Clusters: %v,\nwant: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}

func TestMakeBackendRoutingCluster(t *testing.T) {
	testData := []struct {
		desc                   string
		fakeServiceConfig      *confpb.Service
		backendDnsLookupFamily string
		backendProtocol        string
		tlsContextSni          string
		wantedClusters         []*v2pb.Cluster
		wantedError            string
	}{
		{
			desc: "Success for HTTP backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
							{
								Name: "Bar",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedClusters: []*v2pb.Cluster{
				{
					Name:                 "mybackend.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:      createTransportSocket("mybackend.com"),
				},
			},
		},
		{
			desc:                   "Succeess, providing correct backend_dns_lookup_family flag",
			backendDnsLookupFamily: "v4only",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog.run.app",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.run.app",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedClusters: []*v2pb.Cluster{
				{
					Name:                 "mybackend.run.app:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &v2pb.Cluster_Type{Type: v2pb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.run.app", 443),
					TransportSocket:      createTransportSocket("mybackend.run.app"),
				},
			},
		},
		{
			desc:                   "Failure, providing incorrect backend_dns_lookup_family flag",
			backendDnsLookupFamily: "v5only",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog.run.app",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.run.app",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedError:     "Invalid DnsLookupFamily: v5only;",
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		if tc.backendDnsLookupFamily != "" {
			opts.BackendDnsLookupFamily = tc.backendDnsLookupFamily
		}
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeBackendRoutingClusters(fakeServiceInfo)
		if err != nil {
			if tc.wantedError == "" || !strings.Contains(err.Error(), tc.wantedError) {
				t.Fatal(err)

			}
		}

		if tc.wantedClusters != nil && !cmp.Equal(clusters, tc.wantedClusters, cmp.Comparer(proto.Equal)) {
			t.Errorf("Test Desc(%d): %s, makeBackendRoutingClusters got: %v, want: %v", i, tc.desc, clusters, tc.wantedClusters)
		}
	}
}

func TestMakeJwtProviderClusters(t *testing.T) {
	testData := []struct {
		desc            string
		fakeProviders   []*confpb.AuthProvider
		backendProtocol string
		wantedClusters  []*v2pb.Cluster
		wantedError     string
	}{
		{
			desc: "Use https jwksUri and http jwksUri",
			fakeProviders: []*confpb.AuthProvider{
				&confpb.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_0",
					JwksUri: "https://metadata.com/pkey",
				},
				&confpb.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_1",
					JwksUri: "http://metadata.com/pkey",
				},
			},
			wantedClusters: []*v2pb.Cluster{
				{
					Name:                 "metadata.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 443),
					TransportSocket:      createTransportSocket("metadata.com"),
				},
				{
					Name:                 "metadata.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 80),
				},
			},
		},
		{
			desc: "Failed with wrong-format jwksUri",
			fakeProviders: []*confpb.AuthProvider{
				&confpb.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_2",
					JwksUri: "%",
				}},
			wantedError: "Fail to parse uri % with error parse https://%: invalid URL escape \"%\"",
		},
		{
			desc: "Deduplicate Auth Provider With Same Host",
			fakeProviders: []*confpb.AuthProvider{
				&confpb.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_0",
					JwksUri: "https://metadata.com/pkey",
				},
				&confpb.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_1",
					JwksUri: "https://metadata.com/pkey",
				},
			},
			wantedClusters: []*v2pb.Cluster{
				{
					Name:                 "metadata.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 443),
					TransportSocket:      createTransportSocket("metadata.com"),
				},
			},
		},
	}
	for i, tc := range testData {
		fakeServiceConfig := &confpb.Service{
			Apis: []*apipb.Api{
				{
					Name: testApiName,
				},
			},
			Authentication: &confpb.Authentication{
				Providers: tc.fakeProviders,
			},
		}

		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = "grpc"
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeJwtProviderClusters(fakeServiceInfo)
		if err != nil && err.Error() != tc.wantedError {
			t.Fatalf("Test Desc(%d): %s, got error:%v, wanted error:%v", i, tc.desc, err, tc.wantedError)
		}

		if !cmp.Equal(clusters, tc.wantedClusters, cmp.Comparer(proto.Equal)) {
			t.Errorf("Test Desc(%d): %s, makeJwtProviderClusters\ngot: %v,\nwant: %v", i, tc.desc, clusters, tc.wantedClusters)
		}

	}
}

func TestMakeIamCluster(t *testing.T) {
	testData := []struct {
		desc              string
		backendProtocol   string
		iamServiceAccount string
		fakeServiceConfig *confpb.Service
		wantedCluster     *v2pb.Cluster
		wantedError       string
	}{
		{
			desc:              "Success, generate iam cluster when iam service acount is set",
			iamServiceAccount: "service-account@google.com",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
					},
				},
			},
			backendProtocol: "grpc",
			wantedCluster: &v2pb.Cluster{
				Name:                 util.IamServerClusterName,
				ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
				DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
				ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_STRICT_DNS},
				LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
				TransportSocket:      createTransportSocket("iamcredentials.googleapis.com"),
			},
		},
		{
			desc: "Success, not generate a iam cluster without iam service acount",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
					},
				},
			},
			backendProtocol: "grpc",
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		opts.IamServiceAccount = tc.iamServiceAccount

		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		cluster, err := makeIamCluster(fakeServiceInfo)
		if err != nil {
			if tc.wantedError == "" || !strings.Contains(err.Error(), tc.wantedError) {
				t.Fatal(err)

			}
		}

		if tc.wantedCluster != nil && !proto.Equal(cluster, tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeBackendRoutingClusters\ngot: %v,\nwant: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}
