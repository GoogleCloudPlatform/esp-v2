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
	"encoding/json"
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
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	pmpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	grpcstatspb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_stats/v2alpha"
	jwtauthnpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

const (
	testProjectName  = "bookstore.endpoints.project123.cloud.goog"
	testEndpointName = "endpoints.examples.bookstore.Bookstore"
	testConfigID     = "2017-05-01r0"
	testProjectID    = "project123"
	fakeJwks         = "FAKEJWKS"
	fakeToken        = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
)

var (
	fakeConfig             []byte
	fakeRollout            []byte
	fakeProtoDescriptor    = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))
	testBackendClusterName = fmt.Sprintf("%s_local", testProjectName)
)

func TestFetchListeners(t *testing.T) {
	testData := []struct {
		desc              string
		enableTracing     bool
		backendProtocol   string
		fakeServiceConfig string
		wantedListeners   string
	}{
		{
			desc:            "Success for grpc backend with transcoding",
			backendProtocol: "grpc",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
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
                        "name":"envoy.grpc_json_transcoder",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.transcoder.v2.GrpcJsonTranscoder",
                           "convertGrpcStatus":true,
                           "ignoredQueryParameters":[
                              "api_key",
                              "key",
                              "access_token"
                           ],
                           "protoDescriptorBin":"%s",
                           "services":[
                              "%s"
                           ]
                        }
                     },
                     {
                        "name":"envoy.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig",
                           "emitFilterState":true
                        }
                     },
                     {
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "statPrefix":"ingress_http",
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
			desc:            "Success for grpc backend, with Jwt filter, with audiences, no Http Rules",
			backendProtocol: "grpc",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
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
                           "@type":"type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication",
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
                        "name":"envoy.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig",
                           "emitFilterState":true
                        }
                     },
                     {
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "statPrefix":"ingress_http",
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
			desc:            "Success for gRPC backend, with Jwt filter, without audiences",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
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
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/v1/shelves/{shelf}"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/ListShelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication",
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
                        "name":"envoy.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig",
                           "emitFilterState":true
                        }
                     },
                     {
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
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
			desc:            "Success for gRPC backend, with Jwt filter, with multi requirements, matching with regex",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.DeleteBook",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/DeleteBook"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.DeleteBook",
                                 "pattern":{
                                    "httpMethod":"DELETE",
                                    "uriTemplate":"/v1/shelves/{shelf}/books/{book}"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.GetBook",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/GetBook"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.GetBook",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves/{shelf}/books/{book}"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.jwt_authn",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication",
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
                        "name":"envoy.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig",
                           "emitFilterState":true
                        }
                     },
                     {
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
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
			desc:            "Success for gRPC backend with Service Control",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
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
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/endpoints.examples.bookstore.Bookstore/ListShelves"
                                 }
                              },
                              {
                                 "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/v1/shelves"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.service_control",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.service_control.FilterConfig",
                           "accessToken":{
                              "remoteToken":{
                                 "cluster":"metadata-cluster",
                                 "timeout":"5s",
                                 "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                              }
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
                           "requirements":[
                              {
                                 "operationName":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              },
                              {
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
                        "name":"envoy.grpc_web"
                     },
                     {
                        "name":"envoy.filters.http.grpc_stats",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig",
                           "emitFilterState":true
                        }
                     },
                     {
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "statPrefix":"ingress_http",
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
			desc:            "Success for HTTP1 backend, with Jwt filter, with audiences",
			backendProtocol: "http1",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"bookstore.endpoints.project123.cloud.goog",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
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
                           "@type":"type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication",
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
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router"
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
                                    "cluster":"%s"
                                 }
                              }
                           ]
                        }
                     ]
                  },
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
			desc:            "Success for backend that allow CORS, with tracing enabled",
			enableTracing:   true,
			backendProtocol: "http1",
			fakeServiceConfig: fmt.Sprintf(`{
                "name":"%s",
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
   "filterChains":[
      {
         "filters":[
            {
               "name":"envoy.http_connection_manager",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                  "httpFilters":[
                     {
                        "name":"envoy.filters.http.path_matcher",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
                           "rules":[
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_0",
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
                           "accessToken":{
                              "remoteToken":{
                                 "cluster":"metadata-cluster",
                                 "timeout":"5s",
                                 "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                              }
                           },
                           "gcpAttributes":{
                              "platform":"GCE(ESPv2)"
                           },
                           "requirements":[
                              {
                                 "apiKey":{
                                    "allowWithoutApiKey":true
                                 },
                                 "operationName":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_0",
                                 "serviceName":"bookstore.endpoints.project123.cloud.goog"
                              },
                              {
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
                        "name":"envoy.router",
                        "typedConfig":{
                           "@type":"type.googleapis.com/envoy.config.filter.http.router.v2.Router",
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
                                    "cluster":"bookstore.endpoints.project123.cloud.goog_local"
                                 }
                              }
                           ]
                        }
                     ]
                  },
                  "statPrefix":"ingress_http",
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
		opts.BackendProtocol = tc.backendProtocol
		opts.DisableTracing = !tc.enableTracing

		flag.Set("service", testProjectName)
		flag.Set("service_config_id", testConfigID)
		flag.Set("rollout_strategy", util.FixedRolloutStrategy)
		flag.Set("check_rollout_interval", "100ms")
		flag.Set("service_json_path", "")

		runTest(t, opts, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			req := v2pb.DiscoveryRequest{
				Node: &corepb.Node{
					Id: opts.Node,
				},
				TypeUrl: cache.ListenerType,
			}
			resp, err := env.configManager.cache.Fetch(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			marshaler := &jsonpb.Marshaler{
				AnyResolver: Resolver,
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

			// Normalize both wantedListeners and gotListeners.
			gotListeners = normalizeJson(gotListeners, t)
			if want := normalizeJson(tc.wantedListeners, t); gotListeners != want {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got unexpected Listeners", i, tc.desc)
				t.Errorf("Actual: %s", gotListeners)
				t.Errorf("Expected: %s", want)
			}
		})
	}
}

func TestDynamicBackendRouting(t *testing.T) {
	testData := []struct {
		desc              string
		serviceConfigPath string
		backendProtocol   string
		wantedClusters    []string
		wantedListener    string
	}{
		{
			desc:              "Success for http1 with dynamic routing",
			serviceConfigPath: "testdata/service_config_for_dynamic_routing.json",
			backendProtocol:   "http1",
			wantedClusters:    testdata.FakeWantedClustersForDynamicRouting,
			wantedListener:    testdata.FakeWantedListenerForDynamicRouting,
		},
	}

	marshaler := &jsonpb.Marshaler{}
	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		opts.DisableTracing = true

		flag.Set("service_json_path", tc.serviceConfigPath)

		manager, err := NewConfigManager(nil, opts)
		if err != nil {
			t.Fatal("fail to initialize ConfigManager: ", err)
		}
		ctx := context.Background()
		// First request, VersionId should be empty.
		reqForClusters := v2pb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: cache.ClusterType,
		}

		respForClusters, err := manager.cache.Fetch(ctx, reqForClusters)
		if err != nil {
			t.Fatal(err)
		}

		if !proto.Equal(&respForClusters.Request, &reqForClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForClusters.Request, reqForClusters)
		}

		sortedClusters := sortResources(respForClusters)
		for idx, want := range tc.wantedClusters {
			gotCluster, err := marshaler.MarshalToString(sortedClusters[idx])
			if err != nil {
				t.Fatal(err)
			}
			gotCluster = normalizeJson(gotCluster, t)
			if want = normalizeJson(want, t); gotCluster != want {
				t.Errorf("Test Desc(%d): %s, idx %d snapshot cache fetch got Cluster: %s, want: %s", i, tc.desc, idx, gotCluster, want)
			}
		}

		reqForListener := v2pb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: cache.ListenerType,
		}

		respForListener, err := manager.cache.Fetch(ctx, reqForListener)
		if err != nil {
			t.Fatal(err)
		}
		if respForListener.Version != testConfigID {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, respForListener.Version, testConfigID)
		}
		if !proto.Equal(&respForListener.Request, &reqForListener) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForListener.Request, reqForListener)
		}

		gotListener, err := marshaler.MarshalToString(respForListener.Resources[0])
		if err != nil {
			t.Fatal(err)
		}
		gotListener = normalizeJson(gotListener, t)
		if wantListener := normalizeJson(tc.wantedListener, t); gotListener != wantListener {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got Listener: %s,\n\t want: %s", i, tc.desc, gotListener, wantListener)
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
		backendProtocol: "grpc",
	}

	// Overrides fakeConfig with fakeOldServiceConfig for the test case.
	var err error
	if fakeConfig, err = genFakeConfig(testCase.fakeOldServiceConfig); err != nil {
		t.Fatalf("genFakeConfig failed: %v", err)
	}

	if fakeRollout, err = genFakeRollout(testCase.fakeOldServiceRollout); err != nil {
		t.Fatalf("genFakeRollout failed: %v", err)
	}

	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendProtocol = testCase.backendProtocol

	flag.Set("service_config_id", testConfigID)
	flag.Set("rollout_strategy", util.ManagedRolloutStrategy)
	flag.Set("check_rollout_interval", "100ms")
	flag.Set("service_json_path", "")

	runTest(t, opts, func(env *testEnv) {
		var resp *cache.Response
		var err error
		ctx := context.Background()
		req := v2pb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
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
		if !proto.Equal(&resp.Request, &req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}

		if fakeConfig, err = genFakeConfig(testCase.fakeNewServiceConfig); err != nil {
			t.Fatalf("genFakeConfig failed: %v", err)
		}
		if fakeRollout, err = genFakeRollout(testCase.fakeNewServiceRollout); err != nil {
			t.Fatalf("genFakeRollout failed: %v", err)
		}

		time.Sleep(time.Duration(*checkNewRolloutInterval + time.Second))

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

	mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
		util.AccessTokenSuffix: fakeToken,
	})
	defer mockMetadataServer.Close()

	metadataFetcher := metadata.NewMockMetadataFetcher(mockMetadataServer.URL, time.Now())

	manager, err := NewConfigManager(metadataFetcher, opts)
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
		_, err := w.Write(fakeConfig)
		if err != nil {
			t.Fatal("fail to write config: ", err)
		}
	}))
}

func initMockRolloutServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(fakeRollout)
		if err != nil {
			t.Fatal("fail to write rollout config: ", err)
		}
	}))
}

func sortResources(response *cache.Response) []cache.Resource {
	// configManager.cache may change the order
	// sort them before comparing results.
	sortedResources := response.Resources
	sort.Slice(sortedResources, func(i, j int) bool {
		return cache.GetResourceName(sortedResources[i]) < cache.GetResourceName(sortedResources[j])
	})
	return sortedResources
}

func normalizeJson(input string, t *testing.T) string {
	var jsonObject map[string]interface{}
	err := json.Unmarshal([]byte(input), &jsonObject)
	if err != nil {
		t.Fatal("fail to normalize json: ", err)
	}
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}

func genFakeConfig(input string) ([]byte, error) {
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: Resolver,
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

func genFakeRollout(input string) ([]byte, error) {
	unmarshaler := &jsonpb.Unmarshaler{}
	rollout := new(smpb.ListServiceRolloutsResponse)
	if err := unmarshaler.Unmarshal(strings.NewReader(input), rollout); err != nil {
		return nil, err
	}

	protoBytesArray, err := proto.Marshal(rollout)
	if err != nil {
		return nil, err
	}
	return protoBytesArray, nil
}

type FuncResolver func(url string) (proto.Message, error)

func (fn FuncResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var Resolver = FuncResolver(func(url string) (proto.Message, error) {
	switch url {
	case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
		return new(smpb.ConfigFile), nil
	case "type.googleapis.com/google.api.HttpRule":
		return new(annotationspb.HttpRule), nil
	case "type.googleapis.com/google.protobuf.BoolValue":
		return new(wrapperspb.BoolValue), nil
	case "type.googleapis.com/google.api.Service":
		return new(confpb.Service), nil
	case "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager":
		return new(hcmpb.HttpConnectionManager), nil
	case "type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig":
		return new(pmpb.FilterConfig), nil
	case "type.googleapis.com/google.api.envoy.http.service_control.FilterConfig":
		return new(scpb.FilterConfig), nil
	case "type.googleapis.com/envoy.config.filter.http.router.v2.Router":
		return new(routerpb.Router), nil
	case "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext":
		return new(authpb.UpstreamTlsContext), nil
	case "type.googleapis.com/envoy.config.filter.http.transcoder.v2.GrpcJsonTranscoder":
		return new(transcoderpb.GrpcJsonTranscoder), nil
	case "type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication":
		return new(jwtauthnpb.JwtAuthentication), nil
	case "type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig":
		return new(grpcstatspb.FilterConfig), nil
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})
