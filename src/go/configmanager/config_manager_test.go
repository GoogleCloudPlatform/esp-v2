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
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/configmanager/testdata"
	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/gogo/protobuf/jsonpb"

	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	testProjectName  = "bookstore.endpoints.project123.cloud.goog"
	testEndpointName = "endpoints.examples.bookstore.Bookstore"
	testConfigID     = "2017-05-01r0"
	testProjectID    = "project123"
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
                                                    "key",
                                                    "access_token"
                                                ],
                                                "proto_descriptor_bin":"%s",
                                                "services":[
                                                    "%s"
                                                ]
                                            },
                                            "name":"envoy.grpc_json_transcoder"
                                        },
                                        {
                                            "config":{},
                                            "name":"envoy.grpc_web"
                                        },
                                        {
                                            "config":{},
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
                                    "rules": [
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/v1/shelves"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                        "pattern": {
                                          "http_method": "POST",
                                          "uri_template": "/v1/shelves/{shelf}"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
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
                                    "rules": [
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.GetBook",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/v1/shelves/{shelf}/books/{book}"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                                        "pattern": {
                                          "http_method": "DELETE",
                                          "uri_template": "/v1/shelves/{shelf}/books/{book}"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
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
                                "logging": {
                                   "producerDestinations": [{
                                       "logs": [
                          "endpoints_log"
                       ],
                       "monitoredResource": "api"
                   }]
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
                                          "config": {
                                            "rules": [
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                                "pattern": {
                                                  "http_method": "GET",
                                                  "uri_template": "/v1/shelves"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/v1/shelves"
                                                }
                                              }
                                            ]
                                          },
                                          "name": "envoy.filters.http.path_matcher"
                                        },
                                        {
                                            "config":{
                                                "gcp_attributes":{
                                                    "platform": "GCE"
                                                },
                                                "requirements": [
                                                  {
                                                    "operation_name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                                    "service_name": "bookstore.endpoints.project123.cloud.goog"
                                                  },
                                                  {
                                                    "operation_name": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                                    "service_name": "bookstore.endpoints.project123.cloud.goog"
                                                  }
                                                ],
                                                "services":[
                                                    {
                                                        "backend_protocol": "grpc",
                                                        "service_control_uri":{
                                                            "cluster":"service-control-cluster",
                                                            "timeout":"5s",
                                                            "uri":"https://servicecontrol.googleapis.com/v1/services/"
                                                        },
                                                        "service_name":"%s",
                                                        "service_config_id":"%s",
                                                        "producer_project_id":"%s",
                                                        "token_cluster": "ads_cluster",
                                                        "service_config":{
                                                           "@type":"type.googleapis.com/google.api.Service",
                                                           "logging":{"producer_destinations":[{"logs":["endpoints_log"],"monitored_resource":"api"}]},
                                                           "logs":[{"name":"endpoints_log"}]
                                                         }
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
            }`, testProjectName, testConfigID, testProjectID),
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
                                    "rules": [
                                      {
                                        "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/auth/info/googlejwt"
                                        }
                                      },
                                      {
                                        "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                                        "pattern": {
                                          "http_method": "POST",
                                          "uri_template": "/echo"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
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
                                "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                                "pattern": {
                                "http_method": "GET",
                                "uri_template": "/simplegetcors"
                                }
                              },
                              {
                                "operation": "CORS.0",
                                "pattern": {
                                "http_method": "OPTIONS",
                                "uri_template": "/simplegetcors"
                                }
                             }
                          ]
                        },
                      "name": "envoy.filters.http.path_matcher"
                      },
                      {
                        "config": {
                          "gcp_attributes":{
                             "platform": "GCE"
                          },
                          "requirements": [
                            {
                              "operation_name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                              "service_name": "bookstore.endpoints.project123.cloud.goog"
                            },
                            {
                              "api_key": {
                                "allow_without_api_key": true
                            },
                              "operation_name": "CORS.0",
                              "service_name": "bookstore.endpoints.project123.cloud.goog"
                            }
                          ],
                          "services": [
                            {
                              "backend_protocol": "http1",
                              "producer_project_id": "project123",
                              "service_config_id": "2017-05-01r0",
                              "service_control_uri": {
                                "cluster": "service-control-cluster",
                                "timeout": "5s",
                                "uri": "https://servicecontrol.googleapis.com/v1/services/"
                              },
                              "service_name": "%s",
                              "token_cluster": "ads_cluster",
                      "service_config":{"@type":"type.googleapis.com/google.api.Service"}
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
            }`, testProjectName, testEndpointName),
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("service", testProjectName)
		flag.Set("version", testConfigID)
		flag.Set("rollout_strategy", ut.FixedRolloutStrategy)
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
			marshaler := &jsonpb.Marshaler{
				AnyResolver: ut.Resolver,
			}
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

func TestDynamicBackendRouting(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig string
		backendProtocol   string
		wantedClusters    []string
		wantedListener    string
	}{
		{
			desc:              "Success for http1 with dynamic routing",
			fakeServiceConfig: marshalServiceConfigToString(testdata.FakeConfigForDynamicRouting, t),
			backendProtocol:   "http1",
			wantedClusters:    testdata.FakeWantedClustersForDynamicRouting,
			wantedListener:    testdata.FakeWantedListenerForDynamicRouting,
		},
	}

	marshaler := &jsonpb.Marshaler{}
	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		flag.Set("service", testProjectName)
		flag.Set("version", testConfigID)
		flag.Set("rollout_strategy", ut.FixedRolloutStrategy)
		flag.Set("backend_protocol", tc.backendProtocol)
		flag.Set("enable_backend_routing", "true")

		runTest(t, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			reqForClusters := v2.DiscoveryRequest{
				Node: &core.Node{
					Id: *flags.Node,
				},
				TypeUrl: cache.ClusterType,
			}

			respForClusters, err := env.configManager.cache.Fetch(ctx, reqForClusters)
			if err != nil {
				t.Fatal(err)
			}

			if respForClusters.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, respForClusters.Version, testConfigID)
			}
			if !reflect.DeepEqual(respForClusters.Request, reqForClusters) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForClusters.Request, reqForClusters)
			}

			sortedClusters := sortResources(respForClusters)
			for idx, want := range tc.wantedClusters {
				gotCluster, err := marshaler.MarshalToString(sortedClusters[idx])
				if err != nil {
					t.Fatal(err)
				}
				gotCluster = normalizeJson(gotCluster)
				if want = normalizeJson(want); gotCluster != want {
					t.Errorf("Test Desc(%d): %s, idx %d snapshot cache fetch got Cluster: %s, want: %s", i, tc.desc, idx, gotCluster, want)
				}
			}

			reqForListener := v2.DiscoveryRequest{
				Node: &core.Node{
					Id: *flags.Node,
				},
				TypeUrl: cache.ListenerType,
			}

			respForListener, err := env.configManager.cache.Fetch(ctx, reqForListener)
			if err != nil {
				t.Fatal(err)
			}
			if respForListener.Version != testConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, respForListener.Version, testConfigID)
			}
			if !reflect.DeepEqual(respForListener.Request, reqForListener) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForListener.Request, reqForListener)
			}

			gotListener, err := marshaler.MarshalToString(respForListener.Resources[0])
			if err != nil {
				t.Fatal(err)
			}
			gotListener = normalizeJson(gotListener)
			if wantListener := normalizeJson(tc.wantedListener); gotListener != wantListener {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got Listener: %s, want: %s", i, tc.desc, gotListener, wantListener)
			}
		})
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

	runTest(t, func(env *testEnv) {
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

	mockRollout := initMockRolloutServer(t)
	defer mockRollout.Close()
	fetchRolloutsURL = func(serviceName string) string {
		return mockRollout.URL
	}

	mockMetadata := initMockMetadataServerFromPathResp(
		map[string]string{
			util.ServiceAccountTokenSuffix: fakeToken,
		})
	defer mockMetadata.Close()
	fetchMetadataURL = func(suffix string) string {
		return mockMetadata.URL + suffix
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

func sortResources(response *cache.Response) []cache.Resource {
	// configManager.cache may change the order
	// sort them before comparing results.
	sortedResources := response.Resources
	sort.Slice(sortedResources, func(i, j int) bool {
		return cache.GetResourceName(sortedResources[i]) < cache.GetResourceName(sortedResources[j])
	})
	return sortedResources
}

func marshalServiceConfigToString(serviceConfig *conf.Service, t *testing.T) string {
	m := &jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(serviceConfig)
	if err != nil {
		t.Fatal("fail to convert service config to string: ", err)
	}
	return jsonStr
}

func normalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}
