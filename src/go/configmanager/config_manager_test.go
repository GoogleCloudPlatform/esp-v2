// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0 //
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmanager

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

const (
	testProjectName  = "bookstore.endpoints.project123.cloud.goog"
	testEndpointName = "endpoints.examples.bookstore.Bookstore"
	testConfigID     = "2017-05-01r0"
	testProjectID    = "project123"
	fakeToken        = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
)

var (
	fakeConfig             []byte
	fakeScReport           []byte
	fakeRollouts           []byte
	fakeProtoDescriptor    = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))
	testBackendClusterName = fmt.Sprintf("%s_local", testProjectName)
)

func TestFetchListeners(t *testing.T) {
	testData := []struct {
		desc              string
		enableTracing     bool
		enableDebug       bool
		BackendAddress    string
		fakeServiceConfig string
		wantedListeners   string
	}{
		{
			desc:           "Success for grpc backend with transcoding",
			BackendAddress: "grpc://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testProjectName, testEndpointName, testEndpointName, fakeProtoDescriptor),
			wantedListeners: fmt.Sprintf(`
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                        "name":"envoy.filters.http.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig",
                           "emitFilterState":true,
                           "statsForAllMethods":true
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}
`,
				fakeProtoDescriptor, testEndpointName, testBackendClusterName),
		},
		{
			desc:           "Success for grpc backend, with Jwt filter, with audiences, no Http Rules",
			BackendAddress: "grpc://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testEndpointName, testEndpointName),

			wantedListeners: fmt.Sprintf(`
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                              "name":"envoy.filters.http.path_matcher.operation",
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
                                       "timeout":"5s",
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
                           "statsForAllMethods":true
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}
              `, testBackendClusterName),
		},
		{
			desc:           "Success for gRPC backend, with Jwt filter, without audiences",
			BackendAddress: "grpc://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testEndpointName, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                              "name":"envoy.filters.http.path_matcher.operation",
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
                                       "timeout":"5s",
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
                           "statsForAllMethods":true
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName),
		},
		{
			desc:           "Success for gRPC backend, with Jwt filter, with multi requirements, matching with regex",
			BackendAddress: "grpc://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testEndpointName, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                              "name":"envoy.filters.http.path_matcher.operation",
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
                                       "timeout":"5s",
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
                                       "timeout":"5s",
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
                           "statsForAllMethods":true
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName),
		},
		{
			desc:           "Success for gRPC backend with Service Control",
			BackendAddress: "grpc://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testProjectName, testEndpointName, testProjectID, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                        "name":"envoy.filters.http.service_control",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.service_control.FilterConfig",
                           "imdsToken":{
                              "cluster":"metadata-cluster",
                              "timeout":"5s",
                              "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
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
                              "timeout":"5s",
                              "uri":"https://servicecontrol.googleapis.com/v1/services/"
                           },
                           "services":[
                              {
                                 "backendProtocol":"grpc",
                                 "jwtPayloadMetadataName":"jwt_payloads",
                                 "producerProjectId":"%v",
                                 "serviceConfig":{
                                    "@type":"type.googleapis.com/google.api.Service",
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
                           "statsForAllMethods":true
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testProjectID, testConfigID, testProjectName, testBackendClusterName),
		},
		{
			desc:           "Success for http backend, with Jwt filter, with audiences",
			BackendAddress: "http://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testEndpointName),
			wantedListeners: fmt.Sprintf(`{
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                              "name":"envoy.filters.http.path_matcher.operation",
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
                                       "timeout":"5s",
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`, testBackendClusterName),
		},
		{
			desc:           "Success for backend that allow CORS, with tracing and debug enabled",
			enableTracing:  true,
			enableDebug:    true,
			BackendAddress: "http://127.0.0.1:80",
			fakeServiceConfig: fmt.Sprintf(`{
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
            }`, testProjectName, testProjectID, testProjectName),
			wantedListeners: `{
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
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
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
                        "name":"envoy.filters.http.service_control",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.service_control.FilterConfig",
                           "imdsToken":{
                              "cluster":"metadata-cluster",
                              "timeout":"5s",
                              "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
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
                              "timeout":"5s",
                              "uri":"https://servicecontrol.googleapis.com/v1/services/"
                           },
                           "services":[
                              {
                                 "backendProtocol":"http1",
                                 "jwtPayloadMetadataName":"jwt_payloads",
                                 "producerProjectId":"project123",
                                 "serviceConfig":{
                                    "@type":"type.googleapis.com/google.api.Service"
                                 },
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
                  "xffNumTrustedHops":2
               }
            }
         ]
      }
   ]
}`,
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		var err error
		if fakeConfig, err = genFakeConfig(tc.fakeServiceConfig); err != nil {
			t.Fatalf("genFakeConfig failed: %v", err)
		}

		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
		opts.DisableTracing = !tc.enableTracing
		opts.SuppressEnvoyHeaders = !tc.enableDebug

		_ = flag.Set("service", testProjectName)
		_ = flag.Set("service_config_id", testConfigID)
		_ = flag.Set("rollout_strategy", util.FixedRolloutStrategy)
		_ = flag.Set("check_rollout_interval", "100ms")
		_ = flag.Set("service_json_path", "")

		runTest(t, opts, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			req := discoverypb.DiscoveryRequest{
				Node: &corepb.Node{
					Id: opts.Node,
				},
				TypeUrl: resource.ListenerType,
			}
			resp, err := env.configManager.cache.Fetch(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			marshaler := &jsonpb.Marshaler{
				AnyResolver: util.Resolver,
			}
			gotListeners, err := marshaler.MarshalToString(resp.Resources[0])
			if err != nil {
				t.Fatal(err)
			}

			if resp.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, resp.Version, testConfigID)
			}
			if !proto.Equal(&resp.Request, &req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			if err := util.JsonEqual(tc.wantedListeners, gotListeners); err != nil {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got unexpected Listeners, %v", i, tc.desc, err)
			}
		})
	}
}

