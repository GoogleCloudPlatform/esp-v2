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
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	FakeConfigID = "2019-06-25r0"

	FakeBookstoreConfig = &confpb.Service{
		Name:              "bookstore.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Bookstore gRPC API",
		ProducerProjectId: "producer project",
		Apis: []*apipb.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*apipb.Method{
					{
						Name:            "ListShelves",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.ListShelvesResponse",
					},
				},
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves",
					},
				},
			},
		},
		Authentication: &confpb.Authentication{
			Rules: []*confpb.AuthenticationRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: "google_service_account",
							Audiences:  "bookstore_test_client.cloud.goog",
						},
					},
				},
			},
		},
		Usage: &confpb.Usage{
			Rules: []*confpb.UsageRule{},
		},
		Control: &confpb.Control{
			Environment: "servicecontrol.googleapis.com",
		},
	}

	ExpectedBookstoreEnvoyConfig = `
{
   "node":{
      "id":"api_proxy",
      "cluster":"api_proxy_cluster"
   },
   "staticResources":{
      "listeners":[
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
                           "statPrefix":"ingress_http",
                           "routeConfig":{
                              "name":"local_route",
                              "virtualHosts":[
                                 {
                                    "name":"backend",
                                    "domains":[
                                       "*"
                                    ],
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
                           "httpFilters":[
                              {
                                 "name":"envoy.filters.http.path_matcher",
                                 "typedConfig":{
                                    "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
                                    "rules":[
                                       {
                                          "pattern":{
                                             "uriTemplate":"/v1/shelves",
                                             "httpMethod":"GET"
                                          },
                                          "operation":"endpoints.examples.bookstore.Bookstore.ListShelves"
                                       }
                                    ]
                                 }
                              },
                              {
                                 "name":"envoy.filters.http.service_control",
                                 "typedConfig":{
                                    "@type":"type.googleapis.com/google.api.envoy.http.service_control.FilterConfig",
                                    "services":[
                                       {
                                          "serviceName":"bookstore.endpoints.cloudesf-testing.cloud.goog",
                                          "serviceConfigId":"2019-06-25r0",
                                          "producerProjectId":"producer project",
                                          "serviceConfig":{
                                             "@type":"type.googleapis.com/google.api.Service"
                                          },
                                          "backendProtocol":"http1",
                                          "jwtPayloadMetadataName":"jwt_payloads"
                                       }
                                    ],
                                    "requirements":[
                                       {
                                          "serviceName":"bookstore.endpoints.cloudesf-testing.cloud.goog",
                                          "operationName":"endpoints.examples.bookstore.Bookstore.ListShelves"
                                       }
                                    ],
                                    "accessToken":{
                                       "remoteToken":{
                                          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token",
                                          "cluster":"metadata-cluster",
                                          "timeout":"5s"
                                       }
                                    },
                                    "scCallingConfig":{
                                       "networkFailOpen":true
                                    },
                                    "serviceControlUri":{
                                       "uri":"https://servicecontrol.googleapis.com/v1/services/",
                                       "cluster":"service-control-cluster",
                                       "timeout":"5s"
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
                           "useRemoteAddress":false,
                           "xffNumTrustedHops":2
                        }
                     }
                  ]
               }
            ]
         }
      ],
      "clusters":[
         {
            "name":"endpoints.examples.bookstore.Bookstore",
            "type":"STRICT_DNS",
            "connectTimeout":"20s",
            "loadAssignment":{
               "clusterName":"127.0.0.1",
               "endpoints":[
                  {
                     "lbEndpoints":[
                        {
                           "endpoint":{
                              "address":{
                                 "socketAddress":{
                                    "address":"127.0.0.1",
                                    "portValue":8082
                                 }
                              }
                           }
                        }
                     ]
                  }
               ]
            }
         },
         {
            "name":"metadata-cluster",
            "type":"STRICT_DNS",
            "connectTimeout":"20s",
            "loadAssignment":{
               "clusterName":"169.254.169.254",
               "endpoints":[
                  {
                     "lbEndpoints":[
                        {
                           "endpoint":{
                              "address":{
                                 "socketAddress":{
                                    "address":"169.254.169.254",
                                    "portValue":80
                                 }
                              }
                           }
                        }
                     ]
                  }
               ]
            }
         },
         {
            "name":"service-control-cluster",
            "type":"LOGICAL_DNS",
            "connectTimeout":"5s",
            "loadAssignment":{
               "clusterName":"servicecontrol.googleapis.com",
               "endpoints":[
                  {
                     "lbEndpoints":[
                        {
                           "endpoint":{
                              "address":{
                                 "socketAddress":{
                                    "address":"servicecontrol.googleapis.com",
                                    "portValue":443
                                 }
                              }
                           }
                        }
                     ]
                  }
               ]
            },
            "dnsLookupFamily":"V4_ONLY",
            "transportSocket":{
               "name":"tls",
               "typedConfig":{
                  "@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
                  "sni":"servicecontrol.googleapis.com"
               }
            }
         }
      ]
   },
   "admin":{
      "accessLogPath":"/dev/null",
      "address":{
         "socketAddress":{
            "address":"0.0.0.0",
            "portValue":8001
         }
      }
   }
}
	`
)
