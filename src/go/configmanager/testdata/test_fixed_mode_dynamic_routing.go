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
	},
	"dnsLookupFamily":"V4_PREFERRED"
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
		"type":"LOGICAL_DNS",
		"dnsLookupFamily":"V4_PREFERRED"
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
		"type":"LOGICAL_DNS",
		"dnsLookupFamily":"V4_PREFERRED"
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
			"type":"LOGICAL_DNS",
			"dnsLookupFamily":"V4_PREFERRED"
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
			"type":"LOGICAL_DNS",
			"dnsLookupFamily":"V4_PREFERRED"
		}`,
		`{
			"@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
		}`,
		`{
			"@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
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
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
					"sni":"servicecontrol.googleapis.com",
					"commonTlsContext": {
						"validationContext": {
							"trustedCa": {
								"filename": "/etc/ssl/certs/ca-certificates.crt"
							}
						}
					}
				}
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
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.FilterConfig",
                  "depErrorBehavior": "BLOCK_INIT_ON_ANY_ERROR",
                  "imdsToken": {
                    "cluster": "metadata-cluster",
                    "timeout": "30s",
                    "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
                  },
                  "jwtAudienceList": [
                    "1083071298623-e...t.apps.googleusercontent.com",
                    "https://us-central1-cloud-esf.cloudfunctions.net/hello",
                    "https://us-west2-cloud-esf.cloudfunctions.net/search"
                  ]
                }
              },
              {
                "name": "com.google.espv2.filters.http.path_rewrite",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.FilterConfig"
                }
              },
              {
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig"
                 }
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
            "mergeSlashes": true,
            "normalizePath": true,
            "pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
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
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/echo"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                      "route": {
                        "cluster": "backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress Echo"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/echo/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
                      "route": {
                        "cluster": "backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress dynamic_routing_Hello"
                      },
                      "match": {
                        "headers": [
                          {
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/hello"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                      "route": {
                        "cluster": "backend-cluster-us-central1-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-central1-cloud-esf.cloudfunctions.net",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-central1-cloud-esf.cloudfunctions.net/hello"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/hello"
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/hello/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Hello",
                      "route": {
                        "cluster": "backend-cluster-us-central1-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-central1-cloud-esf.cloudfunctions.net",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-central1-cloud-esf.cloudfunctions.net/hello"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/hello"
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
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/pet"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
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
                            "stringMatch":{"exact":"POST"},
                            "name": ":method"
                          }
                        ],
                        "path": "/pet/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AddPet",
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "safeRegex": {
                          "regex": "^/pet/[^\\/]+\\/?$"
                        }
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:8008",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/pets"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/pets/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
                      "route": {
                        "cluster": "backend-cluster-pets.appspot.com:443",
                        "hostRewriteLiteral": "pets.appspot.com",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "1083071298623-e...t.apps.googleusercontent.com"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "pathPrefix": "/api"
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/search"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                      "route": {
                        "cluster": "backend-cluster-us-west2-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-west2-cloud-esf.cloudfunctions.net",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-west2-cloud-esf.cloudfunctions.net/search"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "constantPath": {
                            "path": "/search"
                          }
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
                            "stringMatch":{"exact":"GET"},
                            "name": ":method"
                          }
                        ],
                        "path": "/search/"
                      },
                      "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Search",
                      "route": {
                        "cluster": "backend-cluster-us-west2-cloud-esf.cloudfunctions.net:443",
                        "hostRewriteLiteral": "us-west2-cloud-esf.cloudfunctions.net",
                        "idleTimeout": "300s",
                        "retryPolicy": {
                          "numRetries": 1,
                          "retryOn": "reset,connect-failure,refused-stream"
                        },
                        "timeout": "15s"
                      },
                      "typedPerFilterConfig": {
                        "com.google.espv2.filters.http.backend_auth": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig",
                          "jwtAudience": "https://us-west2-cloud-esf.cloudfunctions.net/search"
                        },
                        "com.google.espv2.filters.http.path_rewrite": {
                          "@type": "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig",
                          "constantPath": {
                            "path": "/search"
                          }
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/echo"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/echo"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/echo"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/echo/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/hello"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/hello\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/hello"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/hello"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/hello\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/hello/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/pet"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/pet\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/pet"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/pet"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/pet\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/pet/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/pet/{pet_id}"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/pet/{pet_id}\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "safeRegex": {
                          "regex": "^/pet/[^\\/]+\\/?$"
                        }
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/pets"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/pets\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/pets"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/pets"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/pets\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/pets/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/search"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/search\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/search"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/search"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/search\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/search/"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownOperationName"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is not defined by this API."
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
