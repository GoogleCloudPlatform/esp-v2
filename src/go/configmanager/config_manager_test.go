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
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
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
	fakeJwks         = "FAKEJWKS"
)

var (
	fakeConfig          = ``
	fakeProtoDescriptor = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))
)

func TestFetchListeners(t *testing.T) {
	testData := []struct {
		desc              string
		backendProtocol   string
		fakeServiceConfig string
		wantedListeners   string
	}{
		{
			desc:            "Success for gRPC backend with transcoding",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
				"name":"%s",
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
							"filePath":"api_descriptor.pb",
							"fileContents":"%s",
							"fileType":"FILE_DESCRIPTOR_SET_PROTO"
						}
					]
				}
			}`, testProjectName, testEndpointName, fakeProtoDescriptor),
			wantedListeners: fmt.Sprintf(`{
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
															"prefix":"/"
														},
														"route":{
															"cluster": "%s"
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
			}`,
				fakeProtoDescriptor, testEndpointName, testEndpointName),
		},
		{
			desc:            "Success for gRPC backend, with Jwt filter, with audiences",
			backendProtocol: "grpc",
			fakeServiceConfig: fmt.Sprintf(`{
				"apis":[
					{
						"name":"%s"
					}
				],
				"authentication": {
					"providers": [
						{
							"id": "firebase",
							"issuer": "https://test_issuer.google.com/",
							"jwks_uri": "$JWKSURI",
							"audiences": "test_audience1, test_audience2 "
						},
                        {
                            "id": "unknownId",
                            "issuer": "https://test_issuer.google.com/",
                            "jwks_uri": "invalidUrl"
                        }
					],
					"rules": [
                        {
                	        "selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                            "requirements": [
                                {
                                    "provider_id": "firebase",
                                    "audiences": "test_audience1"
                                }
                            ]
                        },
                        {
                	        "selector": "endpoints.examples.bookstore.Bookstore.ListShelf"
                        }
        	        ]
                }
            }`, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
                "filters":[
                    {
                        "config":{
                            "http_filters":[
                                {
                                    "config": {
                                        "providers": {
                                            "firebase": {
                                                "audiences":["test_audience1", "test_audience2"],
                                               	"issuer":"https://test_issuer.google.com/",
                                               	"local_jwks": {
                                               	    "inline_string": "%s"
                                               	}
                                            }
                                        },
                                        "rules": [
                                            {
						"match":{
						    "regex":""
						},
						"requires": {
						    "provider_and_audiences": {
							"audiences":["test_audience1"],
							"provider_name":"firebase"
						    }
					        }
					   },
					   {
                                                "match":{
                                                    "path":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                                },
                                                "requires": {
                                                    "provider_and_audiences": {
                                                	"audiences": ["test_audience1"],
                                                        "provider_name":"firebase"
                                                    }
                                                }
                                            }
					]
                                    },
                                    "name":"envoy.filters.http.jwt_authn"
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
                                                        "prefix":"/"
                                                    },
                                                    "route":{
                                                        "cluster": "%s"
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
            }`, fakeJwks, testEndpointName),
		},
		{
			desc:            "Success for gRPC backend, with Jwt filter, without audiences",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "apis":[
                    {
                        "name":"%s"
                    }
                ],
                "authentication": {
        	        "providers": [
        	            {
        	 	            "id": "firebase",
        	 	            "issuer": "https://test_issuer.google.com/",
        	 	            "jwks_uri": "$JWKSURI"
        	            }
        	        ],
        	        "rules": [
                        {
                	        "selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                            "requirements": [
                                {
                                    "provider_id": "firebase"
                                }
                            ]
                        },
                        {
                	        "selector": "endpoints.examples.bookstore.Bookstore.ListShelf",
                            "requirements": [
                                {
                                    "provider_id": "firebase"
                                }
                            ]
                        }
        	        ]
                }
            }`, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
                "filters":[
                    {
                        "config":{
                            "http_filters":[
                                {
                                    "config": {
                                        "providers": {
                                            "firebase": {
                                               	"issuer":"https://test_issuer.google.com/",
                                               	"local_jwks": {
                                               	    "inline_string": "%s"
                                               	}
                                            }
                                        },
                                        "rules": [
                                            {
						"match":{
						    "regex":""
						},
						"requires": {
						    "provider_name":"firebase"
                                                }
					    },
					    {
                                                "match":{
                                                    "path":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                                },
                                                "requires": {
                                                    "provider_name":"firebase"
                                                }
                                            },
					    {
						"match":{
						    "regex":""
					        },
						"requires":{
						    "provider_name":"firebase"
						}
					    },
					    {
                                                "match":{
                                                    "path":"/endpoints.examples.bookstore.Bookstore/ListShelf"
                                                },
                                                "requires": {
                                                    "provider_name":"firebase"
                                                }
                                            }
					   
                                        ]
                                    },
                                    "name":"envoy.filters.http.jwt_authn"
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
                                                        "prefix":"/"
                                                    },
                                                    "route":{
                                                        "cluster": "%s"
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
            }`, fakeJwks, testEndpointName),
		},
		{
			desc:            "Success for gRPC backend, with Jwt filter, with multi requirements",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "apis":[
                    {
                        "name":"%s",
                        "sourceContext": {
							"fileName": "bookstore.proto"
						}
					}
                ],
                "authentication": {
        	        "providers": [
        	            {
        	 	            "id": "firebase1",
        	 	            "issuer": "https://test_issuer.google.com/",
        	 	            "jwks_uri": "$JWKSURI"
        	            },
         	            {
        	 	            "id": "firebase2",
        	 	            "issuer": "https://test_issuer.google.com/",
        	 	            "jwks_uri": "$JWKSURI"
        	            }
        	        ],
        	        "rules": [
                        {
                	        "selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                            "requirements": [
                                {
                                    "provider_id": "firebase1"
                                },
                                {
                                    "provider_id": "firebase2"
                                }
                            ]
                        }
        	        ]
                }
            }`, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
                "filters":[
                    {
                        "config":{
                            "http_filters":[
                                {
                                    "config": {
                                        "providers": {
                                            "firebase1": {
                                               	"issuer":"https://test_issuer.google.com/",
                                               	"local_jwks": {
                                               	    "inline_string": "%s"
                                               	}
                                            },
                                            "firebase2": {
                                               	"issuer":"https://test_issuer.google.com/",
                                               	"local_jwks": {
                                               	    "inline_string": "%s"
                                               	}
                                            }
                                        },
                                        "rules": [
					    {
						"match":{
						    "regex": ""
					         },
						 "requires": {
                                                    "requires_any": {
                                                    	"requirements": [
                                                    	    {
                                                    	    	"provider_name": "firebase1"
                                                    	    },
                                                    	    {
                                                    	    	"provider_name": "firebase2"
                                                    	    }
                                                    	]
                                                    }
					        } 
                                            },
					    {
                                                "match":{
                                                    "path":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                                },
                                                "requires": {
                                                    "requires_any": {
                                                    	"requirements": [
                                                    	    {
                                                    	    	"provider_name": "firebase1"
                                                    	    },
                                                    	    {
                                                    	    	"provider_name": "firebase2"
                                                    	    }
                                                    	]
                                                    }
                                                }
                                            }
                                        ]
                                    },
                                    "name":"envoy.filters.http.jwt_authn"
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
                                                        "prefix":"/"
                                                    },
                                                    "route":{
                                                        "cluster": "%s"
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
            }`, fakeJwks, fakeJwks, testEndpointName),
		},
		{
			desc:            "Success for gRPC backend with Service Control",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
				"name":"%s",
				"control" : {
					"environment": "servivcecontrol.googleapis.com"
				},
				"apis":[
					{
						"name":"%s",
						"version":"v1",
						"syntax":"SYNTAX_PROTO3",
                        "sourceContext": {
							"fileName": "bookstore.proto"
						},
						"methods":[
							{
								"name": "ListShelves"
							},
							{
								"name": "CreateShelf"
							}
						]
					}
				],
				"http": {
					"rules": [
						{
							"selector": "endpoints.examples.bookstore.Bookstore.ListShelves",
							"get": "/v1/shelves"
						},
						{
							"selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
							"post": "/v1/shelves",
							"body": "shelf"
						}
					]
				}
			}`, testProjectName, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
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
												"rules":[
													{
														"pattern": {
															"http_method":"POST",
															"uri_template":"/endpoints.examples.bookstore.Bookstore/ListShelves"
														},
														"requires":{
															"operation_name":"endpoints.examples.bookstore.Bookstore.ListShelves",
															"service_name":"%s"
														}
													},
													{
														"pattern": {
															"http_method":"GET",
															"uri_template":"/v1/shelves"
														},
														"requires":{
															"operation_name":"endpoints.examples.bookstore.Bookstore.ListShelves",
															"service_name":"%s"
														}
													},
													{
														"pattern": {
															"http_method":"POST",
															"uri_template":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
														},
														"requires":{
															"operation_name":"endpoints.examples.bookstore.Bookstore.CreateShelf",
															"service_name":"%s"
														}
													},
													{
														"pattern": {
															"http_method":"POST",
															"uri_template":"/v1/shelves"
														},
														"requires":{
															"operation_name":"endpoints.examples.bookstore.Bookstore.CreateShelf",
															"service_name":"%s"
														}
													}
												],
												"service_control_uri":{
													"cluster":"service_control_cluster",
													"timeout":"5s",
													"uri":"https://servicecontrol.googleapis.com/v1/services/"
												},
												"service_name":"%s",
												"services":[
													{
														"service_control_uri":{
															"cluster":"service_control_cluster",
															"timeout":"5s",
															"uri":"https://servicecontrol.googleapis.com/v1/services/"
														},
														"service_name":"%s",
														"token_uri":{
															"cluster":"gcp_metadata_cluster",
															"timeout":"5s",
															"uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
														}
													}
												]
											},
											"name":"envoy.filters.http.service_control"
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
															"prefix":"/"
														},
														"route":{
															"cluster":"endpoints.examples.bookstore.Bookstore"
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
			}`, testProjectName, testProjectName, testProjectName, testProjectName, testProjectName, testProjectName)},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("backend_protocol", tc.backendProtocol)

		runTest(t, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			req := v2.DiscoveryRequest{
				Node: &core.Node{
					Id: *flags.Node,
				},
				TypeUrl: cache.ListenerType,
			}
			resp, err := env.configManager.cache.Fetch(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			marshaler := &jsonpb.Marshaler{}
			gotListeners, err := marshaler.MarshalToString(resp.Resources[0])
			if err != nil {
				t.Fatal(err)
			}

			if resp.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, resp.Version, testConfigID)
			}
			if !reflect.DeepEqual(resp.Request, req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			// Normalize both wantedListeners and gotListeners.
			gotListeners = normalizeJson(gotListeners)
			if want := normalizeJson(tc.wantedListeners); gotListeners != want && !strings.Contains(gotListeners, want) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got Listeners: %s, want: %s", i, tc.desc, gotListeners, want)
			}
		})
	}
}

