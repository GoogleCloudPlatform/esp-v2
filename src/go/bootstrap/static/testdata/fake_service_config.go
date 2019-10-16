// Copyright 2019 Google Cloud Platform Proxy Authors
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
      "cluster":"api_proxy_cluster",
      "id":"api_proxy"
   },
   "admin":{
      "accessLogPath":"/dev/null",
      "address":{
         "socketAddress":{
            "address":"0.0.0.0",
            "portValue":8001
         }
      }
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
                        "config":{
                           "http_filters":[
                              {
                                 "config":{
                                    "rules":[
                                       {
                                          "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                          "pattern":{
                                             "http_method":"GET",
                                             "uri_template":"/v1/shelves"
                                          }
                                       }
                                    ]
                                 },
                                 "name":"envoy.filters.http.path_matcher"
                              },
                              {
                                 "config":{
                                    "access_token":{
                                       "remote_token":{
                                          "cluster":"metadata-cluster",
                                          "timeout":"5s",
                                          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
                                       }
                                    },
                                    "requirements":[
                                       {
                                          "operation_name":"endpoints.examples.bookstore.Bookstore.ListShelves",
                                          "service_name":"bookstore.endpoints.cloudesf-testing.cloud.goog"
                                       }
                                    ],
                                    "sc_calling_config":{
                                       "network_fail_open":true
                                    },
                                    "service_control_uri":{
                                       "cluster":"service-control-cluster",
                                       "timeout":"5s",
                                       "uri":"https://servicecontrol.googleapis.com/v1/services/"
                                    },
                                    "services":[
                                       {
                                          "backend_protocol":"http1",
                                          "jwt_payload_metadata_name":"jwt_payloads",
                                          "producer_project_id":"producer project",
                                          "service_config":{
                                             "@type":"type.googleapis.com/google.api.Service"
                                          },
                                          "service_config_id":"2019-06-25r0",
                                          "service_name":"bookstore.endpoints.cloudesf-testing.cloud.goog"
                                       }
                                    ]
                                 },
                                 "name":"envoy.filters.http.service_control"
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
            "tlsContext":{
               "sni":"servicecontrol.googleapis.com"
            },
            "dnsLookupFamily":"V4_ONLY"
         }
      ]
   }
}
	`
)
