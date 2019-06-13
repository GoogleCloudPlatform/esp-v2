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
	"flag"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/gorilla/mux"
	"google.golang.org/genproto/protobuf/api"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
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
		wantedCluster     *v2.Cluster
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
			wantedCluster: &v2.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       5 * time.Second,
				ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      v2.Cluster_V4_ONLY,
				LoadAssignment:       ut.CreateLoadAssignment(testServiceControlEnv, 443),
				TlsContext: &auth.UpstreamTlsContext{
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
			wantedCluster: &v2.Cluster{
				Name:                 "service-control-cluster",
				ConnectTimeout:       5 * time.Second,
				ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
				DnsLookupFamily:      v2.Cluster_V4_ONLY,
				LoadAssignment:       ut.CreateLoadAssignment("127.0.0.1", 8000),
			},
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", tc.backendProtocol)
		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}

		cluster, err := makeServiceControlCluster(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cluster, tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeServiceControlCluster\ngot Clusters: %v,\nwant: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}

func TestMakeBackendRoutingCluster(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *conf.Service
		wantedClusters    []cache.Resource
		backendProtocol   string
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
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 "DynamicRouting_0",
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					LoadAssignment:       ut.CreateLoadAssignment("mybackend.com", 443),
					TlsContext: &auth.UpstreamTlsContext{
						Sni: "mybackend.com",
					},
				},
			},
		},
		{
			desc: "Success for Cloud Run backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog.run.app",
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
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.run.app",
							},
						},
						{
							Address:         "https://mybackend.run.app",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.run.app",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 "DynamicRouting_0",
					ConnectTimeout:       20 * time.Second,
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					LoadAssignment:       ut.CreateLoadAssignment("mybackend.run.app", 443),
					TlsContext: &auth.UpstreamTlsContext{
						Sni: "mybackend.run.app",
					},
				},
			},
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", tc.backendProtocol)
		flag.Set("enable_backend_routing", "true")
		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}

		clusters, err := makeBackendRoutingClusters(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(clusters, tc.wantedClusters) {
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
		wantedClusters []cache.Resource
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
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 "issuer_0",
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment("metadata.com", 443),
					TlsContext: &auth.UpstreamTlsContext{
						Sni: "metadata.com",
					},
				},
				&v2.Cluster{
					Name:                 "issuer_1",
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
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
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 "issuer_2",
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
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
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 openIDServer.URL,
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
					LoadAssignment:       ut.CreateLoadAssignment("this-is-jwksUri", 443),
					TlsContext: &auth.UpstreamTlsContext{
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
			wantedClusters: []cache.Resource{
				&v2.Cluster{
					Name:                 "aaaaa.bbbbbb.ccccc/inaccessible_uri/",
					ConnectTimeout:       20 * time.Second,
					ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
					DnsLookupFamily:      v2.Cluster_V4_ONLY,
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

		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID)
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
