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

package filtergen_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen/filtergentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control"
	"github.com/google/go-cmp/cmp"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestNewServiceControlFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testData := []struct {
		filtergentest.SuccessOPTestCase
		FactoryParamsIn filtergen.ServiceControlOPFactoryParams
	}{
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, get access token from imds",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "imdsToken":{
         "cluster":"metadata-cluster",
         "timeout":"30s",
         "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
      },
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, get access token from iam",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
				},
				OptsIn: options.ConfigGeneratorOptions{
					CommonOptions: options.CommonOptions{
						ServiceControlCredentials: &options.IAMCredentialsOptions{
							ServiceAccountEmail: "ServiceControl@iam.com",
							Delegates:           []string{"delegate_foo", "delegate_bar"},
						},
					},
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "iamToken":{
         "accessToken":{
            "remoteToken":{
               "cluster":"metadata-cluster",
               "timeout":"30s",
               "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
            }
         },
         "delegates":[
            "delegate_foo",
            "delegate_bar"
         ],
         "iamUri":{
            "cluster":"iam-cluster",
            "timeout":"30s",
            "uri":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/ServiceControl@iam.com:generateAccessToken"
         },
         "serviceAccountEmail":"ServiceControl@iam.com"
      },
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, get access token from the token agent server",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
				},
				OptsIn: options.ConfigGeneratorOptions{
					ServiceAccountKey: "this-is-sa-cred",
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "imdsToken":{
         "cluster":"token-agent-cluster",
         "timeout":"30s",
         "uri":"http://127.0.0.1:8791/local/access_token"
      },
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			FactoryParamsIn: filtergen.ServiceControlOPFactoryParams{
				GCPAttributes: &scpb.GcpAttributes{
					ProjectId: "cloudesf-tenant-project",
					Zone:      "us-central1c",
					Platform:  "Cloud Run",
				},
			},
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, test various options",
				ServiceConfigIn: &servicepb.Service{
					Name:              "test-bookstore.endpoints.project123.cloud.goog",
					Id:                "2023-05-05r1",
					ProducerProjectId: "cloudesf-testing",
					Control: &servicepb.Control{
						Environment: "staging-servicecontrol.googleapis.com",
					},
				},
				OptsIn: options.ConfigGeneratorOptions{
					CommonOptions: options.CommonOptions{
						TracingOptions: &options.TracingOptions{
							ProjectId: "cloud-api-proxy-testing",
						},
						HttpRequestTimeout:    2 * time.Minute,
						GeneratedHeaderPrefix: "X-Test-Header-",
					},
					DependencyErrorBehavior:                commonpb.DependencyErrorBehavior_ALWAYS_INIT.String(),
					ClientIPFromForwardedHeader:            true,
					LogRequestHeaders:                      ":method",
					LogResponseHeaders:                     ":status",
					LogJwtPayloads:                         "my-payload",
					MinStreamReportIntervalMs:              8000,
					ComputePlatformOverride:                "ESPv2(Cloud Run)",
					ScCheckTimeoutMs:                       5020,
					ScQuotaRetries:                         8,
					ServiceControlNetworkFailOpen:          false,
					ServiceControlEnableApiKeyUidReporting: false,
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"ALWAYS_INIT",
      "generatedHeaderPrefix":"X-Test-Header-",
      "gcpAttributes":{
         "platform":"ESPv2(Cloud Run)",
         "projectId":"cloudesf-tenant-project",
         "zone":"us-central1c"
      },
      "imdsToken":{
         "cluster":"metadata-cluster",
         "timeout":"120s",
         "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
      },
      "scCallingConfig":{
         "checkTimeoutMs":5020,
         "networkFailOpen":true,
         "quotaRetries":8
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"120s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "clientIpFromForwardedHeader":true,
            "jwtPayloadMetadataName":"jwt_payloads",
            "logJwtPayloads":[
               "my-payload"
            ],
            "logRequestHeaders":[
               ":method"
            ],
            "logResponseHeaders":[
               ":status"
            ],
            "minStreamReportIntervalMs":"8000",
            "producerProjectId":"cloudesf-testing",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2023-05-05r1",
            "serviceName":"test-bookstore.endpoints.project123.cloud.goog",
            "tracingProjectId":"cloud-api-proxy-testing"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, gRPC backend",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
				},
				OptsIn: options.ConfigGeneratorOptions{
					BackendAddress: "grpc://127.0.0.0:80",
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "imdsToken":{
         "cluster":"metadata-cluster",
         "timeout":"30s",
         "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
      },
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"grpc",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "No methods, copy subset of the service config",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
					Logs: []*servicepb.LogDescriptor{
						{
							Name: "test-logs-1",
						},
					},
					Metrics: []*metricpb.MetricDescriptor{
						{
							Name: "test-metrics-1",
						},
					},
					MonitoredResources: []*monitoredrespb.MonitoredResourceDescriptor{
						{
							Name: "test-monitored-resources-1",
						},
					},
					Monitoring: &servicepb.Monitoring{
						ProducerDestinations: []*servicepb.Monitoring_MonitoringDestination{
							{
								MonitoredResource: "test-producer-dest-1",
							},
						},
					},
					Logging: &servicepb.Logging{
						ProducerDestinations: []*servicepb.Logging_LoggingDestination{
							{
								MonitoredResource: "test-producer-dest-2",
							},
						},
					},
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "imdsToken":{
         "cluster":"metadata-cluster",
         "timeout":"30s",
         "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
      },
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               "logging":{
                  "producerDestinations":[
                     {
                        "monitoredResource":"test-producer-dest-2"
                     }
                  ]
               },
               "logs":[
                  {
                     "name":"test-logs-1"
                  }
               ],
               "metrics":[
                  {
                     "name":"test-metrics-1"
                  }
               ],
               "monitoredResources":[
                  {
                     "name":"test-monitored-resources-1"
                  }
               ],
               "monitoring":{
                  "producerDestinations":[
                     {
                        "monitoredResource":"test-producer-dest-1"
                     }
                  ]
               }
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
		{
			SuccessOPTestCase: filtergentest.SuccessOPTestCase{
				Desc: "Success with some method requirements",
				ServiceConfigIn: &servicepb.Service{
					Name: "bookstore.endpoints.project123.cloud.goog",
					Id:   "2019-03-02r0",
					Control: &servicepb.Control{
						Environment: "servicecontrol.googleapis.com",
					},
					Apis: []*apipb.Api{
						{
							Name:    "google.library.Bookstore",
							Version: "2.0.0",
							Methods: []*apipb.Method{
								{
									Name: "GetShelves",
								},
								{
									Name: "GetBooks",
								},
							},
						},
						{
							// Ignored by default.
							Name:    "google.discovery",
							Version: "1.0.0",
							Methods: []*apipb.Method{
								{
									Name: "GetDiscoveryRest",
								},
							},
						},
					},
					Quota: &servicepb.Quota{
						MetricRules: []*servicepb.MetricRule{
							{
								Selector: "google.library.Bookstore.GetBooks",
								MetricCosts: map[string]int64{
									"metric_a": 2,
									"metric_b": 1,
								},
							},
						},
					},
				},
				WantFilterConfigs: []string{`
{
   "name":"com.google.espv2.filters.http.service_control",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.service_control.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "generatedHeaderPrefix":"X-Endpoint-",
      "imdsToken":{
         "cluster":"metadata-cluster",
         "timeout":"30s",
         "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
      },
      "requirements":[
         {
            "apiName":"google.library.Bookstore",
            "apiVersion":"2.0.0",
            "operationName":"google.library.Bookstore.GetShelves",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         },
         {
            "apiName":"google.library.Bookstore",
            "apiVersion":"2.0.0",
            "metricCosts":[
               {
                  "cost":"2",
                  "name":"metric_a"
               },
               {
                  "cost":"1",
                  "name":"metric_b"
               }
            ],
            "operationName":"google.library.Bookstore.GetBooks",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ],
      "scCallingConfig":{
         "networkFailOpen":true
      },
      "serviceControlUri":{
         "cluster":"service-control-cluster",
         "timeout":"30s",
         "uri":"https://servicecontrol.googleapis.com:443/v1/services"
      },
      "services":[
         {
            "backendProtocol":"http1",
            "jwtPayloadMetadataName":"jwt_payloads",
            "serviceConfig":{
               
            },
            "serviceConfigId":"2019-03-02r0",
            "serviceName":"bookstore.endpoints.project123.cloud.goog"
         }
      ]
   }
}
`,
				},
			},
		},
	}
	for _, tc := range testData {
		tc.RunTest(t, func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]filtergen.FilterGenerator, error) {
			return filtergen.NewServiceControlFilterGensFromOPConfig(serviceConfig, opts, tc.FactoryParamsIn)
		})
	}
}

