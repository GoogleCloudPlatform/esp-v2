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
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/google/go-cmp/cmp"
	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestNewGRPCTranscoderFilterGensFromOPConfig_GenConfig(t *testing.T) {
	rawDescriptor, err := proto.Marshal(&descpb.FileDescriptorSet{
		File: []*descpb.FileDescriptorProto{
			{
				Name: proto.String("test_file_desciptor_name.proto"),
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal FileDescriptorSet: %v", err)
	}

	fakeProtoDescriptor := base64.StdEncoding.EncodeToString(rawDescriptor)
	sourceFile := &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: rawDescriptor,
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, err := anypb.New(sourceFile)
	if err != nil {
		t.Fatalf("Failed to marshal source file into any: %v", err)
	}

	testData := []SuccessOPTestCase{
		{
			Desc: "Success. Generate transcoder filter with default apiKey locations and default jwt locations",
			ServiceConfigIn: &confpb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.0:80",
			},
			WantFilterConfigs: []string{
				fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "queryParamUnescapePlus":true,
      "ignoredQueryParameters":[
         "access_token",
         "api_key",
         "key"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "services":[
         "endpoints.examples.bookstore.Bookstore"
      ]
   }
}
      `, fakeProtoDescriptor),
			},
		},
		{
			Desc: "Success. Generate transcoder filter with custom apiKey locations and custom jwt locations",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
							Selector: "endpoints.examples.bookstore.Bookstore.Foo",
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
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.0:80",
			},
			WantFilterConfigs: []string{
				fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "queryParamUnescapePlus":true,
      "ignoredQueryParameters":[
         "jwt_query_param",
         "query_name_1",
         "query_name_2"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "services":[
         "endpoints.examples.bookstore.Bookstore"
      ]
   }
}
      `, fakeProtoDescriptor),
			},
		},
		{
			Desc: "Success. Generate transcoder filter with print options",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:                                "grpc://127.0.0.0:80",
				TranscodingAlwaysPrintPrimitiveFields:         true,
				TranscodingAlwaysPrintEnumsAsInts:             true,
				TranscodingStreamNewLineDelimited:             true,
				TranscodingPreserveProtoFieldNames:            true,
				TranscodingIgnoreQueryParameters:              "parameter_foo,parameter_bar",
				TranscodingIgnoreUnknownQueryParameters:       true,
				TranscodingQueryParametersDisableUnescapePlus: true,
				TranscodingCaseInsensitiveEnumParsing:         true,
			},
			WantFilterConfigs: []string{
				fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "caseInsensitiveEnumParsing":true,
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
         "preserveProtoFieldNames":true,
         "streamNewLineDelimited":true
      },
      "protoDescriptorBin":"%s",
      "services":[
         "endpoints.examples.bookstore.Bookstore"
      ]
   }
}
      `, fakeProtoDescriptor),
			},
		},
		{
			Desc: "Success. Generate transcoder filter with strict request validation",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
			OptsIn: options.ConfigGeneratorOptions{
				TranscodingStrictRequestValidation: true,
				BackendAddress:                     "grpc://127.0.0.0:80",
			},
			WantFilterConfigs: []string{
				fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "queryParamUnescapePlus":true,
      "ignoredQueryParameters":[
         "api_key",
         "key"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "requestValidationOptions":{
         "rejectUnknownMethod":true,
         "rejectUnknownQueryParameters":true
      },
      "services":[
         "endpoints.examples.bookstore.Bookstore"
      ]
   }
}
      `, fakeProtoDescriptor),
			},
		},
		{
			Desc: "Not generate transcoder filter without protofile",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.0:80",
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "Not generate transcoder filter with local http backend address",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress:          "grpc://127.0.0.0:80",
				LocalHTTPBackendAddress: "http://127.0.0.1:8080",
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "Not generate transcoder filter when all backends at NOT gRPC",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address: "https://remote-backend.com",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.0:80",
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "Success. Generate transcoder filter when at least 1 remote backend is gRPC",
			ServiceConfigIn: &confpb.Service{
				Name: "endpoints.examples.bookstore.Bookstore",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address: "https://remote-backend-1.com",
						},
						{
							Address: "grpcs://remote-backend-2.com",
						},
						{
							Address: "https://remote-backend-3.com",
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.0:80",
			},
			WantFilterConfigs: []string{
				fmt.Sprintf(`
{
   "name":"envoy.filters.http.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder",
      "autoMapping":true,
      "convertGrpcStatus":true,
      "queryParamUnescapePlus":true,
      "ignoredQueryParameters":[
         "api_key",
         "key"
      ],
      "printOptions":{},
      "protoDescriptorBin":"%s",
      "services":[
         "endpoints.examples.bookstore.Bookstore"
      ]
   }
}
      `, fakeProtoDescriptor),
			},
		},
		// TODO: discovery APIs and options
	}

	for _, tc := range testData {
		tc.RunTest(t, filtergen.NewGRPCTranscoderFilterGensFromOPConfig)
	}
}