func TestFetchClusters(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig string
		wantedClusters    string
	}{
		{
			desc: "Success for gRPC backend",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
                "apis":[
                    {
                        "name":"%s",
                        "version":"v1",
                        "syntax":"SYNTAX_PROTO3",
                        "sourceContext": {
							"fileName": "bookstore.proto"
						}
					}
                ]
		    }`, testProjectName, testEndpointName),
			wantedClusters: fmt.Sprintf(`{
	    	    "hosts": [
	    	        {
	    	      	    "socketAddress": {
	    	      	  	    "address": "%s",
	    	      	  	    "portValue": %d
	    	      	    }
	    	        }
	    	    ],
	    	    "name": "%s",
	    	    "connectTimeout": "%ds",
                "http2ProtocolOptions": {}
	        }`, *flags.ClusterAddress, *flags.ClusterPort, testEndpointName, *flags.ClusterConnectTimeout/1e9),
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig

		runTest(t, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			req := v2.DiscoveryRequest{
				Node: &core.Node{
					Id: *flags.Node,
				},
				TypeUrl: cache.ClusterType,
			}

			resp, err := env.configManager.cache.Fetch(ctx, req)
			if err != nil {
				t.Fatal(err)
			}

			marshaler := &jsonpb.Marshaler{}
			gotClusters, err := marshaler.MarshalToString(resp.Resources[0])
			if err != nil {
				t.Fatal(err)
			}

			if resp.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, resp.Version, testConfigID)
			}
			if !reflect.DeepEqual(resp.Request, req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			gotClusters = normalizeJson(gotClusters)
			if want := normalizeJson(tc.wantedClusters); gotClusters != want {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got Clusters: %s, want: %s", i, tc.desc, gotClusters, want)
			}
		})
	}
}

// Test Environment setup.

type testEnv struct {
	configManager *ConfigManager
}

func runTest(t *testing.T, f func(*testEnv)) {
	mockConfig := initMockConfigServer(t)
	defer mockConfig.Close()
	fetchConfigURL = func(serviceName, configID string) string {
		return mockConfig.URL
	}
	mockMetadata := initMockMetadataServer(fakeToken)
	defer mockMetadata.Close()
	fetchMetadataURL = func(_ string) string {
		return mockMetadata.URL
	}

	mockJwksIssuer := initMockJwksIssuer(t)
	defer mockJwksIssuer.Close()

	// Replace $JWKSURI here, since it depends on the mock server.
	fakeConfig = strings.Replace(fakeConfig, "$JWKSURI", mockJwksIssuer.URL, -1)
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

func initMockJwksIssuer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fakeJwks))
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
