// Copyright 2018 Google Cloud Platform Proxy Authors
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
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/configmanager/testdata"
	"cloudesf.googlesource.com/gcpproxy/src/go/metadata"
	"cloudesf.googlesource.com/gcpproxy/src/go/options"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/googleapis/api/annotations"

	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
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
	fakeConfig          = ``
	fakeRollout         = ``
	fakeProtoDescriptor = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))
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
                                    "stat_prefix":"ingress_http",
                                    "use_remote_address":false,
                                    "xff_num_trusted_hops":2
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
                                        "filter_state_rules": {
                                          "name": "envoy.filters.http.path_matcher.operation",
                                          "requires": {
                                            "endpoints.examples.bookstore.Bookstore.CreateShelf": {
                                              "provider_and_audiences": {
                                                "audiences": [
                                                  "test_audience1"
                                                ],
                                                "provider_name": "firebase"
                                              }
                                            }
                                          }
                                        },
                                        "providers": {
                                            "firebase": {
                                                "audiences":["test_audience1", "test_audience2"],
                                                "from_headers":[{"name":"Authorization","value_prefix":"Bearer "},{"name":"X-Goog-Iap-Jwt-Assertion"}],
                                                "from_params":["access_token"],
                                                "issuer":"https://test_issuer.google.com/",
                                                "remote_jwks":{
                                                    "cache_duration":"300s",
                                                    "http_uri":{
                                                        "cluster":"https://test_issuer.google.com/",
                                                        "uri":"$JWKSURI"
                                                  }
                                                },
                                                "forward_payload_header": "X-Endpoint-API-UserInfo",
                                                "payload_in_metadata":"jwt_payloads"
                                            }
                                        }
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
                            "stat_prefix":"ingress_http",
                            "use_remote_address":false,
                            "xff_num_trusted_hops":2
                         },
                        "name":"envoy.http_connection_manager"
                    }
                ]
            }`, testEndpointName),
		},
		{
			desc:            "Success for gRPC backend, with Jwt filter, without audiences",
			backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
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
                                        "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
                                        "pattern":{
                                            "http_method":"POST",
                                            "uri_template":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                        "pattern": {
                                          "http_method": "POST",
                                          "uri_template": "/v1/shelves/{shelf}"
                                        }
                                      },
                                      {
                                         "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                         "pattern": {
                                            "http_method":"POST",
                                            "uri_template":"/endpoints.examples.bookstore.Bookstore/ListShelves"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/v1/shelves"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
                                {
                                    "config": {
                                        "filter_state_rules": {
                                          "name": "envoy.filters.http.path_matcher.operation",
                                          "requires": {
                                            "endpoints.examples.bookstore.Bookstore.CreateShelf": {
                                              "provider_name": "firebase"
                                            },
                                            "endpoints.examples.bookstore.Bookstore.ListShelves": {
                                              "provider_name": "firebase"
                                            }
                                          }
                                        },
                                        "providers": {
                                            "firebase": {
                                                "issuer":"https://test_issuer.google.com/",
                                                "from_headers":[{"name":"Authorization","value_prefix":"Bearer "},{"name":"X-Goog-Iap-Jwt-Assertion"}],
                                                "from_params":["access_token"],
                                                "remote_jwks":{
                                                    "cache_duration":"300s",
                                                    "http_uri":{
                                                        "cluster":"https://test_issuer.google.com/",
                                                        "uri":"$JWKSURI"
                                                  }
                                                },
                                                "forward_payload_header": "X-Endpoint-API-UserInfo",
                                                "payload_in_metadata":"jwt_payloads"
                                            }
                                        }
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
                            "stat_prefix":"ingress_http",
                            "use_remote_address":false,
                            "xff_num_trusted_hops":2
                        },
                        "name":"envoy.http_connection_manager"
                    }
                ]
            }`, testEndpointName),
		},
		{
			desc: "Success for gRPC backend, with Jwt filter, with multi requirements, matching with regex", backendProtocol: "gRPC",
			fakeServiceConfig: fmt.Sprintf(`{
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
                                        "operation":"endpoints.examples.bookstore.Bookstore.DeleteBook",
                                        "pattern":{
                                            "http_method":"POST",
                                            "uri_template":"/endpoints.examples.bookstore.Bookstore/DeleteBook"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.DeleteBook",
                                        "pattern": {
                                          "http_method": "DELETE",
                                          "uri_template": "/v1/shelves/{shelf}/books/{book}"
                                        }
                                      },
                                      {
                                        "operation":"endpoints.examples.bookstore.Bookstore.GetBook",
                                        "pattern":{
                                            "http_method":"POST",
                                            "uri_template":"/endpoints.examples.bookstore.Bookstore/GetBook"
                                        }
                                      },
                                      {
                                        "operation": "endpoints.examples.bookstore.Bookstore.GetBook",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/v1/shelves/{shelf}/books/{book}"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
                                {
                                    "config": {
                                        "filter_state_rules": {
                                          "name": "envoy.filters.http.path_matcher.operation",
                                          "requires": {
                                            "endpoints.examples.bookstore.Bookstore.GetBook": {
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
                                        },
                                        "providers": {
                                            "firebase1": {
                                                "issuer":"https://test_issuer.google.com/",
                                                "from_headers":[{"name":"Authorization","value_prefix":"Bearer "},{"name":"X-Goog-Iap-Jwt-Assertion"}],
                                                "from_params":["access_token"],
                                                "remote_jwks":{
                                                    "cache_duration":"300s",
                                                    "http_uri":{
                                                        "cluster":"https://test_issuer.google.com/",
                                                        "uri":"$JWKSURI"
                                                  }
                                                },
                                                "forward_payload_header": "X-Endpoint-API-UserInfo",
                                                "payload_in_metadata":"jwt_payloads"
                                            },
                                            "firebase2": {
                                                "issuer":"https://test_issuer.google.com/",
                                                "from_headers":[{"name":"Authorization","value_prefix":"Bearer "},{"name":"X-Goog-Iap-Jwt-Assertion"}],
                                                "from_params":["access_token"],
                                                "remote_jwks":{
                                                    "cache_duration":"300s",
                                                    "http_uri":{
                                                        "cluster":"https://test_issuer.google.com/",
                                                        "uri":"$JWKSURI"
                                                  }
                                                },
                                                "forward_payload_header": "X-Endpoint-API-UserInfo",
                                                "payload_in_metadata":"jwt_payloads"
                                            }
                                        }
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
                            "stat_prefix":"ingress_http",
                            "use_remote_address":false,
                            "xff_num_trusted_hops":2
                        },
                        "name":"envoy.http_connection_manager"
                    }
                ]
            }`, testEndpointName),
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
                                                "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/v1/shelves"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                                "pattern": {
                                                  "http_method": "POST",
                                                  "uri_template": "/endpoints.examples.bookstore.Bookstore/ListShelves"
                                                }
                                              },
                                              {
                                                "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
                                                "pattern": {
                                                  "http_method": "GET",
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
                                                        "jwt_payload_metadata_name": "jwt_payloads",
                                                        "service_name":"%s",
                                                        "service_config_id":"%s",
                                                        "producer_project_id":"%s",
                                                        "service_config":{
                                                           "@type":"type.googleapis.com/google.api.Service",
                                                           "logging":{"producer_destinations":[{"logs":["endpoints_log"],"monitored_resource":"api"}]},
                                                           "logs":[{"name":"endpoints_log"}]
                                                         }
                                                    }
                                                ],
                                                "service_control_uri":{
                                                            "cluster":"service-control-cluster",
                                                            "timeout":"5s",
                                                            "uri":"https://servicecontrol.googleapis.com/v1/services/"
                                                },
                                                "sc_calling_config":{"network_fail_open":true},
                                                "access_token":{
                                                  "remote_token":{
                                                    "cluster":"metadata-cluster",
                                                    "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token",
                                                    "timeout":"5s"
                                                  }
                                                }
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
                                    "stat_prefix":"ingress_http",
                                    "use_remote_address":false,
                                    "xff_num_trusted_hops":2
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
			fakeServiceConfig: `{
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
            }`,
			wantedListeners: fmt.Sprintf(`{
                "filters":[
                    {
                        "config":{
                            "http_filters":[
                                {
                                  "config": {
                                    "rules": [
                                      {
                                        "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                                        "pattern": {
                                          "http_method": "POST",
                                          "uri_template": "/echo"
                                        }
                                      },
                                      {
                                        "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
                                        "pattern": {
                                          "http_method": "GET",
                                          "uri_template": "/auth/info/googlejwt"
                                        }
                                      }
                                    ]
                                  },
                                  "name": "envoy.filters.http.path_matcher"
                                },
                                {
                                    "config": {
                                        "filter_state_rules": {
                                          "name": "envoy.filters.http.path_matcher.operation",
                                          "requires": {
                                            "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": {
                                              "provider_and_audiences": {
                                                "audiences": [
                                                  "test_audience1"
                                                ],
                                                "provider_name": "firebase"
                                              }
                                            }
                                          }
                                        },
                                        "providers": {
                                            "firebase": {
                                                "audiences":["test_audience1", "test_audience2"],
                                                "from_headers":[{"name":"Authorization","value_prefix":"Bearer "},{"name":"X-Goog-Iap-Jwt-Assertion"}],
                                                "from_params":["access_token"],
                                                "issuer":"https://test_issuer.google.com/",
                                                "remote_jwks":{
                                                    "cache_duration":"300s",
                                                    "http_uri":{
                                                        "cluster":"https://test_issuer.google.com/",
                                                        "uri":"$JWKSURI"
                                                  }
                                                },
                                                "forward_payload_header": "X-Endpoint-API-UserInfo",
                                                "payload_in_metadata":"jwt_payloads"
                                            }
                                        }
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
                                                        "cluster": "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
                                                    }
                                                }
                                            ]
                                        }
                                    ]
                                },
                            "stat_prefix":"ingress_http",
                            "use_remote_address":false,
                            "xff_num_trusted_hops":2
                        },
                        "name":"envoy.http_connection_manager"
                    }
                ]
            }`),
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
                "filters":[
                    {
                        "config": {
                            "http_filters":[
                                {
                                    "config":{
                                        "rules":[
                                            {
                                                "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_0",
                                                "pattern":{
                                                    "http_method":"OPTIONS",
                                                    "uri_template":"/simplegetcors"
                                                }
                                            },
                                            {
                                                "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                                                "pattern":{
                                                    "http_method":"GET",
                                                    "uri_template":"/simplegetcors"
                                                }
                                            }
                                        ]
                                    },
                                    "name":"envoy.filters.http.path_matcher"
                                },
                                {
                                    "config":{
                                        "gcp_attributes":{"platform":"GCE"},
                                        "requirements":[
                                             {
                                                "api_key":{
                                                    "allow_without_api_key":true
                                                },
                                                "operation_name":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_0",
                                                "service_name":"bookstore.endpoints.project123.cloud.goog"
                                            },
                                            {
                                                "operation_name":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
                                                "service_name":"bookstore.endpoints.project123.cloud.goog"
                                            }
                                        ],
                                        "services":[
                                            {
                                                "backend_protocol":"http1",
                                                "jwt_payload_metadata_name": "jwt_payloads",
                                                "producer_project_id":"project123",
                                                "service_config":{
                                                    "@type":"type.googleapis.com/google.api.Service"
                                                },
                                                "service_config_id":"2017-05-01r0",
                                                "service_name":"bookstore.endpoints.project123.cloud.goog"
                                            }
                                        ],
                                        "sc_calling_config":{"network_fail_open":true},
                                        "service_control_uri": {
                                                    "cluster":"service-control-cluster",
                                                    "timeout":"5s",
                                                    "uri":"https://servicecontrol.googleapis.com/v1/services/"
                                                },

                                        "access_token":{
                                          "remote_token":{
                                            "cluster":"metadata-cluster",
                                            "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token",
                                            "timeout":"5s"
                                          }
                                        }
                                    },
                                    "name":"envoy.filters.http.service_control"
                                },
                                {
                                    "config":{
																			"start_child_span":true
																		},
                                    "name":"envoy.router"
                                }
                            ],
                            "route_config":{
                                "name":"local_route",
                                "virtual_hosts":[
                                    {
                                        "domains":["*"],
                                        "name":"backend",
                                        "routes":[
                                            {
                                                "match":{"prefix":"/"},
                                                "route":{
                                                    "cluster":"1.echo_api_endpoints_cloudesf_testing_cloud_goog"
                                                }
                                            }
                                        ]
                                    }
                                ]
                            },
                            "stat_prefix":"ingress_http",
                            "use_remote_address":false,
                            "tracing":{},
                            "xff_num_trusted_hops":2
                        },
                        "name":"envoy.http_connection_manager"
                    }
                ]
            }`,
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.
		fakeConfig = tc.fakeServiceConfig
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		opts.EnableTracing = tc.enableTracing

		flag.Set("service", testProjectName)
		flag.Set("service_config_id", testConfigID)
		flag.Set("rollout_strategy", util.FixedRolloutStrategy)
		flag.Set("check_rollout_interval", "100ms")

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
			if !reflect.DeepEqual(resp.Request, req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			// Normalize both wantedListeners and gotListeners.
			gotListeners = normalizeJson(gotListeners, t)
			if want := normalizeJson(tc.wantedListeners, t); gotListeners != want && !strings.Contains(gotListeners, want) {
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
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		opts.EnableBackendRouting = true

		flag.Set("service", testProjectName)
		flag.Set("service_config_id", testConfigID)
		flag.Set("rollout_strategy", util.FixedRolloutStrategy)
		flag.Set("check_rollout_interval", "100ms")

		runTest(t, opts, func(env *testEnv) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			reqForClusters := v2pb.DiscoveryRequest{
				Node: &corepb.Node{
					Id: opts.Node,
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
			gotListener = normalizeJson(gotListener, t)
			if wantListener := normalizeJson(tc.wantedListener, t); gotListener != wantListener {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got Listener: %s,\n\t want: %s", i, tc.desc, gotListener, wantListener)
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

	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendProtocol = testCase.backendProtocol

	flag.Set("service_config_id", testConfigID)
	flag.Set("rollout_strategy", util.ManagedRolloutStrategy)
	flag.Set("check_rollout_interval", "100ms")

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
		if !reflect.DeepEqual(resp.Request, req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", testCase.desc, resp.Request, req)
		}

		fakeConfig = testCase.fakeNewServiceConfig
		fakeRollout = testCase.fakeNewServiceRollout
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
		if !reflect.DeepEqual(resp.Request, req) {
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
		util.ServiceAccountTokenSuffix: fakeToken,
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
		_, err := w.Write([]byte(normalizeJson(fakeConfig, t)))
		if err != nil {
			t.Fatal("fail to write config: ", err)
		}
	}))
}

func initMockRolloutServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(normalizeJson(fakeRollout, t)))
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

func marshalServiceConfigToString(serviceConfig *conf.Service, t *testing.T) string {
	m := &jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(serviceConfig)
	if err != nil {
		t.Fatal("fail to convert service config to string: ", err)
	}
	return jsonStr
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

type FuncResolver func(url string) (proto.Message, error)

func (fn FuncResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var Resolver = FuncResolver(func(url string) (proto.Message, error) {
	switch url {
	case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
		return new(sm.ConfigFile), nil
	case "type.googleapis.com/google.api.HttpRule":
		return new(annotations.HttpRule), nil
	case "type.googleapis.com/google.protobuf.BoolValue":
		return new(wrappers.BoolValue), nil
	case "type.googleapis.com/google.api.Service":
		return new(conf.Service), nil
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})