func TestPreserveDefaultHttpBinding(t *testing.T) {
	testData := []struct {
		desc             string
		originalHttpRule string
		wantHttpRule     string
	}{
		{
			// Add the default http binding if it's not present.
			desc: "default http binding is not present",
			originalHttpRule: `
				selector: "package.name.Service.Method"
				post: "/v1/Service/Method"
			`,
			wantHttpRule: `
				selector: "package.name.Service.Method"
				post: "/v1/Service/Method"
				additional_bindings: {
					post: "/package.name.Service/Method"
					body: "*"
				}
			`,
		},
		{
			// Do not add the default binding if it's identitical to the primary
			// binding. Difference in selector and additional_bindings is ignored.
			desc: "default http binding is not present",
			originalHttpRule: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
			wantHttpRule: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
		},
		{
			// Do not add the default binding if it's identitical to any existing
			// additional binding.
			desc: "default http binding is not present",
			originalHttpRule: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
			wantHttpRule: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
		},
	}

	for _, tc := range testData {
		got := &ahpb.HttpRule{}
		if err := prototext.Unmarshal([]byte(tc.originalHttpRule), got); err != nil {
			fmt.Println("failed to unmarshal originalHttpRule: ", err)
		}

		filtergen.PreserveDefaultHttpBinding(got, "/package.name.Service/Method")
		want := &ahpb.HttpRule{}
		if err := prototext.Unmarshal([]byte(tc.wantHttpRule), want); err != nil {
			fmt.Println("failed to unmarshal wantHttpRule: ", err)
		}

		if diff := utils.ProtoDiff(want, got); diff != "" {
			t.Errorf("Result is not the same: diff (-want +got):\n%v", diff)
		}
	}
}

func TestUpdateProtoDescriptorFromOPConfig(t *testing.T) {
	testData := []struct {
		desc      string
		service   string
		opts      options.ConfigGeneratorOptions
		inDesc    string
		wantDesc  string
		wantError string
	}{
		{
			// The input Descriptor is an invalid data, it results in error.
			desc:      "Failed to unmarshal error",
			service:   "",
			inDesc:    "invalid proto descriptor",
			wantError: "failed to unmarshal",
		},
		{
			// ApiNames is a wrong service name, protoDescriptor is not modified.
			desc: "Wrong apiName, not override",
			service: `
apis {
	name: "package.name.WrongService"
}
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/{name=*}"
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor doesn't have MethodOptions, the http rule is copied with the default binding added in its additional bindings
			desc: "Not method options",
			service: `
apis {
	name: "package.name.Service"
}
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            post: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor has an empty MethodOptions, the http rule is copied with the default binding added in its additional bindings
			desc: "Empty method options",
			service: `
apis {
	name: "package.name.Service"
}
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            post: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor has a different annotation, the http rule is copied with the default binding added in its additional bindings
			desc: "Basic overwritten case",
			service: `
apis {
	name: "package.name.Service"
}
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            post: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// The http rule has a different service name. It is not copied but the default binding is added if it is absent
			desc: "Empty http rule as it has different service name",
			service: `
apis {
	name: "package.name.Service"
}
http: {
  rules: {
    selector: "package.name.WrongService.Method"
    post: "/v2/{name=*}"
  }
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
          additional_bindings: {
            post: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// The default http rule will not be added if no http rule is specified, the default binding
			// will be done in Envoy's json transcoder.
			desc: "Default http rule will not be added if no http rule is specified",
			service: `
apis {
	name: "package.name.Service"
}`,
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {}
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {}
    }
  }
}`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			serviceConfig := &confpb.Service{}
			if err := prototext.Unmarshal([]byte(tc.service), serviceConfig); err != nil {
				t.Fatal("failed to unmarshal service config: ", err)
			}

			var byteDesc []byte
			fds := &descpb.FileDescriptorSet{}
			if err := prototext.Unmarshal([]byte(tc.inDesc), fds); err != nil {
				// Failed case is to use raw test to test failure
				byteDesc = []byte(tc.inDesc)
			} else {
				byteDesc, _ = proto.Marshal(fds)
			}

			gotByteDesc, err := filtergen.UpdateProtoDescriptorFromOPConfig(serviceConfig, tc.opts, byteDesc)
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("failed, expected: %s, got: %v", tc.wantError, err)
			}
			if tc.wantError == "" && err != nil {
				t.Errorf("got unexpected error: %v", err)
			}

			if tc.wantDesc != "" {
				got := &descpb.FileDescriptorSet{}
				// Not need to check error, gotByteDesc is just marshaled from the updateProtoDescriptorFromOPConfig()
				proto.Unmarshal(gotByteDesc, got)
				want := &descpb.FileDescriptorSet{}
				if err := prototext.Unmarshal([]byte(tc.wantDesc), want); err != nil {
					t.Fatal("failed to unmarshal wantDesc: ", err)
				}

				if diff := utils.ProtoDiff(want, got); diff != "" {
					t.Errorf("Result is not the same: diff (-want +got):\n%v", diff)
				}
			}
		})
	}
}

