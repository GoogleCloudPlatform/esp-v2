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
	FakeWantedClustersForDynamicRouting = []string{`
{
        "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
	"name": "backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local",
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
		`{
                "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
		"name":"backend-cluster-pets.appspot.com:443",
		"transportSocket":{
			"name":"envoy.transport_sockets.tls",
			"typedConfig":{
				"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
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
                "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
		"name":"backend-cluster-pets.appspot.com:8008",
		"transportSocket":{
			"name":"envoy.transport_sockets.tls",
			"typedConfig":{
				"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
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
		`{
                        "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
			"name":"backend-cluster-us-central1-cloud-esf.cloudfunctions.net:443",
			"transportSocket":{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
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
		`	{
                        "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
			"name":"backend-cluster-us-west2-cloud-esf.cloudfunctions.net:443",
			"transportSocket":{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
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
		`{
                        "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
	}

	FakeWantedListenerForDynamicRouting = `
{
  "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
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
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "commonHttpProtocolOptions": {
              "headersWithUnderscoresAction": "REJECT_REQUEST"
            },
            "httpFilters": [
              {
                "name": "com.google.espv2.filters.http.backend_auth",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.FilterConfig",
                  "imdsToken": {
                    "cluster": "metadata-cluster",
                    "timeout": "30s",
                    "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
                  },
                  "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
                  "jwtAudienceList": [
                    "1083071298623-e...t.apps.googleusercontent.com",
                    "https://us-central1-cloud-esf.cloudfunctions.net/hello",
                    "https://us-west2-cloud-esf.cloudfunctions.net/search"
                  ]
                }
              },
              {
                "name": "com.google.espv2.filters.http.path_rewrite"
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber"
              },
              {
                "name": "envoy.filters.http.router",
                "typedConfig": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
                  "suppressEnvoyHeaders": true
                }
              }
            ],
            "httpProtocolOptions": {
              "enableTrailers": true
            },
           "localReplyConfig": {
              "bodyFormat": {
                "jsonFormat": {
                  "code": "%RESPONSE_CODE%",
                  "message": "%LOCAL_REPLY_BODY%"
                }
              }
            },
            "routeConfig": {
              "name": "local_route",
              "virtualHosts": [
                {
                  "domains": [
                    "*"
                  ],
                  "name": "backend",
                  "routes": [
                    {
                      "decorator": {
                        "operation": "ingress Echo"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "POST",
                            "name": ":method"
                          }
                        ],
                        "path": "/echo"
                      },
                      "route": {
                        "cluster": "backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Echo"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "POST",
                            "name": ":method"
                          }
                        ],
                        "path": "/echo/"
                      },
                      "route": {
                        "cluster": "backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_Hello"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/hello"
                      },
                      "route": {
                        "cluster": "backend-cluster-us-central1-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-central1-cloud-esf.cloudfunctions.net",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-central1-cloud-esf.cloudfunctions.net/hello"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/hello"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_Hello"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/hello/"
                      },
                      "route": {
                        "cluster": "backend-cluster-us-central1-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-central1-cloud-esf.cloudfunctions.net",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-central1-cloud-esf.cloudfunctions.net/hello"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/hello"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_AddPet"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "POST",
                            "name": ":method"
                          }
                        ],
                        "path": "/pet"
                      },
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_AddPet"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "POST",
                            "name": ":method"
                          }
                        ],
                        "path": "/pet/"
                      },
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_GetPetById"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "safeRegex": {
                          "googleRe2": {},
                          "regex": "^/pet/[^\\/]+\\/?$"
                        }
                      },
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:8008",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_ListPets"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/pets"
                      },
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_ListPets"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/pets/"
                      },
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_Search"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/search"
                      },
                      "route": {
                        "cluster": "backend-cluster-us-west2-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-west2-cloud-esf.cloudfunctions.net",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-west2-cloud-esf.cloudfunctions.net/search"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "constantPath": {
                            "path": "/search"
                          }
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_Search"
                      },
                      "match": {
                        "headers": [
                          {
                            "exactMatch": "GET",
                            "name": ":method"
                          }
                        ],
                        "path": "/search/"
                      },
                      "route": {
                        "cluster": "backend-cluster-us-west2-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-west2-cloud-esf.cloudfunctions.net",
                        "retryPolicy":{"numRetries":1,"retryOn":"reset,connect-failure,refused-stream"},
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-west2-cloud-esf.cloudfunctions.net/search"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
                          "constantPath": {
                            "path": "/search"
                          }
                        },
                        "com.google.espv2.filters.http.service_control": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
                          "operationName": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownMethod"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The request is not defined by this API."
                        },
                        "status": 404
                      },
                      "match": {
                        "prefix": "/"
                      }
                    }
                  ]
                }
              ]
            },
            "statPrefix": "ingress_http",
            "upgradeConfigs": [
              {
                "upgradeType": "websocket"
              }
            ],
            "useRemoteAddress": false,
            "xffNumTrustedHops": 2
          }
        }
      ]
    }
  ],
  "name": "ingress_listener"
}
`
)
