// Copyright 2018 Google Cloud Platform Proxy Authors
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

package configmanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/gogo/protobuf/jsonpb"
)

const (
	testProjectName  = "bookstore.endpoints.project123.cloud.goog"
	testEndpointName = "endpoints.examples.bookstore.Bookstore"
	testConfigID     = "2017-05-01r0"
	fakeNodeID       = "id"
)

var (
	fakeConfig = fmt.Sprintf(`
	{
		"name":"%s",
		"title":"Bookstore gRPC API",
		"apis":[
			{
				"name":"%s",
				"version":"v1",
				"syntax":"SYNTAX_PROTO3"
			}
		],
		"sourceInfo":{
			"sourceFiles":[
				{
					"@type":"type.googleapis.com/google.api.servicemanagement.v1.ConfigFile",
					"filePath":"api_config.yaml",
					"fileContents":"%s",
					"fileType":"SERVICE_CONFIG_YAML"
				},
				{
					"@type":"type.googleapis.com/google.api.servicemanagement.v1.ConfigFile",
					"filePath":"api_descriptor.pb",
					"fileContents":"%s",
					"fileType":"FILE_DESCRIPTOR_SET_PROTO"
				}
			]
		}
	}
	`,
		testProjectName,
		testEndpointName,
		base64.StdEncoding.EncodeToString([]byte("raw_config")),
		base64.StdEncoding.EncodeToString([]byte("rawDescriptor")))
)

func TestFetchRollouts(t *testing.T) {
	runTest(t, func(env *testEnv) {
		ctx := context.Background()
		// First request, VersionId should be empty.
		req := v2.DiscoveryRequest{
			Node: &core.Node{
				Id: node,
			},
			TypeUrl: cache.ListenerType,
		}

		resp, err := env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotListeners, err := marshaler.MarshalToString(resp.Resources[0])

		wantedListeners := fmt.Sprintf(`
		{
			"address":{
				"socketAddress":{
					"address":"0.0.0.0",
					"portValue":8080
				}
			},
			"filterChains":[
				{
					"filters":[
						{
							"config":{
								"http_filters":[
									{
										"config":{
											"proto_descriptor_bin":"%s",
											"services":[
												"%s"
											]
										},
										"name":"envoy.grpc_json_transcoder"
									},
									{
										"config":{

										},
										"name":"envoy.router"
									}
								],
								"route_config":{
									"name":"local_route",
									"virtual_hosts":[
										{
											"domains":[
												"*"
											],
											"name":"backend",
											"routes":[
												{
													"match":{
														"prefix":"/%s"
													},
													"route":{
														"cluster":"grpc_service"
													}
												}
											]
										}
									]
								},
								"stat_prefix":"ingress_http"
							},
							"name":"envoy.http_connection_manager"
						}
					]
				}
			]
		}
		`,
			base64.StdEncoding.EncodeToString([]byte("rawDescriptor")),
			testEndpointName,
			testEndpointName)

		if resp.Version != testConfigID {
			t.Errorf("snapshot cache fetch got version: %v, want: %v", resp.Version, testConfigID)
		}
		if !reflect.DeepEqual(resp.Request, req) {
			t.Errorf("snapshot cache fetch got request: %v, want: %v", resp.Request, req)
		}

		// Normalize both wantedListeners and gotListeners.
		wantedListeners = normalizeJson(wantedListeners)
		gotListeners = normalizeJson(gotListeners)

		if gotListeners != wantedListeners {
			t.Errorf("snapshot cache fetch got Listeners: %s, want: %s", gotListeners, wantedListeners)
		}
	})
}

// Test Environment setup.

type testEnv struct {
	configManager *ConfigManager
}

func runTest(t *testing.T, f func(*testEnv)) {
	mockConfig := initMockConfigServer(t)
	defer mockConfig.Close()
	fetchConfigURL = mockConfig.URL

	mockMetadata := initMockMetadataServer()
	defer mockMetadata.Close()
	serviceAccountTokenURL = mockMetadata.URL

	manager, err := NewConfigManager(testProjectName, testConfigID)
	if err != nil {
		t.Fatal("fail to initialize ConfigManager: ", err)
	}

	env := &testEnv{
		configManager: manager,
	}
	f(env)
}

func initMockConfigServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(normalizeJson(fakeConfig)))
	}))
}

type mock struct{}

func (mock) ID(*core.Node) string {
	return fakeNodeID
}

func normalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}
