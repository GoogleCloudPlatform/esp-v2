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
	"reflect"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"google.golang.org/genproto/protobuf/api"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
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
				Name:            "service-control-cluster",
				ConnectTimeout:  5 * time.Second,
				Type:            v2.Cluster_LOGICAL_DNS,
				DnsLookupFamily: v2.Cluster_V4_ONLY,
				Hosts: []*core.Address{
					{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Address: testServiceControlEnv,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: 443,
								},
							},
						},
					},
				},
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
				Name:            "service-control-cluster",
				ConnectTimeout:  5 * time.Second,
				Type:            v2.Cluster_LOGICAL_DNS,
				DnsLookupFamily: v2.Cluster_V4_ONLY,
				Hosts: []*core.Address{
					{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Address: "127.0.0.1",
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: 8000,
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testData {
		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}

		cluster, err := makeServiceControlCluster(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cluster, tc.wantedCluster) {
			t.Errorf("Test Desc(%d): %s, makeServiceControlCluster got Clusters: %v, want: %v", i, tc.desc, cluster, tc.wantedCluster)
		}
	}
}
