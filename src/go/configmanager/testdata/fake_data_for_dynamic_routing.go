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
	FakeConfigForDynamicRouting = &confpb.Service{
		Name:              "echo-api.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Endpoints Example",
		ProducerProjectId: "producer-project",
		Apis: []*apipb.Api{
			{
				Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
				Methods: []*apipb.Method{
					{
						Name:            "Echo",
						RequestTypeUrl:  "type.googleapis.com/EchoRequest",
						ResponseTypeUrl: "type.googleapis.com/EchoMessage",
					},
					{
						Name: "dynamic_routing_Hello",
					},
					{
						Name: "dynamic_routing_Search",
					},
					{
						Name: "dynamic_routing_GetPetById",
					},
					{
						Name: "dynamic_routing_AddPet",
					},
					{
						Name: "dynamic_routing_ListPets",
					},
				},
				Version: "1.0.0",
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/echo",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/hello",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/search",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/pet/{pet_id}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/pet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/pets",
					},
				},
			},
		},
		Backend: &confpb.Backend{
			Rules: []*confpb.BackendRule{
				&confpb.BackendRule{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				// goes to cluster DynamicRouting_0
				&confpb.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
					Address:         "https://us-central1-cloud-esf.cloudfunctions.net/hello",
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://us-central1-cloud-esf.cloudfunctions.net/hello",
					},
				},
				// goes to cluster DynamicRouting_1
				&confpb.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
					Address:         "https://us-west2-cloud-esf.cloudfunctions.net/search",
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://us-west2-cloud-esf.cloudfunctions.net/search",
					},
				},
				// goes to cluster DynamicRouting_2
				&confpb.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Address:         "https://pets.appspot.com:8008/api",
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "1083071298623-e...t.apps.googleusercontent.com",
					},
				},
				// goes to cluster DynamicRouting_3
				&confpb.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
					Address:         "https://pets.appspot.com/api",
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "1083071298623-e...t.apps.googleusercontent.com",
					},
				},
				// goes to cluster DynamicRouting_3
				&confpb.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Address:         "https://pets.appspot.com/api",
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "1083071298623-e...t.apps.googleusercontent.com",
					},
				},
			},
		},
	}

	FakeWantedClustersForDynamicRouting = []string{
		`
{
  "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
  "type": "STRICT_DNS",
  "connectTimeout": "20s",
  "loadAssignment": {
    "clusterName": "127.0.0.1",
    "endpoints": [
      {
       "lbEndpoints": [
         {
           "endpoint": {
             "address": {
	              "socketAddress": {
	                "address": "127.0.0.1",
	                "portValue": 8082
	              }
	            }
	          }
          }
        ]
      }
    ]
  }
}`,
		`
{
  "name": "DynamicRouting_0",
  "type": "LOGICAL_DNS",
  "connectTimeout": "20s",
  "loadAssignment": {
    "clusterName": "us-central1-cloud-esf.cloudfunctions.net",
    "endpoints": [
      {
        "lbEndpoints": [
          {
            "endpoint": {
              "address": {
	              "socketAddress": {
	                "address": "us-central1-cloud-esf.cloudfunctions.net",
	                "portValue": 443
	              }
	            }
	          }
          }
        ]
      }
    ]
  },
  "tlsContext": {
    "sni": "us-central1-cloud-esf.cloudfunctions.net"
  }
}`,
		`
{
  "name": "DynamicRouting_1",
  "type": "LOGICAL_DNS",
  "connectTimeout": "20s",
  "loadAssignment": {
    "clusterName": "us-west2-cloud-esf.cloudfunctions.net",
    "endpoints": [{
       "lbEndpoints": [{
         "endpoint": {
          "address": {
	    "socketAddress": {
	      "address": "us-west2-cloud-esf.cloudfunctions.net",
	      "portValue": 443
	    }
	  }
	}
      }]
   }]
  },
  "tlsContext": {
    "sni": "us-west2-cloud-esf.cloudfunctions.net"
  }
}`, `
{
  "name": "DynamicRouting_2",
  "type": "LOGICAL_DNS",
  "connectTimeout": "20s",
  "loadAssignment":{
    "clusterName":"pets.appspot.com",
    "endpoints":[
      {
        "lbEndpoints":[
          {
            "endpoint":{
              "address":{
	              "socketAddress":{
	                "address":"pets.appspot.com",
	                "portValue":8008
	              }
	            }
	          }
          }
        ]
      }
    ]
  },
  "tlsContext": {
    "sni": "pets.appspot.com"
  }
}`,
		`
{
  "name": "DynamicRouting_3",
  "type": "LOGICAL_DNS",
  "connectTimeout": "20s",
  "loadAssignment":{
    "clusterName":"pets.appspot.com",
    "endpoints":[
      {
        "lbEndpoints":[
          {
            "endpoint":{
              "address":{
	              "socketAddress":{
	                "address":"pets.appspot.com",
	                "portValue":443
	              }
	            }
	          }
          }
        ]
      }
    ]
  },
  "tlsContext": {
    "sni": "pets.appspot.com"
  }
}`}

	FakeWantedListenerForDynamicRouting = `
{
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
          "config": {
            "http_filters": [
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
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                      "pattern": {
                        "http_method": "POST",
                        "uri_template": "/pet"
                      }
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
                      "pattern": {
                        "http_method": "GET",
                        "uri_template": "/pet/{pet_id}"
                      }
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                      "pattern": {
                        "http_method": "GET",
                        "uri_template": "/hello"
                      }
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                      "pattern": {
                        "http_method": "GET",
                        "uri_template": "/pets"
                      }
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                      "pattern": {
                        "http_method": "GET",
                        "uri_template": "/search"
                      }
                    }
                  ]
                },
                "name": "envoy.filters.http.path_matcher"
              },
              {
                "config": {
                  "rules": [
                     {
                      "jwt_audience": "1083071298623-e...t.apps.googleusercontent.com",
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet"
                    },
                    {
                      "jwt_audience": "1083071298623-e...t.apps.googleusercontent.com",
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById"
                    },
                    {
                      "jwt_audience": "https://us-central1-cloud-esf.cloudfunctions.net/hello",
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello"
                    },
                    {
                      "jwt_audience": "1083071298623-e...t.apps.googleusercontent.com",
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets"
                    },
                    {
                      "jwt_audience": "https://us-west2-cloud-esf.cloudfunctions.net/search",
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search"
                    }
                  ],
                  "imds_token":{
                    "imds_server_uri":{
                      "cluster":"metadata-cluster",
                      "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity",
                      "timeout":"5s"
                    }
                  }
                },
                "name": "envoy.filters.http.backend_auth"
              },
              {
                "config": {
                  "rules": [
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                      "path_prefix": "/api"
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
                      "path_prefix": "/api"
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                      "path_prefix": "/hello"
                    },
                    {
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                      "path_prefix": "/api"
                    },
                    {
                      "is_const_address": true,
                      "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                      "path_prefix": "/search"
                    }
                  ]
                },
                "name": "envoy.filters.http.backend_routing"
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
                        "headers": [
                          {
                            "exact_match": "POST",
                            "name": ":method"
                          }
                        ],
                        "path": "/pet"
                      },
                      "route": {
                        "cluster": "DynamicRouting_3",
                        "host_rewrite": "pets.appspot.com"
                      }
                    },
                    {
                      "match": {
                        "headers": [
                          {
                            "exact_match": "GET",
                            "name": ":method"
                          }
                        ],
                        "regex": "/pet/[^\\/]+$"
                      },
                      "route": {
                        "cluster": "DynamicRouting_2",
                        "host_rewrite": "pets.appspot.com"
                      }
                    },
                    {
                      "match": {
                        "headers": [
                          {
                            "exact_match": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/hello"
                      },
                      "route": {
                        "cluster": "DynamicRouting_0",
                        "host_rewrite": "us-central1-cloud-esf.cloudfunctions.net"
                      }
                    },
                    {
                      "match": {
                        "headers": [
                          {
                            "exact_match": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/pets"
                      },
                      "route": {
                        "cluster": "DynamicRouting_3",
                        "host_rewrite": "pets.appspot.com"
                      }
                    },
                    {
                      "match": {
                        "headers": [
                          {
                            "exact_match": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/search"
                      },
                      "route": {
                        "cluster": "DynamicRouting_1",
                        "host_rewrite": "us-west2-cloud-esf.cloudfunctions.net"
                      }
                    },
                    {
                      "match": {
                        "prefix": "/"
                      },
                      "route": {
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
          "name": "envoy.http_connection_manager"
        }
      ]
    }
  ]
}
`
)
