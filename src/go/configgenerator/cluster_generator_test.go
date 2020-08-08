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

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
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
	transportSocket, _ := util.CreateUpstreamTransportSocket(hostname, util.DefaultRootCAPaths, "", nil)
	return transportSocket
}

func createH2TransportSocket(hostname string) *corepb.TransportSocket {
	transportSocket, _ := util.CreateUpstreamTransportSocket(hostname, util.DefaultRootCAPaths, "", []string{"h2"})
	return transportSocket
}

func TestMakeServiceControlCluster(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantedCluster     clusterpb.Cluster
		BackendAddress    string
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
			BackendAddress: "grpc://127.0.0.1:80",
			wantedCluster: clusterpb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
				LoadAssignment:       util.CreateLoadAssignment(testServiceControlEnv, 443),
				TransportSocket:      createTransportSocket("servicecontrol.googleapis.com"),
			},
		},
		{
			desc: "Success for http backend",
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
			BackendAddress: "http://127.0.0.1:80",
			wantedCluster: clusterpb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
				LoadAssignment:       util.CreateLoadAssignment("127.0.0.1", 8000),
			},
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
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
		BackendAddress         string
		tlsContextSni          string
		wantedClusters         []*clusterpb.Cluster
		wantedError            string
	}{
		{
			desc: "Success for HTTPS backend",
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
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:      createTransportSocket("mybackend.com"),
				},
			},
		},
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
							Address:         "http://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 80),
				},
			},
		},
		{
			desc: "Success for mixed http, https backends",
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
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend_http.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_http.com", 80),
				},
				{
					Name:                 "mybackend_https.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_https.com", 443),
					TransportSocket:      createTransportSocket("mybackend_https.com"),
				},
			},
		},
		{
			desc: "Success for grpcs backend",
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
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 443),
					TransportSocket:      createH2TransportSocket("mybackend.com"),
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
				},
			},
		},
		{
			desc: "Success for grpc backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
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
							Address:  "grpc://mybackend.com",
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend.com", 80),
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
				},
			},
		},
		{
			desc: "Success for mixed grpc and grpcs backends",
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
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend_http.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_http.com", 80),
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
				},
				{
					Name:                 "mybackend_https.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					LoadAssignment:       util.CreateLoadAssignment("mybackend_https.com", 443),
					TransportSocket:      createH2TransportSocket("mybackend_https.com"),
					Http2ProtocolOptions: &corepb.Http2ProtocolOptions{},
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
			BackendAddress: "http://127.0.0.1:80",
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "mybackend.run.app:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
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
			BackendAddress: "http://127.0.0.1:80",
			wantedError:    "Invalid DnsLookupFamily: v5only;",
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
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
		wantedClusters  []*clusterpb.Cluster
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
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "metadata.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
					LoadAssignment:       util.CreateLoadAssignment("metadata.com", 443),
					TransportSocket:      createTransportSocket("metadata.com"),
				},
				{
					Name:                 "metadata.com:80",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
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
			wantedError: "Fail to parse uri %",
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
			wantedClusters: []*clusterpb.Cluster{
				{
					Name:                 "metadata.com:443",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
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
		opts.BackendAddress = "grpc://127.0.0.1:80"
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeJwtProviderClusters(fakeServiceInfo)
		if err != nil && !strings.Contains(err.Error(), tc.wantedError) {
			t.Fatalf("Test Desc(%d): %s, got error:%v, wanted error:%v", i, tc.desc, err, tc.wantedError)
		}

		if !cmp.Equal(clusters, tc.wantedClusters, cmp.Comparer(proto.Equal)) {
			t.Errorf("Test Desc(%d): %s, makeJwtProviderClusters\ngot: %v,\nwant: %v", i, tc.desc, clusters, tc.wantedClusters)
		}

	}
}

func TestMakeIamCluster(t *testing.T) {
	testData := []struct {
		desc                        string
		BackendAddress              string
		backendAuthIamCredential    *options.IAMCredentialsOptions
		serviceControlIamCredential *options.IAMCredentialsOptions
		fakeServiceConfig           *confpb.Service
		wantedCluster               *clusterpb.Cluster
		wantedError                 string
	}{
		{
			desc: "Success, generate iam cluster when backendAuthIamCredential is set",
			backendAuthIamCredential: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "service-account@google.com",
				Delegates:           nil,
			},
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantedCluster: &clusterpb.Cluster{
				Name:                 util.IamServerClusterName,
				ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
				DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
				ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_STRICT_DNS},
				LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
				TransportSocket:      createTransportSocket("iamcredentials.googleapis.com"),
			},
		},
		{
			desc: "Success, generate iam cluster when serviceControlIamCredential is set",
			serviceControlIamCredential: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "service-account@google.com",
				Delegates:           nil,
			},
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantedCluster: &clusterpb.Cluster{
				Name:                 util.IamServerClusterName,
				ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
				DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
				ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_STRICT_DNS},
				LoadAssignment:       util.CreateLoadAssignment("iamcredentials.googleapis.com", 443),
				TransportSocket:      createTransportSocket("iamcredentials.googleapis.com"),
			},
		},
		{
			desc: "Success, not generate a iam cluster without any iam service credential",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantedCluster:  nil,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
		opts.BackendAuthCredentials = tc.backendAuthIamCredential
		opts.ServiceControlCredentials = tc.serviceControlIamCredential

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

		if !proto.Equal(cluster, tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeBackendRoutingClusters\ngot: %v,\nwant: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}

func TestMakeTokenAgentCluster(t *testing.T) {
	fakeServiceInfo, _ := configinfo.NewServiceInfoFromServiceConfig(&confpb.Service{
		Apis: []*apipb.Api{
			{
				Name: testApiName,
			},
		},
	}, testConfigID, options.DefaultConfigGeneratorOptions())

	cluster := makeTokenAgentCluster(fakeServiceInfo)
	wantCluster := &clusterpb.Cluster{
		Name:           util.TokenAgentClusterName,
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: ptypes.DurationProto(fakeServiceInfo.Options.ClusterConnectTimeout),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STATIC,
		},
		LoadAssignment: util.CreateLoadAssignment("127.0.0.1", uint32(fakeServiceInfo.Options.TokenAgentPort)),
	}

	if !proto.Equal(cluster, wantCluster) {
		t.Errorf("Test makeTokenAgentClusters, \ngot: %v,\nwant: %v", cluster, wantCluster)
	}
}
