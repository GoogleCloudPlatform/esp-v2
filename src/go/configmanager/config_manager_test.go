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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"google.golang.org/genproto/protobuf/api"
)

const (
	testProjectName  = "bookstore.endpoints.project123.cloud.goog"
	testEndpointName = "endpoints.examples.bookstore.Bookstore"
	testConfigID     = "2017-05-01r0"
	testProjectID    = "project123"
	fakeNodeID       = "id"
	fakeJwks         = "FAKEJWKS"
)

var (
	fakeConfig          = ``
	fakeRollout         = ``
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
												"ignored_query_parameters": [
                                                    "api_key",
													"key"
												],
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
											"name":"envoy.grpc_web"
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
			desc:            "Success for gRPC backend, with Jwt filter, with audiences, no Http Rules",
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
                	        "selector": "endpoints.examples.bookstore.Bookstore.ListShelves"
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
									"name":"envoy.grpc_web"
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
                "http": {
                    "rules": [
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.ListShelves",
                            "get": "/v1/shelves"
                        },
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                            "post": "/v1/shelves/{shelf}"
                        }
                    ]
                },
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
                            "selector": "endpoints.examples.bookstore.Bookstore.ListShelves",
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
                                                   "headers": [
                                                       {
                                                           "exact_match": "POST",
                                                           "name" : ":method"
                                                       }
                                                   ],
                                                   "regex": "/v1/shelves/[^\\/]+$"
                                                },
                                                "requires":{
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
                                                   "headers": [
                                                       {
                                                           "exact_match": "GET",
                                                           "name" : ":method"
                                                       }
                                                   ],
                                                   "path": "/v1/shelves"
                                                },
                                                "requires":{
                                                    "provider_name":"firebase"
                                                }
                                            },
					                        {
                                                "match":{
                                                    "path":"/endpoints.examples.bookstore.Bookstore/ListShelves"
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
                                    "name":"envoy.grpc_web"
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
			desc: "Success for gRPC backend, with Jwt filter, with multi requirements, matching with regex", backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "apis":[
                    {
                        "name":"%s",
                        "sourceContext": {
							"fileName": "bookstore.proto"
						}
					}
                ],
                "http": {
                    "rules": [
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.GetBook",
                            "get": "/v1/shelves/{shelf}/books/{book}"
                        },
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                            "delete": "/v1/shelves/{shelf}/books/{book}"
                        }
                    ]
                },
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
                            "selector": "endpoints.examples.bookstore.Bookstore.GetBook",
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
                                                    "headers": [
                                                        {
                                                            "exact_match": "GET",
                                                            "name" : ":method"
                                                        }
                                                    ],
                                                    "regex": "/v1/shelves/[^\\/]+/books/[^\\/]+$"
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
                                                    "path":"/endpoints.examples.bookstore.Bookstore/GetBook"
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
                                    "name":"envoy.grpc_web"
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
				"producer_project_id":"%s",
				"control" : {
					"environment": "servicecontrol.googleapis.com"
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
			}`, testProjectName, testProjectID, testEndpointName),
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
												"services":[
													{
														"service_control_uri":{
															"cluster":"service_control_cluster",
															"timeout":"5s",
															"uri":"https://servicecontrol.googleapis.com/v1/services/"
														},
														"service_name":"%s",
														"service_config_id":"%s",
														"producer_project_id":"%s",
														"token_cluster": "ads_cluster"
													}
												]
											},
											"name":"envoy.filters.http.service_control"
										},
										{
											"config":{
											},
											"name":"envoy.grpc_web"
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
			}`, testProjectName, testProjectName, testProjectName, testProjectName, testProjectName, testConfigID, testProjectID),
		},
		{
			desc:            "Success for HTTP1 backend, with Jwt filter, with audiences",
			backendProtocol: "http1",
			fakeServiceConfig: fmt.Sprintf(`{
                "apis":[
                    {
                        "name":"%s"
                    }
                ],
                "http": {
                    "rules": [
                        {
                            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                            "get": "/auth/info/googlejwt"
                        },
                        {
                            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                            "post": "/echo",
                            "body": "message"
                        }
                    ]
                },
                "authentication": {
                    "providers": [
                        {
                            "id": "firebase",
                            "issuer": "https://test_issuer.google.com/",
                            "jwks_uri": "$JWKSURI",
                            "audiences": "test_audience1, test_audience2 "
                        }
                    ],
                    "rules": [
                        {
                            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo"
                        },
                        {
                            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                            "requirements": [
                                {
                                    "provider_id": "firebase",
                                    "audiences": "test_audience1"
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
                                                    "headers":[
                                                        {
                                                            "exact_match":"GET",
                                                            "name":":method"
                                                        }
                                                    ],
                                                    "path":"/auth/info/googlejwt"
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
			desc:            "Success for backend that allow CORS",
			backendProtocol: "http1",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
		"producer_project_id":"%s",
		"control" : {
			"environment": "servicecontrol.googleapis.com"
		},
		"apis":[
                   {
                        "name":"%s",
                        "methods":[
			{
				"name": "Simplegetcors"
			}
			]
                    }
                ],
                "http": {
                    "rules": [
                        {
                            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                            "get": "/simplegetcors"
                        }
                    ]
                },
                "endpoints": [
		{
			"name": "%s",
			"allow_cors": true
		}
                ]
            }`, testProjectName, testProjectID, testEndpointName, testProjectName),
			wantedListeners: fmt.Sprintf(`{
			  "filters": [
			    {
			      "config": {
			        "http_filters": [
			          {
			            "config": {
			              "rules": [
			                {
			                  "pattern": {
			                    "http_method": "GET",
			                    "uri_template": "/simplegetcors"
			                  },
			                  "requires": {
			                    "operation_name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
			                    "service_name": "%s"
			                  }
			                },
			                {
			                  "pattern": {
			                    "http_method": "OPTIONS",
			                    "uri_template": "/simplegetcors"
			                  },
			                  "requires": {
			                    "api_key": {
			                      "allow_without_api_key": true
			                    },
			                    "operation_name": "CORS.0",
			                    "service_name": "%s"
			                  }
			                }
			              ],
			              "services": [
			                {
			                  "producer_project_id": "project123",
			                  "service_config_id": "2017-05-01r0",
			                  "service_control_uri": {
			                    "cluster": "service_control_cluster",
			                    "timeout": "5s",
			                    "uri": "https://servicecontrol.googleapis.com/v1/services/"
			                  },
			                  "service_name": "%s",
			                  "token_cluster": "ads_cluster"
			                }
			              ]
			            },
			            "name": "envoy.filters.http.service_control"
			          },
			          {
			            "config": {},
			            "name": "envoy.router"
			          }
			        ],
			        "route_config": {
			          "name": "local_route",
			          "virtual_hosts": [
			            {
			              "domains": [
			                "*"
			              ],
			              "name": "backend",
			              "routes": [
			                {
			                  "match": {
			                    "prefix": "/"
			                  },
			                  "route": {
			                    "cluster": "%s"
			                  }
			                }
			              ]
			            }
			          ]
			        },
			        "stat_prefix": "ingress_http"
			      },
			      "name": "envoy.http_connection_manager"
			    }
			  ]
			}`, testProjectName, testProjectName, testProjectName, testEndpointName),
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("service", testProjectName)
		flag.Set("version", testConfigID)
		flag.Set("rollout_strategy", ut.FixedRolloutStrategy)
		flag.Set("backend_protocol", tc.backendProtocol)

		runTest(t, ut.FixedRolloutStrategy, func(env *testEnv) {
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
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got unexpected Listeners", i, tc.desc)
				t.Errorf("Actual: %s", gotListeners)
				t.Errorf("Expected: %s", want)
			}
		})
	}
}