func TestGetIgnoredQueryParamsFromOPConfig(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optsIn          options.ConfigGeneratorOptions
		wantParams      []string
	}{
		{
			desc: "Success. Default jwt locations with --transcoding_ignore_query_params flag",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				TranscodingIgnoreQueryParameters: "foo,bar",
			},
			wantParams: []string{
				"access_token",
				"api_key",
				"bar",
				"foo",
				"key",
			},
		},
		{
			desc: "Success. Custom jwt locations with transcoding_ignore_query_params flag",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
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
			},
			optsIn: options.ConfigGeneratorOptions{
				TranscodingIgnoreQueryParameters: "foo,bar",
			},
			wantParams: []string{
				"api_key",
				"bar",
				"foo",
				"jwt_query_param",
				"key",
			},
		},
		{
			desc: "Succeed, only header, no query params",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*confpb.SystemParameter{
								{
									Name:       "api_key",
									HttpHeader: "header_name",
								},
							},
						},
					},
				},
			},
			wantParams: nil,
		},
		{
			desc: "Succeed, only url query",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*confpb.SystemParameter{
								{
									Name:              "api_key",
									UrlQueryParameter: "query_name",
								},
							},
						},
					},
				},
			},
			wantParams: []string{"query_name"},
		},
		{
			desc: "Succeed, url query plus header",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
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
			wantParams: []string{
				"query_name_1",
				"query_name_2",
			},
		},
		{
			desc: "Succeed, url query plus header for multiple apis with one using default ApiKeyLocation",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
					{
						Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
					{
						Name: "3.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "baz",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.foo",
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
						{
							Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.bar",
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
			wantParams: []string{
				"api_key",
				"key",
				"query_name_1",
				"query_name_2",
			},
		},
		{
			desc: "Skip system parameters for discovery APIs and use default parameters",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "google.discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "google.discovery.GetDiscoveryRest",
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
			wantParams: []string{
				"api_key",
				"key",
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			gotParamsMap, err := filtergen.GetIgnoredQueryParamsFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("GetIgnoredQueryParamsFromOPConfig() got unexpected error: %v", err)
			}

			var gotParams []string
			for param, _ := range gotParamsMap {
				gotParams = append(gotParams, param)
			}
			sort.Strings(gotParams)

			if diff := cmp.Diff(tc.wantParams, gotParams); diff != "" {
				t.Errorf("GetIgnoredQueryParamsFromOPConfig diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetIgnoredQueryParamsFromOPConfig_BadInput(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optsIn          options.ConfigGeneratorOptions
		wantError       string
	}{
		{
			desc: "Failure. Wrong jwt locations setting Query with valuePrefix in the same time",
			serviceConfigIn: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
							JwtLocations: []*confpb.JwtLocation{
								{
									In: &confpb.JwtLocation_Query{
										Query: "jwt_query_param",
									},
									ValuePrefix: "jwt_query_header_prefix",
								},
							},
						},
					},
				},
			},
			wantError: `error processing authentication provider (auth_provider): JwtLocation type [Query] should be set without valuePrefix, but it was set to [jwt_query_header_prefix]`,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := filtergen.GetIgnoredQueryParamsFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err == nil {
				t.Fatalf("GetIgnoredQueryParamsFromOPConfig() got no error, want error to contain %q", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("GetIgnoredQueryParamsFromOPConfig() got error %v, want error to contain %q", err.Error(), tc.wantError)
			}
		})
	}
}

func TestGetDisabledSelectorsFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc                  string
		serviceConfigIn       *confpb.Service
		optsIn                options.ConfigGeneratorOptions
		wantDisabledSelectors map[string]bool
	}{
		{
			desc: "Nothing is disabled",
			serviceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector: "selector_1",
						},
						{
							Selector: "selector_2",
						},
					},
				},
			},
			wantDisabledSelectors: map[string]bool{},
		},
		{
			desc: "Non-OpenAPI HTTP backend is disabled",
			serviceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector: "selector_1",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Selector: "selector_1",
								},
							},
						},
					},
				},
			},
			wantDisabledSelectors: map[string]bool{
				"selector_1": true,
			},
		},
		{
			desc: "Non-OpenAPI HTTP backend is still transcoded when local backend address override is enabled",
			serviceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector: "selector_1",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Selector: "selector_1",
								},
							},
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				EnableBackendAddressOverride: true,
			},
			wantDisabledSelectors: map[string]bool{},
		},
		{
			desc: "Non-OpenAPI HTTP backend is still transcoded when it's a discovery API",
			serviceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector: "google.discovery.GetDiscoveryRest",
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": {
									Selector: "google.discovery.GetDiscoveryRest",
								},
							},
						},
					},
				},
			},
			wantDisabledSelectors: map[string]bool{},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			gotDisabledSelectors, err := filtergen.GetDisabledSelectorsFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("GetDisabledSelectorsFromOPConfig() got unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantDisabledSelectors, gotDisabledSelectors); diff != "" {
				t.Errorf("GenTranslationInfoFromOPConfig() diff (-want +got):\n%s", diff)
			}
		})
	}
}
