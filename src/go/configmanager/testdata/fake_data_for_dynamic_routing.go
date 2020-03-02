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
  "name": "echo-api.endpoints.cloudesf-testing.cloud.goog_local",
  "type": "LOGICAL_DNS",
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
   "connectTimeout":"20s",
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
   "name":"pets.appspot.com:443",
   "transportSocket":{
      "name":"envoy.transport_sockets.tls",
      "typedConfig":{
         "@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
         "sni":"pets.appspot.com",
         "commonTlsContext": {
            "validationContext": {
               "trustedCa": {
                  "filename": "/etc/ssl/certs/ca-certificates.crt"
               }
            }
         }
      }
   },
   "type":"LOGICAL_DNS"
}`, `
{
   "connectTimeout":"20s",
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
   "name":"pets.appspot.com:8008",
   "transportSocket":{
      "name":"envoy.transport_sockets.tls",
      "typedConfig":{
         "@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
         "sni":"pets.appspot.com",
         "commonTlsContext": {
            "validationContext": {
               "trustedCa": {
                  "filename": "/etc/ssl/certs/ca-certificates.crt"
               }
            }
         }
      }
   },
   "type":"LOGICAL_DNS"
}`,
		`
{
   "connectTimeout":"20s",
   "loadAssignment":{
      "clusterName":"us-central1-cloud-esf.cloudfunctions.net",
      "endpoints":[
         {
            "lbEndpoints":[
               {
                  "endpoint":{
                     "address":{
                        "socketAddress":{
                           "address":"us-central1-cloud-esf.cloudfunctions.net",
                           "portValue":443
                        }
                     }
                  }
               }
            ]
         }
      ]
   },
   "name":"us-central1-cloud-esf.cloudfunctions.net:443",
   "transportSocket":{
      "name":"envoy.transport_sockets.tls",
      "typedConfig":{
         "@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
         "sni":"us-central1-cloud-esf.cloudfunctions.net",
         "commonTlsContext": {
            "validationContext": {
               "trustedCa": {
                  "filename": "/etc/ssl/certs/ca-certificates.crt"
               }
            }
         }
      }
   },
   "type":"LOGICAL_DNS"
}`,
		`
{
   "connectTimeout":"20s",
   "loadAssignment":{
      "clusterName":"us-west2-cloud-esf.cloudfunctions.net",
      "endpoints":[
         {
            "lbEndpoints":[
               {
                  "endpoint":{
                     "address":{
                        "socketAddress":{
                           "address":"us-west2-cloud-esf.cloudfunctions.net",
                           "portValue":443
                        }
                     }
                  }
               }
            ]
         }
      ]
   },
   "name":"us-west2-cloud-esf.cloudfunctions.net:443",
   "transportSocket":{
      "name":"envoy.transport_sockets.tls",
      "typedConfig":{
         "@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
         "sni":"us-west2-cloud-esf.cloudfunctions.net",
         "commonTlsContext": {
            "validationContext": {
               "trustedCa": {
                  "filename": "/etc/ssl/certs/ca-certificates.crt"
               }
            }
         }
      }
   },
   "type":"LOGICAL_DNS"
}`,
	}

	FakeWantedListenerForDynamicRouting = `
{
	 "name": "http_listener",
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
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                                 "pattern":{
                                    "httpMethod":"POST",
                                    "uriTemplate":"/pet"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/pet/{pet_id}"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/hello"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/pets"
                                 }
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                                 "pattern":{
                                    "httpMethod":"GET",
                                    "uriTemplate":"/search"
                                 }
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.backend_auth",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.backend_auth.FilterConfig",
                           "imdsToken":{
                               "cluster":"metadata-cluster",
                               "timeout":"5s",
                               "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
                           },
                           "rules":[
                              {
                                 "jwtAudience":"1083071298623-e...t.apps.googleusercontent.com",
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet"
                              },
                              {
                                 "jwtAudience":"1083071298623-e...t.apps.googleusercontent.com",
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById"
                              },
                              {
                                 "jwtAudience":"https://us-central1-cloud-esf.cloudfunctions.net/hello",
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello"
                              },
                              {
                                 "jwtAudience":"1083071298623-e...t.apps.googleusercontent.com",
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets"
                              },
                              {
                                 "jwtAudience":"https://us-west2-cloud-esf.cloudfunctions.net/search",
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search"
                              }
                           ]
                        }
                     },
                     {
                        "name":"envoy.filters.http.backend_routing",
                        "typedConfig":{
                           "@type":"type.googleapis.com/google.api.envoy.http.backend_routing.FilterConfig",
                           "rules":[
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                                 "pathPrefix":"/api"
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
                                 "pathPrefix":"/api"
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                                 "pathPrefix":"/hello"
                              },
                              {
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                                 "pathPrefix":"/api"
                              },
                              {
                                 "isConstAddress":true,
                                 "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                                 "pathPrefix":"/search"
                              }
                           ]
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
                                    "headers":[
                                       {
                                          "exactMatch":"POST",
                                          "name":":method"
                                       }
                                    ],
                                    "path":"/pet"
                                 },
                                 "route":{
                                    "cluster":"pets.appspot.com:443",
                                    "hostRewrite":"pets.appspot.com",
                                    "timeout":"15s"
                                 }
                              },
                              {
                                 "match":{
                                    "headers":[
                                       {
                                          "exactMatch":"GET",
                                          "name":":method"
                                       }
                                    ],
                                    "safeRegex":{
                                      "googleRe2":{
                                        "maxProgramSize":1000
                                      },
                                      "regex":"/pet/[^\\/]+$"
                                    }
                                 },
                                 "route":{
                                    "cluster":"pets.appspot.com:8008",
                                    "hostRewrite":"pets.appspot.com",
                                    "timeout":"15s"
                                 }
                              },
                              {
                                 "match":{
                                    "headers":[
                                       {
                                          "exactMatch":"GET",
                                          "name":":method"
                                       }
                                    ],
                                    "path":"/hello"
                                 },
                                 "route":{
                                    "cluster":"us-central1-cloud-esf.cloudfunctions.net:443",
                                    "hostRewrite":"us-central1-cloud-esf.cloudfunctions.net",
                                    "timeout":"15s"
                                 }
                              },
                              {
                                 "match":{
                                    "headers":[
                                       {
                                          "exactMatch":"GET",
                                          "name":":method"
                                       }
                                    ],
                                    "path":"/pets"
                                 },
                                 "route":{
                                    "cluster":"pets.appspot.com:443",
                                    "hostRewrite":"pets.appspot.com",
                                    "timeout":"15s"
                                 }
                              },
                              {
                                 "match":{
                                    "headers":[
                                       {
                                          "exactMatch":"GET",
                                          "name":":method"
                                       }
                                    ],
                                    "path":"/search"
                                 },
                                 "route":{
                                    "cluster":"us-west2-cloud-esf.cloudfunctions.net:443",
                                    "hostRewrite":"us-west2-cloud-esf.cloudfunctions.net",
                                    "timeout":"15s"
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
`
)
