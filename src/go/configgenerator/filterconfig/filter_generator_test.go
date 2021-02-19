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

package filterconfig

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/common"
	anypb "github.com/golang/protobuf/ptypes/any"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
)

var (
	fakeProtoDescriptor = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))

	sourceFile = &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: []byte("rawDescriptor"),
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, _ = ptypes.MarshalAny(sourceFile)

	testProjectName       = "bookstore.endpoints.project123.cloud.goog"
	testApiName           = "endpoints.examples.bookstore.Bookstore"
	testServiceControlEnv = "servicecontrol.googleapis.com"
	testConfigID          = "2019-03-02r0"
)

func TestTranscoderFilter(t *testing.T) {
	testData := []struct {
		desc                                    string
		fakeServiceConfig                       *confpb.Service
		transcodingAlwaysPrintPrimitiveFields   bool
		transcodingAlwaysPrintEnumsAsInts       bool
		transcodingPreserveProtoFieldNames      bool
		transcodingIgnoreQueryParameters        string
		transcodingIgnoreUnknownQueryParameters bool
		wantTranscoderFilter                    string
	}{
		{
			desc: "Success. Generate transcoder filter with default apiKey locations and default jwt locations",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
						},
					},
				},
			},
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "ignoredQueryParameters":[
         "access_token",
         "api_key",
         "key"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "services":[
         "%s"
      ]
   }
}
      `, fakeProtoDescriptor, testApiName),
		},
		{
			desc: "Success. Generate transcoder filter with custom apiKey locations and custom jwt locations",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
							JwtLocations: []*confpb.JwtLocation{
								{
									In: &confpb.JwtLocation_Header{
										Header: "jwt_query_header",
									},
									ValuePrefix: "jwt_query_header_prefix",
								},
								{
									In: &confpb.JwtLocation_Query{
										Query: "jwt_query_param",
									},
								},
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: fmt.Sprintf("%s.Foo", testApiName),
							Parameters: []*confpb.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_1",
									UrlQueryParameter: "query_name_1",
								},
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
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "ignoredQueryParameters":[
         "jwt_query_param",
         "query_name_1",
         "query_name_2"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "services":[
         "%s"
      ]
   }
}
      `, fakeProtoDescriptor, testApiName),
		},
		{
			desc: "Success. Generate transcoder filter with print options",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
			},
			transcodingAlwaysPrintPrimitiveFields:   true,
			transcodingAlwaysPrintEnumsAsInts:       true,
			transcodingPreserveProtoFieldNames:      true,
			transcodingIgnoreQueryParameters:        "parameter_foo,parameter_bar",
			transcodingIgnoreUnknownQueryParameters: true,
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "ignoreUnknownQueryParameters":true,
      "ignoredQueryParameters":[
         "api_key",
         "key",
         "parameter_bar",
         "parameter_foo"
      ],
      "printOptions":{
         "alwaysPrintEnumsAsInts":true,
         "alwaysPrintPrimitiveFields":true,
         "preserveProtoFieldNames":true
      },
      "protoDescriptorBin":"%s",
      "services":[
         "%s"
      ]
   }
}
      `, fakeProtoDescriptor, testApiName),
		},
	}

	for i, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.0:80"
			opts.TranscodingAlwaysPrintPrimitiveFields = tc.transcodingAlwaysPrintPrimitiveFields
			opts.TranscodingPreserveProtoFieldNames = tc.transcodingPreserveProtoFieldNames
			opts.TranscodingAlwaysPrintEnumsAsInts = tc.transcodingAlwaysPrintEnumsAsInts
			opts.TranscodingIgnoreQueryParameters = tc.transcodingIgnoreQueryParameters
			opts.TranscodingIgnoreUnknownQueryParameters = tc.transcodingIgnoreUnknownQueryParameters
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			marshaler := &jsonpb.Marshaler{}
			gotFilter, err := marshaler.MarshalToString(makeTranscoderFilter(fakeServiceInfo))
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(tc.wantTranscoderFilter, gotFilter); err != nil {
				t.Errorf("Test Desc(%d): %s, makeTranscoderFilter failed, \n %v", i, tc.desc, err)
			}
		})
	}
}

func TestJwtAuthnFilter(t *testing.T) {
	testData := []struct {
		desc               string
		fakeServiceConfig  *confpb.Service
		wantJwtAuthnFilter string
	}{
		{
			desc: "Success. Generate jwt authn filter with default jwt locations",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com"
                    }
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerName": "auth_provider"
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with custom jwt locations",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
							JwtLocations: []*confpb.JwtLocation{
								{
									In: &confpb.JwtLocation_Header{
										Header: "jwt_query_header",
									},
									ValuePrefix: "jwt_query_header_prefix",
								},
								{
									In: &confpb.JwtLocation_Query{
										Query: "jwt_query_param",
									},
								},
							},
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector:               "testapi.foo",
							AllowWithoutCredential: true,
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "jwt_query_header",
                        "valuePrefix": "jwt_query_header_prefix"
                    }
                ],
                "fromParams": [
                    "jwt_query_param"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com"
                    }
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                 "requiresAny":{
                    "requirements":[
                     {
                        "providerName":"auth_provider"
                     },
                     {
                        "allowMissing":{}
                     }
                   ]
                }
            }
        }
    }
}`,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = "grpc://127.0.0.0:80"
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotProto, _, _ := jaFilterGenFunc(fakeServiceInfo)
		gotFilter, err := marshaler.MarshalToString(gotProto)
		if err != nil {
			t.Fatal(err)
		}

		if err := util.JsonEqual(tc.wantJwtAuthnFilter, gotFilter); err != nil {
			t.Errorf("Test Desc(%d): %s, makeJwtAuthnFilter failed, %s", i, tc.desc, err)
		}
	}
}

func TestBackendAuthFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		iamServiceAccount     string
		fakeServiceConfig     *confpb.Service
		delegates             []string
		depErrorBehavior      string
		wantBackendAuthFilter string
		wantError             string
	}{
		{
			desc: "Success, generate backend auth filter in general",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			depErrorBehavior: commonpb.DependencyErrorBehavior_BLOCK_INIT_ON_ANY_ERROR.String(),
			wantBackendAuthFilter: `
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["bar.com","foo.com"]
   }
}
`,
		},
		{
			desc: "Success, generate backend auth filter with allow Cors",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Apis: []*apipb.Api{
					{
						Name: "get_testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
							{
								Name: "bar",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "get_testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo",
							},
						},
						{
							Selector: "get_testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "get_testapi.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "get_testapi.bar",
							Address:         "https://testapipb.com/bar",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			depErrorBehavior: commonpb.DependencyErrorBehavior_BLOCK_INIT_ON_ANY_ERROR.String(),
			wantBackendAuthFilter: `{
        "name":"com.google.espv2.filters.http.backend_auth",
        "typedConfig":{
          "@type":"type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.FilterConfig",
          "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
          "imdsToken":{
            "cluster":"metadata-cluster",
            "timeout":"30s",
            "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
          },
          "jwtAudienceList":["bar.com","foo.com"]
        }
      }`,
		},
		{
			desc:              "Success, set iamIdToken when iam service account is set",
			iamServiceAccount: "service-account@google.com",
			delegates:         []string{"delegate_foo", "delegate_bar", "delegate_baz"},
			depErrorBehavior:  commonpb.DependencyErrorBehavior_ALWAYS_INIT.String(),
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantBackendAuthFilter: `
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.FilterConfig",
			"depErrorBehavior":"ALWAYS_INIT",
      "iamToken":{
         "accessToken":{
            "remoteToken":{
               "cluster":"metadata-cluster",
               "timeout":"30s",
               "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
            }
         },
         "iamUri":{
            "cluster":"iam-cluster",
            "timeout":"30s",
            "uri":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/service-account@google.com:generateIdToken"
         },
         "delegates":["delegate_foo","delegate_bar","delegate_baz"],
         "serviceAccountEmail":"service-account@google.com"
      },
      "jwtAudienceList":["bar.com"]
   }
}
`,
		},
		{
			desc:             "Fail when invalid dependency error behavior is provided",
			depErrorBehavior: "UNKNOWN_ERROR_BEHAVIOR",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantError: `unknown value for DependencyErrorBehavior (UNKNOWN_ERROR_BEHAVIOR), accepted values are: ["ALWAYS_INIT" "BLOCK_INIT_ON_ANY_ERROR" "UNSPECIFIED"]`,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.1:80"
			opts.DependencyErrorBehavior = tc.depErrorBehavior
			if tc.iamServiceAccount != "" {
				opts.BackendAuthCredentials = &options.IAMCredentialsOptions{
					ServiceAccountEmail: tc.iamServiceAccount,
					TokenKind:           options.IDToken,
					Delegates:           tc.delegates,
				}
			}

			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			filterConfig, _, err := baFilterGenFunc(fakeServiceInfo)
			if err != nil {
				if tc.wantError == "" || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("exepected err (%v), got err (%v)", tc.wantError, err)
				}
				return
			}

			marshaler := &jsonpb.Marshaler{}
			gotFilter, err := marshaler.MarshalToString(filterConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := util.JsonEqual(tc.wantBackendAuthFilter, gotFilter); err != nil {
				t.Errorf("makeBackendAuthFilter failed,\n %v", err)
			}
		})
	}
}