func TestFixedModeDynamicRouting(t *testing.T) {
	testData := []struct {
		desc              string
		serviceConfigPath string
		wantedClusters    []string
		wantedListener    string
	}{
		{
			desc:              "Success for http with dynamic routing with fixed config",
			serviceConfigPath: platform.GetFilePath(platform.FixedDrServiceConfig),
			wantedClusters:    testdata.FakeWantedClustersForDynamicRouting,
			wantedListener:    testdata.FakeWantedListenerForDynamicRouting,
		},
	}

	marshaler := &jsonpb.Marshaler{}
	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.DisableTracing = true

		_ = flag.Set("service_json_path", tc.serviceConfigPath)

		manager, err := NewConfigManager(nil, opts)
		if err != nil {
			t.Fatal("fail to initialize Config Manager: ", err)
		}
		ctx := context.Background()
		// First request, VersionId should be empty.
		reqForClusters := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ClusterType,
		}

		respForClusters, err := manager.cache.Fetch(ctx, reqForClusters)
		if err != nil {
			t.Error(err)
			continue
		}

		if !proto.Equal(&respForClusters.Request, &reqForClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForClusters.Request, reqForClusters)
			continue
		}

		sortedClusters := sortResources(respForClusters)

		if len(sortedClusters) != len(tc.wantedClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got clusters: %v, want: %v", i, tc.desc, sortedClusters, tc.wantedClusters)
			continue
		}

		for idx, want := range tc.wantedClusters {
			gotCluster, err := marshaler.MarshalToString(sortedClusters[idx])
			if err != nil {
				t.Error(err)
				continue
			}
			if err := util.JsonEqual(want, gotCluster); err != nil {
				t.Errorf("Test Desc(%d): %s, idx %d snapshot cache fetch got Cluster: \n%v", i, tc.desc, idx, err)
				continue
			}
		}

		reqForListener := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ListenerType,
		}

		respForListener, err := manager.cache.Fetch(ctx, reqForListener)
		if err != nil {
			t.Error(err)
			continue
		}
		if respForListener.Version != testConfigID {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, respForListener.Version, testConfigID)
			continue
		}
		if !proto.Equal(&respForListener.Request, &reqForListener) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForListener.Request, reqForListener)
			continue
		}

		gotListener, err := marshaler.MarshalToString(respForListener.Resources[0])
		if err != nil {
			t.Error(err)
			continue
		}
		if err := util.JsonEqual(tc.wantedListener, gotListener); err != nil {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch Listener,\n\t %v", i, tc.desc, err)
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
		fakeOldScReport       string
		fakeNewScReport       string
		fakeOldServiceRollout string
		fakeNewServiceRollout string
		fakeOldServiceConfig  string
		fakeNewServiceConfig  string
		BackendAddress        string
	}{
		desc: "Success for service config auto update",
		fakeOldScReport: fmt.Sprintf(`{
                "serviceConfigId": "%s",
                "serviceRolloutId": "%s"
            }`, oldRolloutID, oldConfigID),
		fakeNewScReport: fmt.Sprintf(`{
                "serviceConfigId": "%s",
                "serviceRolloutId": "%s"
            }`, newRolloutID, newConfigID),
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
                        "name":"%s",
                        "methods":[
                            {
                                "name": "Simplegetcors"
                            }
                        ]
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
                        "name":"%s",
                        "methods":[
                            {
                                "name": "Simplegetcors"
                            }
                        ]
                    }
                ],
                "id": "%s"
            }`, testProjectName, testEndpointName, newConfigID),
		BackendAddress: "grpc://127.0.0.1:80",
	}

	// Overrides fakeConfig with fakeOldServiceConfig for the test case.
	var err error
	if fakeScReport, err = genFakeScReport(testCase.fakeOldScReport); err != nil {
		t.Fatalf("genFakeScReport failed: %v", err)
	}

	if fakeRollouts, err = genFakeRollouts(testCase.fakeOldServiceRollout); err != nil {
		t.Fatalf("genFakeRollouts failed: %v", err)
	}

	if fakeConfig, err = genFakeConfig(testCase.fakeOldServiceConfig); err != nil {
		t.Fatalf("genFakeConfig failed: %v", err)
	}

	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendAddress = testCase.BackendAddress

	_ = flag.Set("service", testProjectName)
	_ = flag.Set("service_config_id", testConfigID)
	_ = flag.Set("rollout_strategy", util.ManagedRolloutStrategy)
	_ = flag.Set("check_rollout_interval", "100ms")
	_ = flag.Set("service_json_path", "")

	runTest(t, opts, func(env *testEnv) {
		var resp *cache.Response
		var err error
		ctx := context.Background()
		req := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ListenerType,
		}
		resp, err = env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Version != oldConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", testCase.desc, resp.Version, oldConfigID)
		}
		if !proto.Equal(&resp.Request, &req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}

		if fakeScReport, err = genFakeScReport(testCase.fakeNewScReport); err != nil {
			t.Fatalf("genFakeScReport failed: %v", err)
		}
		if fakeRollouts, err = genFakeRollouts(testCase.fakeNewServiceRollout); err != nil {
			t.Fatalf("genFakeRollouts failed: %v", err)
		}
		if fakeConfig, err = genFakeConfig(testCase.fakeNewServiceConfig); err != nil {
			t.Fatalf("genFakeConfig failed: %v", err)
		}

		time.Sleep(time.Duration(*checkNewRolloutInterval + time.Second))

		resp, err = env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Version != newConfigID || env.configManager.curConfigId() != newConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", testCase.desc, resp.Version, newConfigID)
		}

		if !proto.Equal(&resp.Request, &req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}
	})
}

// Test Environment setup.

type testEnv struct {
	configManager *ConfigManager
}

func runTest(t *testing.T, opts options.ConfigGeneratorOptions, f func(*testEnv)) {

	mockServiceControl := initMockScReportServer(t)
	defer mockServiceControl.Close()
	util.FetchRolloutIdURL = func(serviceControlUrl, serviceName string) string {
		return mockServiceControl.URL
	}

	mockRollout := initMockRolloutServer(t)
	defer mockRollout.Close()
	util.FetchRolloutsURL = func(serviceManagementUrl, serviceName string) string {
		return mockRollout.URL
	}

	mockConfig := initMockConfigServer(t)
	defer mockConfig.Close()
	util.FetchConfigURL = func(serviceManagementUrl, serviceName, configId string) string {
		return mockConfig.URL
	}

	mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
		util.AccessTokenSuffix: fakeToken,
	})
	defer mockMetadataServer.Close()

	metadataFetcher := metadata.NewMockMetadataFetcher(mockMetadataServer.URL, time.Now())

	opts.RootCertsPath = platform.GetFilePath(platform.TestRootCaCerts)
	manager, err := NewConfigManager(metadataFetcher, opts)
	if err != nil {
		t.Fatal("fail to initialize Config Manager: ", err)
	}
	env := &testEnv{
		configManager: manager,
	}
	f(env)
}

func initMockConfigServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(fakeConfig)
		if err != nil {
			t.Fatal("fail to write config: ", err)
		}
	}))
}

func initMockRolloutServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(fakeRollouts)
		if err != nil {
			t.Fatal("fail to write rollout config: ", err)
		}
	}))
}

func initMockScReportServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(fakeScReport)
		if err != nil {
			t.Fatal("fail to write service control report: ", err)
		}
	}))
}

func sortResources(response *cache.Response) []types.Resource {
	// configManager.cache may change the order
	// sort them before comparing results.
	sortedResources := response.Resources
	sort.Slice(sortedResources, func(i, j int) bool {
		return cache.GetResourceName(sortedResources[i]) < cache.GetResourceName(sortedResources[j])
	})
	return sortedResources
}

func genFakeConfig(input string) ([]byte, error) {
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: util.Resolver,
	}
	service := new(confpb.Service)
	if err := unmarshaler.Unmarshal(strings.NewReader(input), service); err != nil {
		return nil, err
	}
	protoBytesArray, err := proto.Marshal(service)
	if err != nil {
		return nil, err
	}
	return protoBytesArray, nil
}

func genFakeScReport(input string) ([]byte, error) {
	unmarshaler := &jsonpb.Unmarshaler{}
	scReport := new(servicecontrolpb.ReportResponse)
	if err := unmarshaler.Unmarshal(strings.NewReader(input), scReport); err != nil {
		return nil, err
	}

	protoBytesArray, err := proto.Marshal(scReport)
	if err != nil {
		return nil, err
	}
	return protoBytesArray, nil
}

func genFakeRollouts(input string) ([]byte, error) {
	unmarshaler := &jsonpb.Unmarshaler{}
	rollouts := new(smpb.ListServiceRolloutsResponse)
	if err := unmarshaler.Unmarshal(strings.NewReader(input), rollouts); err != nil {
		return nil, err
	}

	protoBytesArray, err := proto.Marshal(rollouts)
	if err != nil {
		return nil, err
	}
	return protoBytesArray, nil
}
