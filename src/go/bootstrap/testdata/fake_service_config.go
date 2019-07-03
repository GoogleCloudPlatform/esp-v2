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
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/api"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	FakeConfigID = "2019-06-25r0"

	FakeBookstoreConfig = &conf.Service{
		Name:              "bookstore.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Bookstore gRPC API",
		ProducerProjectId: "producer project",
		Apis: []*api.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*api.Method{
					{
						Name:            "ListShelves",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.ListShelvesResponse",
					},
				},
			},
		},
		Http: &annotations.Http{
			Rules: []*annotations.HttpRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Pattern: &annotations.HttpRule_Get{
						Get: "/v1/shelves",
					},
				},
			},
		},
		Authentication: &conf.Authentication{
			Rules: []*conf.AuthenticationRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
							Audiences:  "bookstore_test_client.cloud.goog",
						},
					},
				},
			},
		},
		Usage: &conf.Usage{
			Rules: []*conf.UsageRule{},
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
        "clusters":[
          {
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
    	  	},
    	  	"name":"endpoints.examples.bookstore.Bookstore",
    	  	"type":"STRICT_DNS"
    	  },
    	  {
    	    "connectTimeout":"20s",
    	    "loadAssignment":{
    	      "clusterName":"metadata.google.internal",
    	      "endpoints":[
    	        {
    	          "lbEndpoints":[
    	            {
    	              "endpoint":{
    	  	            "address":{
    	   	              "socketAddress":{
    	  		  	  	    "address":"metadata.google.internal",
                            "portValue":80
    	  		  	  	  }
    	  		  	  	}
    	  		  	  }
    	            }
    	          ]
    	        }
    	      ]
    	    },
    	    "name":"metadata-cluster","type":"STRICT_DNS"
    	  }
    	],
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
    	  	  	  		  "config":{},
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
    	  }
    	]
      }
    }`
)