func TestServiceControl(t *testing.T) {
	fakeServiceConfig := &confpb.Service{
		Name: testProjectName,
		Apis: []*apipb.Api{
			{
				Name: testApiName,
				Methods: []*apipb.Method{
					{
						Name: "ListShelves",
					},
				},
			},
		},
		Control: &confpb.Control{
			Environment: util.StatPrefix,
		},
	}
	testData := []struct {
		desc                            string
		serviceControlCredentials       *options.IAMCredentialsOptions
		serviceAccountKey               string
		wantPartialServiceControlFilter string
	}{
		{
			desc: "get access token from imds",
			wantPartialServiceControlFilter: `
    "imdsToken": {
      "cluster": "metadata-cluster",
      "timeout": "30s",
      "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
    },`,
		},
		{
			desc: "get access token from iam",
			serviceControlCredentials: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "ServiceControl@iam.com",
				Delegates:           []string{"delegate_foo", "delegate_bar"},
			},
			wantPartialServiceControlFilter: `
    "iamToken": {
      "accessToken": {
        "remoteToken": {
          "cluster": "metadata-cluster",
          "timeout": "30s",
          "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
        }
      },
      "delegates": [
        "delegate_foo",
        "delegate_bar"
      ],
      "iamUri": {
        "cluster": "iam-cluster",
        "timeout": "30s",
        "uri": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/ServiceControl@iam.com:generateAccessToken"
      },
      "serviceAccountEmail": "ServiceControl@iam.com"
    },`,
		},
		{
			desc:              "get access token from the token agent server",
			serviceAccountKey: "this-is-sa-cred",
			wantPartialServiceControlFilter: `
    "imdsToken": {
      "cluster": "token-agent-cluster",
      "timeout": "30s",
      "uri": "http://127.0.0.1:8791/local/access_token"
    },`,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			opts := options.DefaultConfigGeneratorOptions()
			opts.ServiceControlCredentials = tc.serviceControlCredentials
			opts.ServiceAccountKey = tc.serviceAccountKey

			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Error(err)
			}

			marshaler := &jsonpb.Marshaler{}
			filter, _, err := scFilterGenFunc(fakeServiceInfo)
			if err != nil {
				t.Fatal(err)
			}

			gotFilter, err := marshaler.MarshalToString(filter)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonContains(gotFilter, tc.wantPartialServiceControlFilter); err != nil {
				t.Errorf("makeServiceControlFilter failed,\n%v", err)
			}
		})
	}
}

func TestHealthCheckFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		BackendAddress        string
		healthz               string
		fakeServiceConfig     *confpb.Service
		wantHealthCheckFilter string
	}{
		{
			desc:           "Success, generate health check filter for gRPC",
			BackendAddress: "grpc://127.0.0.1:80",
			healthz:        "healthz",
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
			wantHealthCheckFilter: `{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "exactMatch": "/healthz",
              "name":":path"
            }
          ]
        }
      }`,
		},
		{
			desc:           "Success, generate health check filter for http",
			BackendAddress: "http://127.0.0.1:80",
			healthz:        "/",
			fakeServiceConfig: &confpb.Service{
				Name: "foo.endpoints.bar.cloud.goog",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo/{id}",
							},
						},
					},
				},
			},
			wantHealthCheckFilter: `{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "exactMatch": "/",
              "name":":path"
            }
          ]
        }
      }`,
		},
	}

	for i, tc := range testdata {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
		opts.Healthz = tc.healthz
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		filter, err := makeHealthCheckFilter(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		gotFilter, err := marshaler.MarshalToString(filter)
		if err != nil {
			t.Fatal(err)
		}

		if err := util.JsonEqual(tc.wantHealthCheckFilter, gotFilter); err != nil {
			t.Errorf("Test Desc(%d): %s, makeHealthCheckFilter failed,\n%v", i, tc.desc, err)
		}
	}
}
