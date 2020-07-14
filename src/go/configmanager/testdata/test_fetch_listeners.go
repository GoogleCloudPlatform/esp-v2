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
	fakeProtoDescriptor    = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))
	testBackendClusterName = fmt.Sprintf("%s_local", TestFetchListenersProjectName)
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
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
	 },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_json_transcoder",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
                           "autoMapping":true,
                           "convertGrpcStatus":true,
                           "ignoredQueryParameters":[
                              "api_key",
                              "key"
                           ],
                           "printOptions":{},
                           "protoDescriptorBin":"%s",
                           "services":[
                              "%s"
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":false
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "statPrefix":"ingress_http",
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}
`,
		fakeProtoDescriptor, TestFetchListenersEndpointName, testBackendClusterName, localReplyConfig)

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
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
   },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                           "filterStateRules":{
                              "name":"com.google.espv2.filters.http.path_matcher.operation",
                              "requires":{
                                 "endpoints.examples.bookstore.Bookstore.CreateShelf":{
                                    "providerAndAudiences":{
                                       "audiences":[
                                          "test_audience1"
                                       ],
                                       "providerName":"firebase"
                                    }
                                 }
                              }
                           },
                           "providers":{
                              "firebase":{
                                 "audiences":[
                                    "test_audience1",
                                    "test_audience2"
                                 ],
                                 "forward": true,
                                 "forwardPayloadHeader":"X-Endpoint-API-UserInfo",
                                 "fromHeaders":[
                                    {
                                       "name":"Authorization",
                                       "valuePrefix":"Bearer "
                                    },
                                    {
                                       "name":"X-Goog-Iap-Jwt-Assertion"
                                    }
                                 ],
                                 "fromParams":[
                                    "access_token"
                                 ],
                                 "issuer":"https://test_issuer.google.com/",
                                 "payloadInMetadata":"jwt_payloads",
                                 "remoteJwks":{
                                    "cacheDuration":"300s",
                                    "httpUri":{
                                       "cluster":"$JWKSURI:443",
                                       "timeout":"30s",
                                       "uri":"$JWKSURI"
                                    }
                                 }
                              }
                           }
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":false
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "statPrefix":"ingress_http",
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}
              `, testBackendClusterName, localReplyConfig)

	FakeServiceConfigForGrpcWithJwtFilterWithoutAuds = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name":"%s",
                        "methods": [
                          {
                             "name": "ListShelves"
                          },
                          {
                             "name": "CreateShelf"
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
            }`, TestFetchListenersEndpointName, TestFetchListenersEndpointName)

	WantedListsenerForGrpcWithJwtFilterWithoutAuds = fmt.Sprintf(`{
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
	 },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/v1/shelves/{shelf}"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/ListShelves"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                           "filterStateRules":{
                              "name":"com.google.espv2.filters.http.path_matcher.operation",
                              "requires":{
                                 "endpoints.examples.bookstore.Bookstore.CreateShelf":{
                                    "providerName":"firebase"
                                 },
                                 "endpoints.examples.bookstore.Bookstore.ListShelves":{
                                    "providerName":"firebase"
                                 }
                              }
                           },
                           "providers":{
                              "firebase":{
                                 "audiences": [
                                     "https://bookstore.endpoints.project123.cloud.goog"
                                 ],
                                 "forward": true,
                                 "forwardPayloadHeader":"X-Endpoint-API-UserInfo",
                                 "fromHeaders":[
                                    {
                                       "name":"Authorization",
                                       "valuePrefix":"Bearer "
                                    },
                                    {
                                       "name":"X-Goog-Iap-Jwt-Assertion"
                                    }
                                 ],
                                 "fromParams":[
                                    "access_token"
                                 ],
                                 "issuer":"https://test_issuer.google.com/",
                                 "payloadInMetadata":"jwt_payloads",
                                 "remoteJwks":{
                                    "cacheDuration":"300s",
                                    "httpUri":{
                                       "cluster":"$JWKSURI:443",
                                       "timeout":"30s",
                                       "uri":"$JWKSURI"
                                    }
                                 }
                              }
                           }
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":false
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "statPrefix":"ingress_http",
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName, localReplyConfig)

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
                             "name": "GetBook"
                          },
                          {
                             "name": "DeleteBook"
                          }
                        ]
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
	WantedListenerForGrpcWithJwtFilterWithMultiReqs = fmt.Sprintf(`{
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
	 },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.DeleteBook",
                                 "pattern":{
                                    "httpMethod":"DELETE",
                                    "uriTemplate":"/v1/shelves/{shelf}/books/{book}"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.DeleteBook",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/DeleteBook"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.GetBook",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves/{shelf}/books/{book}"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.GetBook",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/GetBook"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                           "filterStateRules":{
                              "name":"com.google.espv2.filters.http.path_matcher.operation",
                              "requires":{
                                 "endpoints.examples.bookstore.Bookstore.GetBook":{
                                    "requiresAny":{
                                       "requirements":[
                                          {
                                             "providerName":"firebase1"
                                          },
                                          {
                                             "providerName":"firebase2"
                                          }
                                       ]
                                    }
                                 }
                              }
                           },
                           "providers":{
                              "firebase1":{
                                 "audiences": [
                                     "https://bookstore.endpoints.project123.cloud.goog"
                                 ],
                                 "forward": true,
                                 "forwardPayloadHeader":"X-Endpoint-API-UserInfo",
                                 "fromHeaders":[
                                    {
                                       "name":"Authorization",
                                       "valuePrefix":"Bearer "
                                    },
                                    {
                                       "name":"X-Goog-Iap-Jwt-Assertion"
                                    }
                                 ],
                                 "fromParams":[
                                    "access_token"
                                 ],
                                 "issuer":"https://test_issuer.google.com/",
                                 "payloadInMetadata":"jwt_payloads",
                                 "remoteJwks":{
                                    "cacheDuration":"300s",
                                    "httpUri":{
                                       "cluster":"$JWKSURI:443",
                                       "timeout":"30s",
                                       "uri":"$JWKSURI"
                                    }
                                 }
                              },
                              "firebase2":{
                                 "audiences": [
                                     "https://bookstore.endpoints.project123.cloud.goog"
                                 ],
                                 "forward": true,
                                 "forwardPayloadHeader":"X-Endpoint-API-UserInfo",
                                 "fromHeaders":[
                                    {
                                       "name":"Authorization",
                                       "valuePrefix":"Bearer "
                                    },
                                    {
                                       "name":"X-Goog-Iap-Jwt-Assertion"
                                    }
                                 ],
                                 "fromParams":[
                                    "access_token"
                                 ],
                                 "issuer":"https://test_issuer.google.com/",
                                 "payloadInMetadata":"jwt_payloads",
                                 "remoteJwks":{
                                    "cacheDuration":"300s",
                                    "httpUri":{
                                       "cluster":"$JWKSURI:443",
                                       "timeout":"30s",
                                       "uri":"$JWKSURI"
                                    }
                                 }
                              }
                           }
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":false
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "statPrefix":"ingress_http",
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName, localReplyConfig)

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
            }`, TestFetchListenersProjectName, TestFetchListenersEndpointName, testProjectID, TestFetchListenersEndpointName)

	WantedListenerForGrpcWithServiceControl = fmt.Sprintf(`{
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
	 },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/ListShelves"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"com.google.espv2.filters.http.service_control",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.service_control.FilterConfig",
                           "imdsToken":{
                              "cluster":"metadata-cluster",
                              "timeout":"30s",
                              "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
                           "generatedHeaderPrefix":"X-Endpoint-",
                           "requirements":[
                              {
                                 "apiName":"endpoints.examples.bookstore.Bookstore",
                                 "apiVersion":"v1",
                                 "operationName":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              },
                              {
                                 "apiName":"endpoints.examples.bookstore.Bookstore",
                                 "apiVersion":"v1",
                                 "operationName":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              }
                           ],
                           "scCallingConfig":{
                              "networkFailOpen":true
                           },
                           "serviceControlUri":{
                              "cluster":"service-control-cluster",
                              "timeout":"30s",
                              "uri":"https://servicecontrol.googleapis.com/v1/services/"
                           },
                           "services":[
                              {
                                 "backendProtocol":"grpc",
                                 "jwtPayloadMetadataName":"jwt_payloads",
                                 "producerProjectId":"%v",
                                 "serviceConfig":{
                                    "logging":{
                                       "producerDestinations":[
                                          {
                                             "logs":[
                                                "endpoints_log"
                                             ],
                                             "monitoredResource":"api"
                                          }
                                       ]
                                    },
                                    "logs":[
                                       {
                                          "name":"endpoints_log"
                                       }
                                    ]
                                 },
                                 "serviceConfigId":"%v",
                                 "serviceName":"%v"
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":false
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "statPrefix":"ingress_http",
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testProjectID, TestFetchListenersConfigID, TestFetchListenersProjectName, testBackendClusterName, localReplyConfig)

	FakeServiceConfigForHttp = fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
                "id": "2017-05-01r0",
                "apis":[
                    {
                        "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                        "methods": [
                          {
                             "name": "Echo_Auth_Jwt"
                          },
                          {
                             "name": "Echo"
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
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
   },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/echo"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/auth/info/googlejwt"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
                           "filterStateRules":{
                              "name":"com.google.espv2.filters.http.path_matcher.operation",
                              "requires":{
                                 "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt":{
                                    "providerAndAudiences":{
                                       "audiences":[
                                          "test_audience1"
                                       ],
                                       "providerName":"firebase"
                                    }
                                 }
                              }
                           },
                           "providers":{
                              "firebase":{
                                 "audiences":[
                                    "test_audience1",
                                    "test_audience2"
                                 ],
                                 "forward": true,
                                 "forwardPayloadHeader":"X-Endpoint-API-UserInfo",
                                 "fromHeaders":[
                                    {
                                       "name":"Authorization",
                                       "valuePrefix":"Bearer "
                                    },
                                    {
                                       "name":"X-Goog-Iap-Jwt-Assertion"
                                    }
                                 ],
                                 "fromParams":[
                                    "access_token"
                                 ],
                                 "issuer":"https://test_issuer.google.com/",
                                 "payloadInMetadata":"jwt_payloads",
                                 "remoteJwks":{
                                    "cacheDuration":"300s",
                                    "httpUri":{
                                       "cluster":"$JWKSURI:443",
                                       "timeout":"30s",
                                       "uri":"$JWKSURI"
                                    }
                                 }
                              }
                           }
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                       		 "suppressEnvoyHeaders": true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"%s",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "statPrefix":"ingress_http",
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName, localReplyConfig)

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
   "address":{
      "socketAddress":{
         "address":"0.0.0.0",
         "portValue":8080
      }
	 },
	 "name": "ingress_listener",
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.filters.network.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"com.google.espv2.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_simplegetcors",
                                 "pattern":{
                                    "httpMethod":"OPTIONS",
                                    "uriTemplate":"/simplegetcors"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/simplegetcors"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"com.google.espv2.filters.http.service_control",
                        "typedConfig":{
                           "@type":"type.googleapis.com/espv2.api.envoy.v7.http.service_control.FilterConfig",
                           "imdsToken":{
                              "cluster":"metadata-cluster",
                              "timeout":"30s",
                              "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
                           "generatedHeaderPrefix":"X-Endpoint-",
                           "requirements":[
                              {
                                 "apiKey":{
                                    "allowWithoutApiKey":true
                                 },
                                 "apiName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                                 "operationName":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_simplegetcors",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              },
                              {
                                 "apiName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
                                 "operationName":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              }
                           ],
                           "scCallingConfig":{
                              "networkFailOpen":true
                           },
                           "serviceControlUri":{
                              "cluster":"service-control-cluster",
                              "timeout":"30s",
                              "uri":"https://servicecontrol.googleapis.com/v1/services/"
                           },
                           "services":[
                              {
                                 "backendProtocol":"http1",
                                 "jwtPayloadMetadataName":"jwt_payloads",
                                 "producerProjectId":"project123",
                                 "serviceConfig":{},
                                 "serviceConfigId":"2017-05-01r0",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                           "startChildSpan":true
                        }
                     }
                  ],
                  "routeConfig":{
                     "name":"local_route",
                     "virtualHosts":[
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
                                    "cluster":"bookstore.endpoints.project123.cloud.goog_local",
                                    "timeout":"15s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "upgradeConfigs": [{"upgradeType": "websocket"}],
                  "statPrefix":"ingress_http",
                  "commonHttpProtocolOptions":{"headersWithUnderscoresAction":"REJECT_REQUEST"},
                  "tracing":{

                  },
                  "useRemoteAddress":false,
                  %s,
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, localReplyConfig)
)