func TestFetchClusters(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig string
		wantedClusters    []string
		backendProtocol   string
	}{
		{
			desc: "Success for gRPC backend",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
		"control" : {
			"environment": "servicecontrol.googleapis.com"
		},
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
			backendProtocol: "grpc",
			wantedClusters: []string{
				fmt.Sprintf(`{
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
                "type":"STRICT_DNS",
                "http2ProtocolOptions": {}
	        }`, *flags.ClusterAddress, *flags.ClusterPort, testEndpointName, *flags.ClusterConnectTimeout/1e9),
				`{"name": "service_control_cluster",
		"connectTimeout": "5s",
		"type": "LOGICAL_DNS",
		"dnsLookupFamily": "V4_ONLY",
	    	"hosts": [ {
    	      	    "socketAddress": {
    	      	  	  "address": "servicecontrol.googleapis.com",
	    	      	  "portValue": 443
    	      	    }
    	        } ],
		"tlsContext": { "sni": "servicecontrol.googleapis.com" }
		}`,
			},
		},
		{
			desc: "Success for HTTP1 backend",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
		"control" : {
			"environment": "http://127.0.0.1:8000"
		},
                "apis":[
                    {
                        "name":"%s"
                    }
                ]
            }`, testProjectName, testEndpointName),
			backendProtocol: "http1",
			wantedClusters: []string{
				fmt.Sprintf(`{
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
                "type":"STRICT_DNS"
           }`, *flags.ClusterAddress, *flags.ClusterPort, testEndpointName, *flags.ClusterConnectTimeout/1e9),
				`{"name": "service_control_cluster",
		"connectTimeout": "5s",
		"type": "LOGICAL_DNS",
		"dnsLookupFamily": "V4_ONLY",
	    	"hosts": [ {
    	      	    "socketAddress": {
    	      	  	  "address": "127.0.0.1",
	    	      	  "portValue": 8000
    	      	    }
    	        } ]
		}`,
			},
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("service", testProjectName)
		flag.Set("version", testConfigID)
		flag.Set("rollout_strategy", ut.FixedRolloutStrategy)
		flag.Set("backend_protocol", tc.backendProtocol)

		runTest(t, ut.FixedRolloutStrategy, func(env *testEnv) {
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

			if resp.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, resp.Version, testConfigID)
			}
			if !reflect.DeepEqual(resp.Request, req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			// configManager.cache may change the order
			// sort them before comparing results.
			sorted_sources := resp.Resources
			sort.Slice(sorted_sources, func(i, j int) bool {
				return cache.GetResourceName(sorted_sources[i]) < cache.GetResourceName(sorted_sources[j])
			})

			for idx, want := range tc.wantedClusters {
				marshaler := &jsonpb.Marshaler{}
				gotClusters, err := marshaler.MarshalToString(sorted_sources[idx])
				if err != nil {
					t.Fatal(err)
				}

				gotClusters = normalizeJson(gotClusters)
				if want = normalizeJson(want); gotClusters != want {
					t.Errorf("Test Desc(%d): %s, idx %d snapshot cache fetch got Clusters: %s, want: %s", i, tc.desc, idx, gotClusters, want)
				}
			}
		})
	}
}

func TestMakeRouteConfig(t *testing.T) {
	testData := []struct {
		desc string
		// Test parameters, in the order of "cors_preset", "cors_allow_origin"
		// "cors_allow_origin_regex", "cors_allow_methods", "cors_allow_headers"
		// "cors_expose_headers"
		params           []string
		allowCredentials bool
		wantedError      string
		wantRoute        *route.CorsPolicy
	}{
		{
			desc:      "No Cors",
			wantRoute: nil,
		},
		{
			desc:        "Incorrect configured basic Cors",
			params:      []string{"basic", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: "cors_allow_origin cannot be empty when cors_preset=basic",
		},
		{
			desc:        "Incorrect configured  Cors",
			params:      []string{"", "", "", "GET", "", ""},
			wantedError: "cors_preset must be set in order to enable CORS support",
		},
		{
			desc:        "Incorrect configured regex Cors",
			params:      []string{"cors_with_regexs", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: `cors_preset must be either "basic" or "cors_with_regex"`,
		},
		{
			desc:   "Correct configured basic Cors, with allow methods",
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", ""},
			wantRoute: &route.CorsPolicy{
				AllowOrigin:      []string{"http://example.com"},
				AllowMethods:     "GET,POST,PUT,OPTIONS",
				AllowCredentials: &types.BoolValue{Value: false},
			},
		},
		{
			desc:   "Correct configured regex Cors, with allow headers",
			params: []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "Origin,Content-Type,Accept", ""},
			wantRoute: &route.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				AllowHeaders:     "Origin,Content-Type,Accept",
				AllowCredentials: &types.BoolValue{Value: false},
			},
		},
		{
			desc:             "Correct configured regex Cors, with expose headers",
			params:           []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "", "Content-Length"},
			allowCredentials: true,
			wantRoute: &route.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				ExposeHeaders:    "Content-Length",
				AllowCredentials: &types.BoolValue{Value: true},
			},
		},
	}

	for _, tc := range testData {
		// Initial flags
		if tc.params != nil {
			flag.Set("cors_preset", tc.params[0])
			flag.Set("cors_allow_origin", tc.params[1])
			flag.Set("cors_allow_origin_regex", tc.params[2])
			flag.Set("cors_allow_methods", tc.params[3])
			flag.Set("cors_allow_headers", tc.params[4])
			flag.Set("cors_expose_headers", tc.params[5])
		}
		flag.Set("cors_allow_credentials", strconv.FormatBool(tc.allowCredentials))

		gotRoute, err := makeRouteConfig(&api.Api{Name: "test-api"})
		if tc.wantedError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
				t.Errorf("Test (%s): expected err: %v, got: %v", tc.desc, tc.wantedError, err)
			}
			continue
		}

		gotHost := gotRoute.GetVirtualHosts()
		if len(gotHost) != 1 {
			t.Errorf("Test (%s): got expected number of virtual host", tc.desc)
		}
		gotCors := gotHost[0].GetCors()
		if !reflect.DeepEqual(gotCors, tc.wantRoute) {
			t.Errorf("Test (%s): makeRouteConfig failed, got Cors: %s, want: %s", tc.desc, gotCors, tc.wantRoute)
		}
	}
}

func TestServiceConfigAutoUpdate(t *testing.T) {
	var oldConfigID, oldRolloutID, newConfigID, newRolloutID string
	oldConfigID = "2018-12-05r0"
	oldRolloutID = oldConfigID
	newConfigID = "2018-12-05r1"
	newRolloutID = newConfigID
	testCase := struct {
		desc                  string
		fakeOldServiceRollout string
		fakeNewServiceRollout string
		fakeOldServiceConfig  string
		fakeNewServiceConfig  string
		backendProtocol       string
	}{
		desc: "Success for service config auto update",
		fakeOldServiceRollout: fmt.Sprintf(`{
			"rollouts": [
			    {
			      "rolloutId": "%s",
			      "createTime": "2018-12-05T19:07:18.438Z",
			      "createdBy": "mocktest@google.com",
			      "status": "SUCCESS",
			      "trafficPercentStrategy": {
			        "percentages": {
			          "%s": 100
			        }
			      },
			      "serviceName": "%s"
			    }
			  ]
			}`, oldRolloutID, oldConfigID, testProjectName),
		fakeNewServiceRollout: fmt.Sprintf(`{
			"rollouts": [
			    {
			      "rolloutId": "%s",
			      "createTime": "2018-12-05T19:07:18.438Z",
			      "createdBy": "mocktest@google.com",
			      "status": "SUCCESS",
			      "trafficPercentStrategy": {
			        "percentages": {
			          "%s": 40,
			          "%s": 60
			        }
			      },
			      "serviceName": "%s"
			    },
			    {
			      "rolloutId": "%s",
			      "createTime": "2018-12-05T19:07:18.438Z",
			      "createdBy": "mocktest@google.com",
			      "status": "SUCCESS",
			      "trafficPercentStrategy": {
			        "percentages": {
			          "%s": 100
			        }
			      },
			      "serviceName": "%s"
			    }
			  ]
			}`, newRolloutID, oldConfigID, newConfigID, testProjectName,
			oldRolloutID, oldConfigID, testProjectName),
		fakeOldServiceConfig: fmt.Sprintf(`{
				"name": "%s",
				"title": "Endpoints Example",
				"documentation": {
				"summary": "A simple Google Cloud Endpoints API example."
				},
				"apis":[
					{
						"name":"%s"
					}
				],
				"id": "%s"
			}`, testProjectName, testEndpointName, oldConfigID),
		fakeNewServiceConfig: fmt.Sprintf(`{
				"name": "%s",
				"title": "Endpoints Example",
				"documentation": {
				"summary": "A simple Google Cloud Endpoints API example."
				},
				"apis":[
					{
						"name":"%s"
					}
				],
				"id": "%s"
			}`, testProjectName, testEndpointName, newConfigID),
		backendProtocol: "grpc",
	}

	// Overrides fakeConfig with fakeOldServiceConfig for the test case.
	fakeConfig = testCase.fakeOldServiceConfig
	fakeRollout = testCase.fakeOldServiceRollout
	checkNewRolloutInterval = 1 * time.Second
	flag.Set("service", testProjectName)
	flag.Set("version", testConfigID)
	flag.Set("rollout_strategy", ut.ManagedRolloutStrategy)
	flag.Set("backend_protocol", testCase.backendProtocol)

	runTest(t, ut.ManagedRolloutStrategy, func(env *testEnv) {
		var resp *cache.Response
		var err error
		ctx := context.Background()
		req := v2.DiscoveryRequest{
			Node: &core.Node{
				Id: *flags.Node,
			},
			TypeUrl: cache.ListenerType,
		}
		resp, err = env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Version != oldConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", testCase.desc, resp.Version, oldConfigID)
		}
		if env.configManager.curRolloutID != oldRolloutID {
			t.Errorf("Test Desc: %s, config manager rollout id: %v, want: %v", testCase.desc, env.configManager.curRolloutID, oldRolloutID)
		}
		if !reflect.DeepEqual(resp.Request, req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}

		fakeConfig = testCase.fakeNewServiceConfig
		fakeRollout = testCase.fakeNewServiceRollout
		time.Sleep(time.Duration(checkNewRolloutInterval + time.Second))

		resp, err = env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Version != newConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", testCase.desc, resp.Version, newConfigID)
		}
		if env.configManager.curRolloutID != newRolloutID {
			t.Errorf("Test Desc: %s, config manager rollout id: %v, want: %v", testCase.desc, env.configManager.curRolloutID, newRolloutID)
		}
		if !reflect.DeepEqual(resp.Request, req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}
	})
}

func TestGetEndpointAllowCorsFlag(t *testing.T) {
	testData := []struct {
		desc                string
		fakeServiceConfig   string
		wantedAllowCorsFlag bool
	}{
		{
			desc: "Return true for endpoint name matching service name",
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
                ],
								"endpoints": [
										{
													"name": "%s",
													"allow_cors": true
										}
                ]
		    }`, testProjectName, testEndpointName, testProjectName),
			wantedAllowCorsFlag: true,
		},
		{
			desc: "Return false for not setting allow_cors",
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
                ],
								"endpoints": [
										{
													"name": "%s"
										}
                ]
		    }`, testProjectName, testEndpointName, testProjectName),
			wantedAllowCorsFlag: false,
		},
		{
			desc: "Return false for endpoint name not matching service name",
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
                ],
								"endpoints": [
										{
													"name": "%s",
													"allow_cors": true
										}
                ]
		    }`, testProjectName, testEndpointName, "echo.endpoints.project123.cloud.goog"),
			wantedAllowCorsFlag: false,
		},
		{
			desc: "Return false for empty endpoint field",
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
			wantedAllowCorsFlag: false,
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("service", testProjectName)
		flag.Set("version", testConfigID)
		flag.Set("rollout_strategy", ut.FixedRolloutStrategy)
		flag.Set("backend_protocol", "http1")

		runTest(t, ut.FixedRolloutStrategy, func(env *testEnv) {
			allowCorsFlag := env.configManager.getEndpointAllowCorsFlag()
			if allowCorsFlag != tc.wantedAllowCorsFlag {
				t.Errorf("Test Desc(%d): %s, allow CORS flag got: %v, want: %v", i, tc.desc, allowCorsFlag, tc.wantedAllowCorsFlag)
			}
		})
	}
}

// Test Environment setup.

type testEnv struct {
	configManager *ConfigManager
}

func runTest(t *testing.T, testRolloutStrategy string, f func(*testEnv)) {
	mockConfig := initMockConfigServer(t)
	defer mockConfig.Close()
	fetchConfigURL = func(serviceName, configID string) string {
		return mockConfig.URL
	}

	mockRollout := initMockRolloutServer(t)
	defer mockRollout.Close()
	fetchRolloutsURL = func(serviceName string) string {
		return mockRollout.URL
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
	manager, err := NewConfigManager()
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

func initMockRolloutServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(normalizeJson(fakeRollout)))
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
