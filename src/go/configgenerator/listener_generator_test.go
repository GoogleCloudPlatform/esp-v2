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

package configgenerator

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"google.golang.org/protobuf/types/known/anypb"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	testProjectName = "bookstore.endpoints.project123.cloud.goog"
	testApiName     = "endpoints.examples.bookstore.Bookstore"
	testConfigID    = "2019-03-02r0"
)

func TestMakeListeners(t *testing.T) {
	configFile := &smpb.ConfigFile{
		FileType: smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	data, err := anypb.New(configFile)
	if err != nil {
		t.Fatal(err)
	}
	testdata := []struct {
		desc              string
		sslServerCertPath string
		testGrpc          bool
		fakeServiceConfig *confpb.Service
		wantListeners     []string
	}{
		{
			desc:              "Success, generate redirect listener when ssl_port is configured",
			sslServerCertPath: "/etc/endpoints/ssl",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateShelf",
							},
						},
					},
				},
			},
			wantListeners: []string{`
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
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "commonHttpProtocolOptions": {},
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v12.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v12.http.grpc_metadata_scrubber.FilterConfig"
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
      ],
      "transportSocket": {
        "name": "envoy.transport_sockets.tls",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
          "commonTlsContext": {
            "alpnProtocols": [
              "h2",
              "http/1.1"
            ],
            "tlsCertificates": [
              {
                "certificateChain": {
                  "filename": "/etc/endpoints/ssl/server.crt"
                },
                "privateKey": {
                  "filename": "/etc/endpoints/ssl/server.key"
                }
              }
            ]
          }
        }
      }
    }
  ],
  "name": "ingress_listener",
  "perConnectionBufferLimitBytes": 1024
}`,
			},
		},
		{
			desc:     "Success, http backend in the backend rule",
			testGrpc: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateShelf",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/CreateShelf",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": &confpb.BackendRule{
									Address: "http://http.backend.test:8080",
								},
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{
						data,
					},
				},
			},
			wantListeners: []string{`
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
          "name": "envoy.filters.network.http_connection_manager",
          "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "commonHttpProtocolOptions": {},
            "httpFilters": [
							{
								"name": "com.google.espv2.filters.http.header_sanitizer",
								"typedConfig": {
									"@type": "type.googleapis.com/espv2.api.envoy.v12.http.header_sanitizer.FilterConfig"
								}
							},
							{
                "name": "com.google.espv2.filters.http.grpc_metadata_scrubber",
                "typedConfig": {
                  "@type": "type.googleapis.com/espv2.api.envoy.v12.http.grpc_metadata_scrubber.FilterConfig"
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
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "name": ":method",
                            "stringMatch": {
                              "exact": "GET"
                            }
                          }
                        ],
                        "path": "/CreateShelf"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-http.backend.test:8080",
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
                        "operation": "ingress CreateShelf"
                      },
                      "match": {
                        "headers": [
                          {
                            "name": ":method",
                            "stringMatch": {
                              "exact": "GET"
                            }
                          }
                        ],
                        "path": "/CreateShelf/"
                      },
                      "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
                      "route": {
                        "cluster": "backend-cluster-http.backend.test:8080",
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
                        "operation": "ingress UnknownHttpMethodForPath_/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/CreateShelf"
                      }
                    },
                    {
                      "decorator": {
                        "operation": "ingress UnknownHttpMethodForPath_/CreateShelf"
                      },
                      "directResponse": {
                        "body": {
                          "inlineString": "The current request is matched to the defined url template \"/CreateShelf\" but its http method is not allowed"
                        },
                        "status": 405
                      },
                      "match": {
                        "path": "/CreateShelf/"
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
  "name": "ingress_listener",
  "perConnectionBufferLimitBytes": 1024
}`,
			},
		},
	}

	for i, tc := range testdata {
		opts := options.DefaultConfigGeneratorOptions()
		opts.SslServerCertPath = tc.sslServerCertPath
		opts.UnderscoresInHeaders = true
		opts.CommonOptions.TracingOptions.DisableTracing = true
		opts.ConnectionBufferLimitBytes = 1024
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, opts)
		if err != nil {
			t.Fatal(err)
		}

		listeners, err := MakeListeners(fakeServiceInfo, filtergen.ServiceControlOPFactoryParams{})
		if err != nil {
			t.Fatal(err)
		}
		if len(listeners) != len(tc.wantListeners) {
			t.Errorf("Test Desc(%d): %s, MakeListeners failed,\ngot: %d, \nwant: %d", i, tc.desc, len(listeners), len(tc.wantListeners))
			continue
		}

		for j, wantListener := range tc.wantListeners {
			gotListener, err := util.ProtoToJson(listeners[j])
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(wantListener, gotListener); err != nil {
				t.Errorf("Test Desc(%d): %s, MakeListeners failed for listener(%d), \n %v ", i, tc.desc, j, err)
			}
		}
	}
}
