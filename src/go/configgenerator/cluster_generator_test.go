// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"
	"google.golang.org/genproto/protobuf/api"

	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	cdspb "github.com/envoyproxy/data-plane-api/api/cds"
	certpb "github.com/envoyproxy/data-plane-api/api/cert"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	testProjectName       = "bookstore.endpoints.project123.cloud.goog"
	testApiName           = "endpoints.examples.bookstore.Bookstore"
	testServiceControlEnv = "servicecontrol.googleapis.com"
	testConfigID          = "2019-03-02r0"
)

func TestMakeServiceControlCluster(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *conf.Service
		wantedCluster     cdspb.Cluster
		backendProtocol   string
	}{
		{
			desc: "Success for gRPC backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Control: &conf.Control{
					Environment: testServiceControlEnv,
				},
			},
			backendProtocol: "grpc",
			wantedCluster: cdspb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &cdspb.Cluster_Type{Type: cdspb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
				LoadAssignment:       ut.CreateLoadAssignment(testServiceControlEnv, 443),
				TlsContext: &certpb.UpstreamTlsContext{
					Sni: "servicecontrol.googleapis.com",
				},
			},
		},
		{
			desc: "Success for HTTP1 backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Control: &conf.Control{
					Environment: "http://127.0.0.1:8000",
				},
			},
			backendProtocol: "http1",
			wantedCluster: cdspb.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
				ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
				LoadAssignment:       ut.CreateLoadAssignment("127.0.0.1", 8000),
			},
		},
	}

	for i, tc := range testData {
		options := configinfo.DefaultEnvoyConfigOptions()
		options.BackendProtocol = tc.backendProtocol
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, options)
		if err != nil {
			t.Fatal(err)
		}

		cluster, err := makeServiceControlCluster(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cluster, &tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeServiceControlCluster\ngot Clusters: %v,\nwant: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}

func TestMakeBackendRoutingCluster(t *testing.T) {
	testData := []struct {
		desc                   string
		fakeServiceConfig      *conf.Service
		backendDnsLookupFamily string
		backendProtocol        string
		wantedClusters         []*cdspb.Cluster
		wantedError            string
	}{
		{
			desc: "Success for HTTP backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
							{
								Name: "Bar",
							},
						},
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 "DynamicRouting_0",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					LoadAssignment:       ut.CreateLoadAssignment("mybackend.com", 443),
					TlsContext: &certpb.UpstreamTlsContext{
						Sni: "mybackend.com",
					},
				},
			},
		},
		{
			desc:                   "Succeess, providing correct backend_dns_lookup_family flag",
			backendDnsLookupFamily: "v4only",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog.run.app",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.run.app",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 "DynamicRouting_0",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					ClusterDiscoveryType: &cdspb.Cluster_Type{Type: cdspb.Cluster_LOGICAL_DNS},
					LoadAssignment:       ut.CreateLoadAssignment("mybackend.run.app", 443),
					TlsContext: &certpb.UpstreamTlsContext{
						Sni: "mybackend.run.app",
					},
				},
			},
		},
		{
			desc:                   "Failure, providing incorrect backend_dns_lookup_family flag",
			backendDnsLookupFamily: "v5only",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog.run.app",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
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
		options := configinfo.DefaultEnvoyConfigOptions()
		options.BackendProtocol = tc.backendProtocol
		options.EnableBackendRouting = true
		if tc.backendDnsLookupFamily != "" {
			options.BackendDnsLookupFamily = tc.backendDnsLookupFamily
		}
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, options)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeBackendRoutingClusters(fakeServiceInfo)
		if err != nil {
			if tc.wantedError == "" || !strings.Contains(err.Error(), tc.wantedError) {
				t.Fatal(err)

			}
		}

		if tc.wantedClusters != nil && !reflect.DeepEqual(clusters, tc.wantedClusters) {
			t.Errorf("Test Desc(%d): %s, makeBackendRoutingClusters got: %v, want: %v", i, tc.desc, clusters, tc.wantedClusters)
		}
	}
}

func TestMakeJwtProviderClusters(t *testing.T) {
	_, fakeJwksUriHost, _, _, _ := ut.ParseURI(ut.FakeJwksUri)

	r := mux.NewRouter()
	jwksUriEntry, _ := json.Marshal(map[string]string{"jwks_uri": "this-is-jwksUri"})
	r.Path("/.well-known/openid-configuration/").Methods("GET").Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(jwksUriEntry)
		}))
	openIDServer := httptest.NewServer(r)

	testData := []struct {
		desc           string
		fakeProviders  []*conf.AuthProvider
		wantedClusters []*cdspb.Cluster
	}{
		{
			desc: "Use https jwksUri and http jwksUri",
			fakeProviders: []*conf.AuthProvider{
				&conf.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_0",
					JwksUri: "https://metadata.com/pkey",
				},
				&conf.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_1",
					JwksUri: "http://metadata.com/pkey",
				},
			},
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 "issuer_0",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment("metadata.com", 443),
					TlsContext: &certpb.UpstreamTlsContext{
						Sni: "metadata.com",
					},
				},
				{
					Name:                 "issuer_1",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment("metadata.com", 80),
				},
			},
		},
		{
			desc: "With wrong-format jwksUri, use FakeJwksUri",
			fakeProviders: []*conf.AuthProvider{
				&conf.AuthProvider{
					Id:      "auth_provider",
					Issuer:  "issuer_2",
					JwksUri: "%",
				}},
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 "issuer_2",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment(fakeJwksUriHost, 80),
				},
			},
		},
		{
			desc: "Empty jwksUri, use jwksUri acquired by openID",
			fakeProviders: []*conf.AuthProvider{
				&conf.AuthProvider{
					Id:     "auth_provider",
					Issuer: openIDServer.URL,
				},
			},
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 openIDServer.URL,
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment("this-is-jwksUri", 443),
					TlsContext: &certpb.UpstreamTlsContext{
						Sni: "this-is-jwksUri",
					},
				},
			},
		},
		{
			desc: "Empty jwksUri and no jwksUri acquired by openID, use FakeJwksUri",
			fakeProviders: []*conf.AuthProvider{
				&conf.AuthProvider{
					Id:     "auth_provider",
					Issuer: "aaaaa.bbbbbb.ccccc/inaccessible_uri/",
				},
			},
			wantedClusters: []*cdspb.Cluster{
				{
					Name:                 "aaaaa.bbbbbb.ccccc/inaccessible_uri/",
					ConnectTimeout:       ptypes.DurationProto(20 * time.Second),
					ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment(fakeJwksUriHost, 80),
				},
			},
		},
	}
	for i, tc := range testData {
		fakeServiceConfig := &conf.Service{
			Apis: []*api.Api{
				{
					Name: testApiName,
				},
			},
			Authentication: &conf.Authentication{
				Providers: tc.fakeProviders,
			},
		}

		options := configinfo.DefaultEnvoyConfigOptions()
		options.BackendProtocol = "http2"
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, options)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeJwtProviderClusters(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(clusters, tc.wantedClusters) {
			t.Errorf("Test Desc(%d): %s, makeJwtProviderClusters\ngot: %v,\nwant: %v", i, tc.desc, clusters, tc.wantedClusters)
		}

	}
}
