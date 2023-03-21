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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/protobuf/types/known/anypb"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
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
                        "hostRewriteLiteral": "http.backend.test",
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
                        "hostRewriteLiteral": "http.backend.test",
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
		opts.DisableTracing = true
		opts.ConnectionBufferLimitBytes = 1024
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		listeners, err := MakeListeners(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}
		if len(listeners) != len(tc.wantListeners) {
			t.Errorf("Test Desc(%d): %s, MakeListeners failed,\ngot: %d, \nwant: %d", i, tc.desc, len(listeners), len(tc.wantListeners))
			continue
		}

		marshaler := &jsonpb.Marshaler{}
		for j, wantListener := range tc.wantListeners {
			gotListener, err := marshaler.MarshalToString(listeners[j])
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(wantListener, gotListener); err != nil {
				t.Errorf("Test Desc(%d): %s, MakeListeners failed for listener(%d), \n %v ", i, tc.desc, j, err)
			}
		}
	}
}

func TestMakeHTTPConMgr(t *testing.T) {
	testdata := []struct {
		desc             string
		opts             options.ConfigGeneratorOptions
		localReplyConfig *hcmpb.LocalReplyConfig
		wantHTTPConnMgr  string
	}{
		{
			desc: "Generate HttpConMgr with default options",
			opts: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					DisableTracing: true,
				},
			},
			wantHTTPConnMgr: `
			{
				"commonHttpProtocolOptions": {
					"headersWithUnderscoresAction": "REJECT_REQUEST"
				},
				"localReplyConfig": {
					"bodyFormat": {
						"jsonFormat": {
							"code": "%RESPONSE_CODE%",
							"message": "%LOCAL_REPLY_BODY%"
						}
					}
				},
				"normalizePath": false,
				"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
				"routeConfig": {},
				"statPrefix": "ingress_http",
				"upgradeConfigs": [
					{
						"upgradeType": "websocket"
					}
				],
				"useRemoteAddress": false
			}`,
		},
		{
			desc: "Generate HttpConMgr with custom local reply config",
			opts: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					DisableTracing: true,
				},
			},
			localReplyConfig: &hcmpb.LocalReplyConfig{
				BodyFormat: &corepb.SubstitutionFormatString{
					Format: &corepb.SubstitutionFormatString_JsonFormat{
						JsonFormat: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"foo": {
									Kind: &structpb.Value_StringValue{StringValue: "%bar%"},
								},
							},
						},
					},
				},
			},
			wantHTTPConnMgr: `
			{
				"commonHttpProtocolOptions": {
					"headersWithUnderscoresAction": "REJECT_REQUEST"
				},
				"localReplyConfig": {
					"bodyFormat": {
						"jsonFormat": {
							"foo": "%bar%"
						}
					}
				},
				"normalizePath": false,
				"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
				"routeConfig": {},
				"statPrefix": "ingress_http",
				"upgradeConfigs": [
					{
						"upgradeType": "websocket"
					}
				],
				"useRemoteAddress": false
			}`,
		},
		{
			desc: "Generate HttpConMgr when accessLog is defined",
			opts: options.ConfigGeneratorOptions{
				AccessLog:       "/foo",
				AccessLogFormat: "/bar",
				CommonOptions: options.CommonOptions{
					DisableTracing: true,
				},
			},
			wantHTTPConnMgr: `
				{
					"accessLog": [
						{
							"name": "envoy.access_loggers.file",
							"typedConfig": {
								"@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
								"path": "/foo",
								"logFormat":{"textFormat":"/bar"}
							}
						}
					],
					"commonHttpProtocolOptions": {
						"headersWithUnderscoresAction": "REJECT_REQUEST"
					},
					"localReplyConfig": {
						"bodyFormat": {
							"jsonFormat": {
								"code": "%RESPONSE_CODE%",
								"message": "%LOCAL_REPLY_BODY%"
							}
						}
					},
					"normalizePath": false,
					"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
					"routeConfig": {},
					"statPrefix": "ingress_http",
					"upgradeConfigs": [
						{
							"upgradeType": "websocket"
						}
					],
					"useRemoteAddress": false
				}
				`,
		},
		{
			desc: "Generate HttpConMgr when tracing is enabled",
			opts: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					DisableTracing:      false,
					TracingProjectId:    "test-project",
					TracingSamplingRate: 1,
				},
			},
			wantHTTPConnMgr: `
				{
					"commonHttpProtocolOptions": {
						"headersWithUnderscoresAction": "REJECT_REQUEST"
					},
					"localReplyConfig": {
						"bodyFormat": {
							"jsonFormat": {
								"code": "%RESPONSE_CODE%",
								"message": "%LOCAL_REPLY_BODY%"
							}
						}
					},
					"normalizePath": false,
					"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
					"routeConfig": {},
					"statPrefix": "ingress_http",
					"tracing":{
						"clientSampling":{},
						"overallSampling":{
							"value": 100
						},
						"provider":{
							"name":"envoy.tracers.opencensus",
							"typedConfig":{
								 "@type":"type.googleapis.com/envoy.config.trace.v3.OpenCensusConfig",
								 "stackdriverExporterEnabled":true,
								 "stackdriverProjectId":"test-project",
								 "traceConfig":{}
							}
						},
						"randomSampling":{
							"value": 100
						}
					},
					"upgradeConfigs": [
						{
							"upgradeType": "websocket"
						}
					],
					"useRemoteAddress": false
				}`,
		},
		{
			desc: "Generate HttpConMgr when UnderscoresInHeaders is defined",
			opts: options.ConfigGeneratorOptions{
				UnderscoresInHeaders: true,
				CommonOptions: options.CommonOptions{
					DisableTracing: true,
				},
			},
			wantHTTPConnMgr: `
				{
					"commonHttpProtocolOptions": {},
					"localReplyConfig": {
						"bodyFormat": {
							"jsonFormat": {
								"code": "%RESPONSE_CODE%",
								"message": "%LOCAL_REPLY_BODY%"
							}
						}
					},
					"normalizePath": false,
					"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
					"routeConfig": {},
					"statPrefix": "ingress_http",
					"upgradeConfigs": [
						{
							"upgradeType": "websocket"
						}
					],
					"useRemoteAddress": false
				}`,
		},
		{
			desc: "Generate HttpConMgr when EnableGrpcForHttp1 is defined",
			opts: options.ConfigGeneratorOptions{
				EnableGrpcForHttp1:   true,
				UnderscoresInHeaders: true,
				CommonOptions: options.CommonOptions{
					DisableTracing: true,
				},
			},
			wantHTTPConnMgr: `
				{
					"commonHttpProtocolOptions": {},
                                        "httpProtocolOptions": {"enableTrailers": true},
					"localReplyConfig": {
						"bodyFormat": {
							"jsonFormat": {
								"code": "%RESPONSE_CODE%",
								"message": "%LOCAL_REPLY_BODY%"
							}
						}
					},
					"normalizePath": false,
					"pathWithEscapedSlashesAction": "KEEP_UNCHANGED",
					"routeConfig": {},
					"statPrefix": "ingress_http",
					"upgradeConfigs": [
						{
							"upgradeType": "websocket"
						}
					],
					"useRemoteAddress": false
				}`,
		},
	}

	for _, tc := range testdata {
		routeConfig := routepb.RouteConfiguration{}
		hcm, err := makeHTTPConMgr(&tc.opts, &routeConfig, tc.localReplyConfig)
		if err != nil {
			t.Fatalf("Test (%v) failed with error: %v", tc.desc, err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotHttpConnMgr, err := marshaler.MarshalToString(hcm)
		if err != nil {
			t.Fatalf("Test (%v) failed with error: %v", tc.desc, err)
		}

		if err := util.JsonEqual(tc.wantHTTPConnMgr, gotHttpConnMgr); err != nil {
			t.Errorf("Test (%v): failed, \n %v ", tc.desc, err)
		}
	}
}

func TestMakeSchemeHeaderOverride(t *testing.T) {
	testdata := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		serverLess        bool
		want              string
	}{
		{
			desc:       "https scheme override, grpcs backend and server_less",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "grpcs://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
			want: `{"schemeToOverwrite": "https"}`,
		},
		{
			desc: "no scheme override, grpcs backend but not server_less",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "grpcs://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
		},
		{
			desc:       "no scheme override, not remote backend",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
			},
		},
		{
			desc:       "no scheme override, backend is grpc",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "grpc://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
		},
		{
			desc:       "no scheme override, backend is https",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
		},
		{
			desc:       "no scheme override, backend is http",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "http://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
		},
		{
			desc:       "https scheme override, one of grpc backends uses ssl",
			serverLess: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
							{
								Name: "Bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "grpcs://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
						{
							Address:         "grpc://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
			},
			want: `{"schemeToOverwrite": "https"}`,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			if tc.serverLess {
				opts.ComputePlatformOverride = util.ServerlessPlatform
			}
			si, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			sho := makeSchemeHeaderOverride(si)
			if sho == nil {
				if tc.want != "" {
					t.Fatalf("failed, got nil, want: %v", tc.want)
				}
			} else {
				marshaler := &jsonpb.Marshaler{}
				got, err := marshaler.MarshalToString(sho)
				if err != nil {
					t.Fatalf("failed to marshal to json with error: %v", err)
				}

				if err := util.JsonEqual(tc.want, got); err != nil {
					t.Errorf("failed, diff:\n %v ", err)
				}
			}
		})
	}
}
