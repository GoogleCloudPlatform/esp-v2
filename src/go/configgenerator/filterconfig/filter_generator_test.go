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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

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

	testProjectName         = "bookstore.endpoints.project123.cloud.goog"
	testApiName             = "endpoints.examples.bookstore.Bookstore"
	testServiceControlEnv   = "servicecontrol.googleapis.com"
	testConfigID            = "2019-03-02r0"
	testProtoDescriptorPath = "/host/descriptor"
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
		transcodingQueryParametersUnescapePlus  bool
		transcodingFilePath                     string
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
			transcodingQueryParametersUnescapePlus:  true,
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "ignoreUnknownQueryParameters":true,
      "queryParamUnescapePlus":true,
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
		{
			desc: "Success. Generate transcoder filter with proto descriptor path",
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
			transcodingFilePath: testProtoDescriptorPath,
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
			"ignoredQueryParameters": [
				"api_key",
				"key"
			],
      "printOptions":{},
      "protoDescriptor":"%s",
      "services":[
         "%s"
      ]
   }
}
      `, testProtoDescriptorPath, testApiName),
		},
		{
			desc: "Not generate transcoder filter without protofile",
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
			},
			transcodingFilePath:  testProtoDescriptorPath,
			wantTranscoderFilter: "",
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
			opts.TranscodingQueryParametersUnescapePlus = tc.transcodingQueryParametersUnescapePlus
			opts.TranscodingFilePath = tc.transcodingFilePath
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			filterConfig := makeTranscoderFilter(fakeServiceInfo)
			if filterConfig == nil && tc.wantTranscoderFilter == "" {
				// Expected no filter config generated
				return
			}
			if filterConfig == nil {
				t.Fatal("Got empty filter config.")
			}

			marshaler := &jsonpb.Marshaler{}
			gotFilter, err := marshaler.MarshalToString(filterConfig)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(tc.wantTranscoderFilter, gotFilter); err != nil {
				t.Errorf("Test Desc(%d): %s, makeTranscoderFilter failed, \n %v", i, tc.desc, err)
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
              "stringMatch":{"exact":"/healthz"},
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
              "stringMatch":{"exact":"/"},
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