func TestParseServiceControlURLFromOPConfig(t *testing.T) {
	testData := []struct {
		desc                  string
		serviceConfigIn       *servicepb.Service
		optionsIn             options.ConfigGeneratorOptions
		wantServiceControlURI url.URL
	}{
		{
			desc: "URL from service config by default",
			serviceConfigIn: &servicepb.Service{
				Control: &servicepb.Control{
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			wantServiceControlURI: url.URL{
				Scheme: "https",
				Host:   "staging-servicecontrol.sandbox.googleapis.com:443",
			},
		},
		{
			desc: "option overrides service config",
			serviceConfigIn: &servicepb.Service{
				Control: &servicepb.Control{
					// not used due to non-empty option
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			optionsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "https://servicecontrol.googleapis.com",
			},
			wantServiceControlURI: url.URL{
				Scheme: "https",
				Host:   "servicecontrol.googleapis.com:443",
			},
		},
		{
			desc:                  "Empty inputs results in empty URL",
			serviceConfigIn:       &servicepb.Service{},
			wantServiceControlURI: url.URL{},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			gotURI, err := filtergen.ParseServiceControlURLFromOPConfig(tc.serviceConfigIn, tc.optionsIn)
			if err != nil {
				t.Fatalf("ParseServiceControlURLFromOPConfig(...) got unexpecter error: %v", err)
			}

			if diff := cmp.Diff(tc.wantServiceControlURI, gotURI); diff != "" {
				t.Errorf("ParseServiceControlURLFromOPConfig(...) has unexpected diff for ServiceControlURI (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseServiceControlURLFromOPConfig_BadInput(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		optionsIn       options.ConfigGeneratorOptions
		wantErr         string
	}{
		{
			desc: "url parsing fails",
			serviceConfigIn: &servicepb.Service{
				Control: &servicepb.Control{
					Environment: "https://[::1:80",
				},
			},
			wantErr: `parse "https://[::1:80": missing ']' in host`,
		},
		{
			desc: "url should not have path segment",
			serviceConfigIn: &servicepb.Service{
				Control: &servicepb.Control{
					Environment: "https://servicecontrol.googleapis.com/v1/services",
				},
			},
			wantErr: `should not have path part: /v1/services`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := filtergen.ParseServiceControlURLFromOPConfig(tc.serviceConfigIn, tc.optionsIn)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ParseServiceControlURLFromOPConfig(...) has wrong error, got: %v, want: %q", err, tc.wantErr)
			}
		})
	}
}

func TestMakeMethodRequirementsFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc             string
		serviceConfigIn  *servicepb.Service
		optsIn           options.ConfigGeneratorOptions
		wantRequirements []*scpb.Requirement
	}{

		{
			desc: "Methods with quota config, ignore discovery",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Id:   "2019-03-02r0",
				Control: &servicepb.Control{
					Environment: "servicecontrol.googleapis.com",
				},
				Apis: []*apipb.Api{
					{
						Name:    "google.library.Bookstore",
						Version: "2.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetShelves",
							},
							{
								Name: "GetBooks",
							},
						},
					},
					{
						// Ignored by default.
						Name:    "google.discovery",
						Version: "1.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
				Quota: &servicepb.Quota{
					MetricRules: []*servicepb.MetricRule{
						{
							Selector: "google.library.Bookstore.GetBooks",
							MetricCosts: map[string]int64{
								"metric_a": 2,
								"metric_b": 1,
							},
						},
					},
				},
			},
			wantRequirements: []*scpb.Requirement{
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetShelves",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					MetricCosts: []*scpb.MetricCost{
						{
							Name: "metric_a",
							Cost: 2,
						},
						{
							Name: "metric_b",
							Cost: 1,
						},
					},
				},
			},
		},
		{
			desc: "Methods with usage rules",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Id:   "2019-03-02r0",
				Control: &servicepb.Control{
					Environment: "servicecontrol.googleapis.com",
				},
				Apis: []*apipb.Api{
					{
						Name:    "google.library.Bookstore",
						Version: "2.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetShelves",
							},
							{
								Name: "GetBooks",
							},
						},
					},
				},
				Usage: &servicepb.Usage{
					Rules: []*servicepb.UsageRule{
						{
							Selector:           "google.library.Bookstore.GetShelves",
							SkipServiceControl: true,
						},
						{
							Selector:               "google.library.Bookstore.GetBooks",
							AllowUnregisteredCalls: true,
						},
					},
				},
			},
			wantRequirements: []*scpb.Requirement{
				{
					ServiceName:        "bookstore.endpoints.project123.cloud.goog",
					OperationName:      "google.library.Bookstore.GetShelves",
					ApiName:            "google.library.Bookstore",
					ApiVersion:         "2.0.0",
					SkipServiceControl: true,
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					ApiKey: &scpb.ApiKeyRequirement{
						AllowWithoutApiKey: true,
					},
				},
			},
		},
		{
			desc: "Methods with API Key system parameters",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Id:   "2019-03-02r0",
				Control: &servicepb.Control{
					Environment: "servicecontrol.googleapis.com",
				},
				Apis: []*apipb.Api{
					{
						Name:    "google.library.Bookstore",
						Version: "2.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetShelves",
							},
							{
								Name: "GetBooks",
							},
						},
					},
				},
				SystemParameters: &servicepb.SystemParameters{
					Rules: []*servicepb.SystemParameterRule{
						{
							Selector: "google.library.Bookstore.GetShelves",
							Parameters: []*servicepb.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_1",
									UrlQueryParameter: "query_name_1",
								},
							},
						},
						{
							Selector: "google.library.Bookstore.GetBooks",
							Parameters: []*servicepb.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_2",
									UrlQueryParameter: "query_name_2",
								},
							},
						},
					},
				},
			},
			wantRequirements: []*scpb.Requirement{
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetShelves",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					ApiKey: &scpb.ApiKeyRequirement{
						Locations: []*scpb.ApiKeyLocation{
							{
								Key: &scpb.ApiKeyLocation_Query{
									Query: "query_name_1",
								},
							},
							{
								Key: &scpb.ApiKeyLocation_Header{
									Header: "header_name_1",
								},
							},
						},
					},
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					ApiKey: &scpb.ApiKeyRequirement{
						Locations: []*scpb.ApiKeyLocation{
							{
								Key: &scpb.ApiKeyLocation_Query{
									Query: "query_name_2",
								},
							},
							{
								Key: &scpb.ApiKeyLocation_Header{
									Header: "header_name_2",
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Methods with allow CORS",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Id:   "2019-03-02r0",
				Control: &servicepb.Control{
					Environment: "servicecontrol.googleapis.com",
				},
				Apis: []*apipb.Api{
					{
						Name:    "google.library.Bookstore",
						Version: "2.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetShelves",
							},
							{
								Name: "GetBooks",
							},
						},
					},
				},
				Endpoints: []*servicepb.Endpoint{
					{
						Name:      "bookstore.endpoints.project123.cloud.goog",
						AllowCors: true,
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				CorsOperationDelimiter: ".ESPv2_Autogenerated_CORS_",
			},
			wantRequirements: []*scpb.Requirement{
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetShelves",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.ESPv2_Autogenerated_CORS_GetShelves",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					ApiKey: &scpb.ApiKeyRequirement{
						AllowWithoutApiKey: true,
					},
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.ESPv2_Autogenerated_CORS_GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
					ApiKey: &scpb.ApiKeyRequirement{
						AllowWithoutApiKey: true,
					},
				},
			},
		},

		{
			desc: "Methods with healthz",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Id:   "2019-03-02r0",
				Control: &servicepb.Control{
					Environment: "servicecontrol.googleapis.com",
				},
				Apis: []*apipb.Api{
					{
						Name:    "google.library.Bookstore",
						Version: "2.0.0",
						Methods: []*apipb.Method{
							{
								Name: "GetShelves",
							},
							{
								Name: "GetBooks",
							},
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				Healthz:                                 "healthz",
				HealthCheckOperation:                    "espv2_deployment",
				HealthCheckAutogeneratedOperationPrefix: "ESPv2_Autogenerated",
			},
			wantRequirements: []*scpb.Requirement{
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetShelves",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
				},
				{
					ServiceName:   "bookstore.endpoints.project123.cloud.goog",
					OperationName: "google.library.Bookstore.GetBooks",
					ApiName:       "google.library.Bookstore",
					ApiVersion:    "2.0.0",
				},
				{
					ServiceName:        "bookstore.endpoints.project123.cloud.goog",
					OperationName:      "espv2_deployment.ESPv2_Autogenerated_HealthCheck",
					ApiName:            "espv2_deployment",
					SkipServiceControl: true,
					ApiKey: &scpb.ApiKeyRequirement{
						AllowWithoutApiKey: true,
					},
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			gotRequirements, err := filtergen.MakeMethodRequirementsFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("MakeMethodRequirementsFromOPConfig() got unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantRequirements, gotRequirements, protocmp.Transform()); diff != "" {
				t.Errorf("MakeMethodRequirementsFromOPConfig(...) has unexpected diff for requirements (-want +got):\n%s", diff)
			}
		})
	}
}
