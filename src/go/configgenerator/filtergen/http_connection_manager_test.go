// Copyright 2023 Google LLC
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

package filtergen_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen/filtergentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/imdario/mergo"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestNewHTTPConnectionManagerGenFromOPConfig_GenConfig(t *testing.T) {
	testdata := []filtergentest.SuccessOPTestCase{
		{
			Desc: "Generate HttpConMgr with default options",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						DisableTracing: true,
					},
				},
			},
			OptsMergeBehavior:     mergo.WithOverwriteWithEmptyValue,
			OnlyCheckFilterConfig: true,
			WantFilterConfigs: []string{
				`
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
}
`,
			},
		},
		{
			Desc: "Generate HttpConMgr when accessLog is defined",
			OptsIn: options.ConfigGeneratorOptions{
				AccessLog:       "/foo",
				AccessLogFormat: "/bar",
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						DisableTracing: true,
					},
				},
			},
			OptsMergeBehavior:     mergo.WithOverwriteWithEmptyValue,
			OnlyCheckFilterConfig: true,
			WantFilterConfigs: []string{
				`
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
		},
		{
			Desc: "Generate HttpConMgr when tracing is enabled",
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						DisableTracing: false,
						ProjectId:      "test-project",
						SamplingRate:   1,
					},
				},
			},
			OptsMergeBehavior:     mergo.WithOverwriteWithEmptyValue,
			OnlyCheckFilterConfig: true,
			WantFilterConfigs: []string{
				`
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
}
`,
			},
		},
		{
			Desc: "Generate HttpConMgr when UnderscoresInHeaders is defined",
			OptsIn: options.ConfigGeneratorOptions{
				UnderscoresInHeaders: true,
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						DisableTracing: true,
					},
				},
			},
			OptsMergeBehavior:     mergo.WithOverwriteWithEmptyValue,
			OnlyCheckFilterConfig: true,
			WantFilterConfigs: []string{
				`
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
}
`,
			},
		},
		{
			Desc: "Generate HttpConMgr when EnableGrpcForHttp1 is defined",
			OptsIn: options.ConfigGeneratorOptions{
				EnableGrpcForHttp1:   true,
				UnderscoresInHeaders: true,
				CommonOptions: options.CommonOptions{
					TracingOptions: &options.TracingOptions{
						DisableTracing: true,
					},
				},
			},
			OptsMergeBehavior:     mergo.WithOverwriteWithEmptyValue,
			OnlyCheckFilterConfig: true,
			WantFilterConfigs: []string{
				`
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
}
`,
			},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, func(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) ([]filtergen.FilterGenerator, error) {
			gen, err := filtergen.NewHTTPConnectionManagerGenFromOPConfig(serviceConfig, opts)
			if err != nil {
				return nil, err
			}

			return []filtergen.FilterGenerator{
				gen,
			}, nil
		})
	}
}

func TestIsSchemeHeaderOverrideRequiredForOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optsIn          options.ConfigGeneratorOptions
		want            bool
	}{
		{
			desc: "https scheme override, grpcs backend and server_less",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: true,
		},
		{
			desc: "no scheme override, grpcs backend but not server_less",
			serviceConfigIn: &confpb.Service{
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
			want: false,
		},
		{
			desc:            "no scheme override, not remote backend",
			serviceConfigIn: &confpb.Service{},
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: false,
		},
		{
			desc: "no scheme override, backend is grpc",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: false,
		},
		{
			desc: "no scheme override, backend is https, gRPC support not required",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: false,
		},
		{
			desc: "no scheme override, backend is http",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: false,
		},
		{
			desc: "https scheme override, one of grpc backends uses ssl",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride: util.ServerlessPlatform,
			},
			want: true,
		},
		{
			desc: "no scheme override, grpcs backend but enable backend address override",
			serviceConfigIn: &confpb.Service{
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
			optsIn: options.ConfigGeneratorOptions{
				ComputePlatformOverride:      util.ServerlessPlatform,
				EnableBackendAddressOverride: true,
			},
			want: false,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			if err := mergo.Merge(&opts, tc.optsIn); err != nil {
				t.Fatalf("Merge() of test opts into default opts got err: %v", err)
			}

			got, err := filtergen.IsSchemeHeaderOverrideRequiredForOPConfig(tc.serviceConfigIn, opts)
			if err != nil {
				t.Fatalf("IsSchemeHeaderOverrideRequiredForOPConfig() got error %v, want no error", err)
			}

			if got != tc.want {
				t.Errorf("IsSchemeHeaderOverrideRequiredForOPConfig() got %v, want %v", got, tc.want)
			}
		})
	}
}
