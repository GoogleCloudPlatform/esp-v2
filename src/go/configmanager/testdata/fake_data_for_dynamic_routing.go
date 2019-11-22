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

var (
	// These resources must be ordered in alphabetic order by name
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
  "name": "metadata-cluster",
  "type": "STRICT_DNS",
  "connectTimeout": "20s",
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
}`,
		`
{
  "name": "pets.appspot.com:443",
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
}`, `
{
  "name": "pets.appspot.com:8008",
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
  "name": "us-central1-cloud-esf.cloudfunctions.net:443",
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
  "name": "us-west2-cloud-esf.cloudfunctions.net:443",
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
}`,
	}

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
                        "cluster": "pets.appspot.com:443",
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
                        "cluster": "pets.appspot.com:8008",
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
                        "cluster": "us-central1-cloud-esf.cloudfunctions.net:443",
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
                        "cluster": "pets.appspot.com:443",
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
                        "cluster": "us-west2-cloud-esf.cloudfunctions.net:443",
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
