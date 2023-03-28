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

package testdata

import (
	"encoding/base64"
	"fmt"

	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/protobuf/proto"
)

const (
	TestFetchListenersProjectName  = "bookstore.endpoints.project123.cloud.goog"
	TestFetchListenersEndpointName = "endpoints.examples.bookstore.Bookstore"
	TestFetchListenersConfigID     = "2017-05-01r0"
	testProjectID                  = "project123"
	localReplyConfig               = `"localReplyConfig": {
                    "bodyFormat": {
                      "jsonFormat": {
                        "code": "%RESPONSE_CODE%",
                        "message":"%LOCAL_REPLY_BODY%"
                      }
                    }
                  }`
)

var (
	rawDescriptor, _       = proto.Marshal(&descpb.FileDescriptorSet{})
	fakeProtoDescriptor    = base64.StdEncoding.EncodeToString(rawDescriptor)
	testBackendClusterName = fmt.Sprintf("backend-cluster-%s_local", TestFetchListenersProjectName)
)

var (
	FakeServiceConfigForGrpcWithTranscoding = fmt.Sprintf(`{
                "name":"%s",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name":"%s",
                        "version":"v1",
                        "syntax":"SYNTAX_PROTO3",
                    "methods": [
                          {
                             "name": "CreateShelf"
                          }
                        ]
                    }
                ],
                "endpoints": [{"name": "%s"}],
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
            }`, TestFetchListenersProjectName, TestFetchListenersEndpointName, TestFetchListenersEndpointName, fakeProtoDescriptor)

	WantedListsenerForGrpcWithTranscoding = fmt.Sprintf(`
{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "envoy.filters.http.grpc_web",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
                }
              },
              {
                "name": "envoy.filters.http.grpc_json_transcoder",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
                  "autoMapping": true,
                  "convertGrpcStatus": true,
                  "queryParamUnescapePlus": true,
                  "ignoredQueryParameters": [
                    "api_key",
                    "key"
                  ],
                  "printOptions": {},
                  "protoDescriptorBin": "%s",
                  "services": [
                    "%s"
                  ]
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "%s",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "%s",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                         "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "statPrefix": "ingress_http",
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ],
  "name": "ingress_listener"
}
`,
		fakeProtoDescriptor, TestFetchListenersEndpointName, testBackendClusterName, testBackendClusterName, localReplyConfig)

	FakeServiceConfigForGrpcWithJwtFilterWithAuds = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name":"%s",
                        "methods": [
                           {
                                "name": "CreateShelf"
                           }
                        ]
                    }
                ],
                "endpoints": [{"name": "%s"}],
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
            }`, TestFetchListenersEndpointName, TestFetchListenersEndpointName)

	WantedListsenerForGrpcWithJwtFilterWithAuds = fmt.Sprintf(`
{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "envoy.filters.http.jwt_authn",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                  "requirementMap": {
                    "endpoints.examples.bookstore.Bookstore.CreateShelf": {
                      "providerAndAudiences": {
                        "audiences": [
                          "test_audience1"
                        ],
                        "providerName": "firebase"
                      }
                    }
                  },
                  "providers": {
                    "firebase": {
                      "audiences": [
                        "test_audience1",
                        "test_audience2"
                      ],
                      "forward": true,
                      "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                      "fromHeaders": [
                        {
                          "name": "Authorization",
                          "valuePrefix": "Bearer "
                        },
                        {
                          "name": "X-Goog-Iap-Jwt-Assertion"
                        }
                      ],
                      "fromParams": [
                        "access_token"
                      ],
                      "issuer": "https://test_issuer.google.com/",
                      "jwtCacheConfig": {
                        "jwtCacheSize": 1000
                      },
                      "payloadInMetadata": "jwt_payloads",
                      "remoteJwks": {
                        "cacheDuration": "300s",
                        "httpUri": {
                          "cluster": "jwt-provider-cluster-$JWKSURI:443",
                          "timeout": "30s",
                          "uri": "$JWKSURI"
                        },
                        "asyncFetch": {}
                      }
                    }
                  }
                }
              },
              {
                "name": "envoy.filters.http.grpc_web",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "%s",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "%s",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "statPrefix": "ingress_http",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}
              `, testBackendClusterName, testBackendClusterName, localReplyConfig)

	FakeServiceConfigForGrpcWithJwtFilterWithoutAuds = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name":"%s",
                        "methods": [
                          {
                             "name": "CreateShelf"
                          },
                          {
                             "name": "ListShelves"
                          }
                        ]
                    }
                ],
                "endpoints": [{"name": "%s"}],
                "http": {
                    "rules": [
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.ListShelves",
                            "get": "/v1/shelves"
                        },
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                            "post": "/v1/shelves/{shelf=*}"
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
            }`, TestFetchListenersEndpointName, TestFetchListenersEndpointName)

	WantedListsenerForGrpcWithJwtFilterWithoutAuds = fmt.Sprintf(`{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "envoy.filters.http.jwt_authn",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                  "requirementMap": {
                    "endpoints.examples.bookstore.Bookstore.CreateShelf": {
                      "providerName": "firebase"
                    },
                    "endpoints.examples.bookstore.Bookstore.ListShelves": {
                      "providerName": "firebase"
                    }
                  },
                  "providers": {
                    "firebase": {
                      "audiences": [
                        "https://bookstore.endpoints.project123.cloud.goog"
                      ],
                      "forward": true,
                      "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                      "fromHeaders": [
                        {
                          "name": "Authorization",
                          "valuePrefix": "Bearer "
                        },
                        {
                          "name": "X-Goog-Iap-Jwt-Assertion"
                        }
                      ],
                      "fromParams": [
                        "access_token"
                      ],
                      "issuer": "https://test_issuer.google.com/",
                      "jwtCacheConfig": {
                        "jwtCacheSize": 1000
                      },
                      "payloadInMetadata": "jwt_payloads",
                      "remoteJwks": {
                        "cacheDuration": "300s",
                        "httpUri": {
                          "cluster": "jwt-provider-cluster-$JWKSURI:443",
                          "timeout": "30s",
                          "uri": "$JWKSURI"
                        },
                        "asyncFetch": {}
                      }
                    }
                  }
                }
              },
              {
                "name": "envoy.filters.http.grpc_web",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      },
                     	"name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "safeRegex": {
                          "regex": "^/v1/shelves/[^\\/]+\\/?$"
                        }
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/ListShelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/ListShelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/v1/shelves"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/v1/shelves/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves/{shelf=*}"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves/{shelf=*}\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "safeRegex": {
                          "regex": "^/v1/shelves/[^\\/]+\\/?$"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "statPrefix": "ingress_http",
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}
`, localReplyConfig)

	FakeServiceConfigForGrpcWithJwtFilterWithMultiReqs = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name":"%s",
                        "sourceContext": {
                            "fileName": "bookstore.proto"
                        },
                        "methods": [
                          {
                             "name": "DeleteBook"
                          },
                          {
                             "name": "GetBook"
                          }
                        ]
                    }
                ],
                "http": {
                    "rules": [
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.GetBook",
                            "get": "/v1/shelves/{shelf=*}/books/{book=*}"
                        },
                        {
                            "selector": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                            "delete": "/v1/shelves/{shelf=*}/books/{book=*}"
                        }
                    ]
                },
                "endpoints": [{"name": "%s"}],
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
            }`, TestFetchListenersEndpointName, TestFetchListenersEndpointName)
	WantedListenerForGrpcWithJwtFilterWithMultiReqs = fmt.Sprintf(`
{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "envoy.filters.http.jwt_authn",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                  "requirementMap": {
                    "endpoints.examples.bookstore.Bookstore.GetBook": {
                      "requiresAny": {
                        "requirements": [
                          {
                            "providerName": "firebase1"
                          },
                          {
                            "providerName": "firebase2"
                          }
                        ]
                      }
                    }
                  },
                  "providers": {
                    "firebase1": {
                      "audiences": [
                        "https://bookstore.endpoints.project123.cloud.goog"
                      ],
                      "forward": true,
                      "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                      "fromHeaders": [
                        {
                          "name": "Authorization",
                          "valuePrefix": "Bearer "
                        },
                        {
                          "name": "X-Goog-Iap-Jwt-Assertion"
                        }
                      ],
                      "fromParams": [
                        "access_token"
                      ],
                      "issuer": "https://test_issuer.google.com/",
                      "jwtCacheConfig": {
                        "jwtCacheSize": 1000
                      },
                      "payloadInMetadata": "jwt_payloads",
                      "remoteJwks": {
                        "cacheDuration": "300s",
                        "httpUri": {
                          "cluster": "jwt-provider-cluster-$JWKSURI:443",
                          "timeout": "30s",
                          "uri": "$JWKSURI"
                        },
                        "asyncFetch": {}
                      }
                    },
                    "firebase2": {
                      "audiences": [
                        "https://bookstore.endpoints.project123.cloud.goog"
                      ],
                      "forward": true,
                      "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                      "fromHeaders": [
                        {
                          "name": "Authorization",
                          "valuePrefix": "Bearer "
                        },
                        {
                          "name": "X-Goog-Iap-Jwt-Assertion"
                        }
                      ],
                      "fromParams": [
                        "access_token"
                      ],
                      "issuer": "https://test_issuer.google.com/",
                      "jwtCacheConfig": {
                        "jwtCacheSize": 1000
                      },
                      "payloadInMetadata": "jwt_payloads",
                      "remoteJwks": {
                        "cacheDuration": "300s",
                        "httpUri": {
                          "cluster": "jwt-provider-cluster-$JWKSURI:443",
                          "timeout": "30s",
                          "uri": "$JWKSURI"
                        },
                        "asyncFetch": {}
                      }
                    }
                  }
                }
              },
              {
                "name": "envoy.filters.http.grpc_web",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress DeleteBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/DeleteBook"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress DeleteBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/DeleteBook/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress GetBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/GetBook"
                      },
                     "name": "endpoints.examples.bookstore.Bookstore.GetBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.GetBook"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress GetBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/GetBook/"
                      },
                     "name": "endpoints.examples.bookstore.Bookstore.GetBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.GetBook"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress DeleteBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"DELETE"},
                            "name": ":method"
                          }
                        ],
                        "safeRegex": {
                          "regex": "^/v1/shelves/[^\\/]+/books/[^\\/]+\\/?$"
                        }
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress GetBook"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "safeRegex": {
                          "regex": "^/v1/shelves/[^\\/]+/books/[^\\/]+\\/?$"
                        }
                      },
                     "name": "endpoints.examples.bookstore.Bookstore.GetBook",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "endpoints.examples.bookstore.Bookstore.GetBook"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/DeleteBook"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/DeleteBook\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/DeleteBook"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/DeleteBook"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/DeleteBook\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/DeleteBook/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/GetBook"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/GetBook\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/GetBook"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/GetBook"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/GetBook\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/GetBook/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves/{shelf=*}/books/{book=*}"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves/{shelf=*}/books/{book=*}\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "safeRegex": {
                          "regex": "^/v1/shelves/[^\\/]+/books/[^\\/]+\\/?$"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "statPrefix": "ingress_http",
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}
`, localReplyConfig)

	FakeServiceConfigForGrpcWithServiceControl = fmt.Sprintf(`{
                "name":"%s",
                "id": "2017-05-01r0",
                "endpoints" : [{"name": "%s"}],
                "producer_project_id":"%s",
                "control" : {
                    "environment": "servicecontrol.googleapis.com"
                },
                "logging": {
                    "producerDestinations": [{
                    "logs": [
                          "endpoints_log"
                       ],
                    "monitoredResource": "api"
                   }
                   ]
                },
                "logs": [
                    {
                       "name": "endpoints_log"
                    }
                ],
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
                                "name": "CreateShelf"
                            },
                            {
                                "name": "ListShelves"
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
            }`, TestFetchListenersProjectName, TestFetchListenersEndpointName, testProjectID, TestFetchListenersEndpointName)

	WantedListenerForGrpcWithServiceControl = fmt.Sprintf(`{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "com.google.espv2.filters.http.service_control",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.FilterConfig",
                  "imdsToken": {
                    "cluster": "metadata-cluster",
                    "timeout": "30s",
                    "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                  },
                  "depErrorBehavior": "BLOCK_INIT_ON_ANY_ERROR",
                  "gcpAttributes": {
                    "platform": "GCE(ESPv2)"
                  },
                  "generatedHeaderPrefix": "X-Endpoint-",
                  "requirements": [
                    {
                      "apiName": "endpoints.examples.bookstore.Bookstore",
                      "apiVersion": "v1",
                      "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "serviceName": "bookstore.endpoints.project123.cloud.goog"
                    },
                    {
                      "apiName": "endpoints.examples.bookstore.Bookstore",
                      "apiVersion": "v1",
                      "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "serviceName": "bookstore.endpoints.project123.cloud.goog"
                    }
                  ],
                  "scCallingConfig": {
                    "networkFailOpen": true
                  },
                  "serviceControlUri": {
                    "cluster": "service-control-cluster",
                    "timeout": "30s",
                    "uri": "https://servicecontrol.googleapis.com:443/v1/services"
                  },
                  "services": [
                    {
                      "backendProtocol": "grpc",
                      "jwtPayloadMetadataName": "jwt_payloads",
                      "producerProjectId": "%v",
                      "serviceConfig": {
                        "logging": {
                          "producerDestinations": [
                            {
                              "logs": [
                                "endpoints_log"
                              ],
                              "monitoredResource": "api"
                            }
                          ]
                        },
                        "logs": [
                          {
                            "name": "endpoints_log"
                          }
                        ]
                      },
                      "serviceConfigId": "%v",
                      "serviceName": "%v",
                      "tracingDisabled": true,
                      "tracingProjectId": "fake-project-id"
                    }
                  ]
                }
              },
              {
                "name": "envoy.filters.http.grpc_web",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ListShelves"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/v1/shelves/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/CreateShelf/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/ListShelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/endpoints.examples.bookstore.Bookstore/ListShelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/endpoints.examples.bookstore.Bookstore/ListShelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/endpoints.examples.bookstore.Bookstore/ListShelves/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/v1/shelves"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/v1/shelves"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/v1/shelves\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/v1/shelves/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "statPrefix": "ingress_http",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}`, testProjectID, TestFetchListenersConfigID, TestFetchListenersProjectName, localReplyConfig)

	FakeServiceConfigForHttp = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                        "methods": [
                          {
                             "name": "Echo"
                          },
                          {
                             "name": "Echo_Auth_Jwt"
                          }
                        ]
                    }
                ],
                "endpoints": [{"name": "%s"}],
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
            }`, TestFetchListenersEndpointName)

	WantedListenerForHttp = fmt.Sprintf(`{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "envoy.filters.http.jwt_authn",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                  "requirementMap": {
                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": {
                      "providerAndAudiences": {
                        "audiences": [
                          "test_audience1"
                        ],
                        "providerName": "firebase"
                      }
                    }
                  },
                  "providers": {
                    "firebase": {
                      "audiences": [
                        "test_audience1",
                        "test_audience2"
                      ],
                      "forward": true,
                      "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                      "fromHeaders": [
                        {
                          "name": "Authorization",
                          "valuePrefix": "Bearer "
                        },
                        {
                          "name": "X-Goog-Iap-Jwt-Assertion"
                        }
                      ],
                      "fromParams": [
                        "access_token"
                      ],
                      "issuer": "https://test_issuer.google.com/",
                      "jwtCacheConfig": {
                        "jwtCacheSize": 1000
                      },
                      "payloadInMetadata": "jwt_payloads",
                      "remoteJwks": {
                        "cacheDuration": "300s",
                        "httpUri": {
                          "cluster": "jwt-provider-cluster-$JWKSURI:443",
                          "timeout": "30s",
                          "uri": "$JWKSURI"
                        },
                        "asyncFetch": {}
                      }
                    }
                  }
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress Echo_Auth_Jwt"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/auth/info/googlejwt"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Echo_Auth_Jwt"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/auth/info/googlejwt/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "envoy.filters.http.jwt_authn": {
                          "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig",
                          "requirementName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Echo"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/echo"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Echo"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/echo/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/auth/info/googlejwt"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/auth/info/googlejwt\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/auth/info/googlejwt"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/auth/info/googlejwt"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/auth/info/googlejwt\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/auth/info/googlejwt/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/echo"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/echo"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/echo"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/echo/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "statPrefix": "ingress_http",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}`, localReplyConfig)

	FakeServiceConfigAllowCorsTracingDebug = fmt.Sprintf(`{
                "name":"%s",
                "id": "2017-05-01r0",
                "producer_project_id":"%s",
                "control" : {
                    "environment": "servicecontrol.googleapis.com"
                },
                "apis":[
                    {
                        "name":"1.echo_api_endpoints_cloudesf_testing_cloud_goog",
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
            }`, TestFetchListenersProjectName, testProjectID, TestFetchListenersProjectName)

	WantedListenersAllowCorsTracingDebug = fmt.Sprintf(`{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
  "address": {
    "socketAddress": {
      "address": "0.0.0.0",
      "portValue": 8080
    }
  },
  "name": "ingress_listener",
  "filterChains": [
    {
      "filters": [
        {
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v11.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "com.google.espv2.filters.http.service_control",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.FilterConfig",
                  "imdsToken": {
                    "cluster": "metadata-cluster",
                    "timeout": "30s",
                    "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                  },
                  "depErrorBehavior": "BLOCK_INIT_ON_ANY_ERROR",
                  "gcpAttributes": {
                    "platform": "GCE(ESPv2)"
                  },
                  "generatedHeaderPrefix": "X-Endpoint-",
                  "requirements": [
                    {
                      "apiName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                      "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                      "serviceName": "bookstore.endpoints.project123.cloud.goog"
                    },
                    {
                      "apiKey": {
                        "allowWithoutApiKey": true
                      },
                      "apiName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                      "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_simplegetcors",
                      "serviceName": "bookstore.endpoints.project123.cloud.goog"
                    }
                  ],
                  "scCallingConfig": {
                    "networkFailOpen": true
                  },
                  "serviceControlUri": {
                    "cluster": "service-control-cluster",
                    "timeout": "30s",
                    "uri": "https://servicecontrol.googleapis.com:443/v1/services"
                  },
                  "services": [
                    {
                      "backendProtocol": "http1",
                      "jwtPayloadMetadataName": "jwt_payloads",
                      "producerProjectId": "project123",
                      "serviceConfig": {},
                      "serviceConfigId": "2017-05-01r0",
                      "serviceName": "bookstore.endpoints.project123.cloud.goog",
                      "tracingProjectId": "fake-project-id"
                    }
                  ]
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                }
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "startChildSpan": true
                }
              }
            ],
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress Simplegetcors"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/simplegetcors"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Simplegetcors"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/simplegetcors/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ESPv2_Autogenerated_CORS_Simplegetcors"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"OPTIONS"},
                            "name": ":method"
                          }
                        ],
                        "path": "/simplegetcors"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_Simplegetcors",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_simplegetcors"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress ESPv2_Autogenerated_CORS_Simplegetcors"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"OPTIONS"},
                            "name": ":method"
                          }
                        ],
                        "path": "/simplegetcors/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_Simplegetcors",
                      "route": {
                        "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_simplegetcors"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/simplegetcors"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/simplegetcors\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/simplegetcors"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/simplegetcors"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/simplegetcors\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/simplegetcors/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "tracing": {
              "clientSampling": {},
              "overallSampling": {
                "value": 0.1
              },
              "provider": {
                "name": "envoy.tracers.opencensus",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.config.trace.v3.OpenCensusConfig",
                  "incomingTraceContext": [
                    "TRACE_CONTEXT",
                    "CLOUD_TRACE_CONTEXT"
                  ],
                  "outgoingTraceContext": [
                    "TRACE_CONTEXT",
                    "CLOUD_TRACE_CONTEXT"
                  ],
                  "stackdriverExporterEnabled": true,
                  "stackdriverProjectId": "fake-project-id",
                  "traceConfig": {
                    "maxNumberOfAnnotations": "32",
                    "maxNumberOfAttributes": "32",
                    "maxNumberOfLinks": "128",
                    "maxNumberOfMessageEvents": "128"
                  }
                }
              },
              "randomSampling": {
                "value": 0.1
              }
            },
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "statPrefix": "ingress_http",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpProtocolOptions": {
              "enableTrailers": true
            },
            "useRemoteAddress": false,
            %s,
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ]
}
`, localReplyConfig)
)
