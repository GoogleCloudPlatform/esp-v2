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

package filtergen

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/golang/protobuf/jsonpb"
	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestTranscoderFilter(t *testing.T) {
	rawDescriptor, err := proto.Marshal(&descpb.FileDescriptorSet{})
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

	testData := []struct {
		desc                                          string
		fakeServiceConfig                             *confpb.Service
		transcodingAlwaysPrintPrimitiveFields         bool
		transcodingAlwaysPrintEnumsAsInts             bool
		transcodingStreamNewLineDelimited             bool
		transcodingPreserveProtoFieldNames            bool
		transcodingIgnoreQueryParameters              string
		transcodingIgnoreUnknownQueryParameters       bool
		transcodingQueryParametersDisableUnescapePlus bool
		transcodingStrictRequestValidation            bool
		transcodingCaseInsensitiveEnumParsing         bool
		wantTranscoderFilter                          string
		localHTTPBackendAddress                       string
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
      "queryParamUnescapePlus":true,
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
      "queryParamUnescapePlus":true,
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
			transcodingAlwaysPrintPrimitiveFields:         true,
			transcodingAlwaysPrintEnumsAsInts:             true,
			transcodingStreamNewLineDelimited:             true,
			transcodingPreserveProtoFieldNames:            true,
			transcodingIgnoreQueryParameters:              "parameter_foo,parameter_bar",
			transcodingIgnoreUnknownQueryParameters:       true,
			transcodingQueryParametersDisableUnescapePlus: true,
			transcodingCaseInsensitiveEnumParsing:         true,
			wantTranscoderFilter: fmt.Sprintf(`
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
         "%s"
      ]
   }
}
      `, fakeProtoDescriptor, testApiName),
		},
		{
			desc: "Success. Generate transcoder filter with strict request validation",
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
			transcodingStrictRequestValidation: true,
			wantTranscoderFilter: fmt.Sprintf(`
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
         "%s"
      ]
   }
}
      `, fakeProtoDescriptor, testApiName),
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.0:80"
			opts.LocalHTTPBackendAddress = tc.localHTTPBackendAddress
			opts.TranscodingAlwaysPrintPrimitiveFields = tc.transcodingAlwaysPrintPrimitiveFields
			opts.TranscodingPreserveProtoFieldNames = tc.transcodingPreserveProtoFieldNames
			opts.TranscodingStreamNewLineDelimited = tc.transcodingStreamNewLineDelimited
			opts.TranscodingAlwaysPrintEnumsAsInts = tc.transcodingAlwaysPrintEnumsAsInts
			opts.TranscodingIgnoreQueryParameters = tc.transcodingIgnoreQueryParameters
			opts.TranscodingIgnoreUnknownQueryParameters = tc.transcodingIgnoreUnknownQueryParameters
			opts.TranscodingQueryParametersDisableUnescapePlus = tc.transcodingQueryParametersDisableUnescapePlus
			opts.TranscodingStrictRequestValidation = tc.transcodingStrictRequestValidation
			opts.TranscodingCaseInsensitiveEnumParsing = tc.transcodingCaseInsensitiveEnumParsing
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gen := NewGRPCTranscoderGenerator(fakeServiceInfo)
			if !gen.IsEnabled() {
				t.Fatal("GRPCTranscoderGenerator is not enabled, want it to be enabled")
			}

			filterConfig, err := gen.GenFilterConfig(fakeServiceInfo)
			if err != nil {
				t.Fatalf("GenFilterConfig got err %v, want no err", err)
			}

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
				t.Errorf("GenFilterConfig has JSON diff\n%v", err)
			}
		})
	}
}

func TestTranscoderFilter_Disabled(t *testing.T) {
	rawDescriptor, err := proto.Marshal(&descpb.FileDescriptorSet{})
	if err != nil {
		t.Fatalf("Failed to marshal FileDescriptorSet: %v", err)
	}

	sourceFile := &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: rawDescriptor,
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, err := anypb.New(sourceFile)
	if err != nil {
		t.Fatalf("Failed to marshal source file into any: %v", err)
	}

	testData := []struct {
		desc                    string
		fakeServiceConfig       *confpb.Service
		localHTTPBackendAddress string
	}{
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
		},
		{
			desc: "Not generate transcoder filter with test-only http backend address",
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
			localHTTPBackendAddress: "http://127.0.0.1:8080",
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.0:80"
			opts.LocalHTTPBackendAddress = tc.localHTTPBackendAddress
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gen := NewGRPCTranscoderGenerator(fakeServiceInfo)
			if gen.IsEnabled() {
				t.Errorf("GRPCTranscoderGenerator is enabled, want it to be disabled")
			}
		})
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

		preserveDefaultHttpBinding(got, "/package.name.Service/Method")
		want := &ahpb.HttpRule{}
		if err := prototext.Unmarshal([]byte(tc.wantHttpRule), want); err != nil {
			fmt.Println("failed to unmarshal wantHttpRule: ", err)
		}

		if diff := utils.ProtoDiff(want, got); diff != "" {
			t.Errorf("Result is not the same: diff (-want +got):\n%v", diff)
		}
	}
}

func TestUpdateProtoDescriptor(t *testing.T) {
	testData := []struct {
		desc      string
		service   string
		apiNames  []string
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
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.WrongService"},
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
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
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
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
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
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
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
http: {
  rules: {
    selector: "package.name.WrongService.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
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
			desc:     "Default http rule will not be added if no http rule is specified",
			service:  "",
			apiNames: []string{"package.name.Service"},
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

			gotByteDesc, err := updateProtoDescriptor(serviceConfig, tc.apiNames, byteDesc)
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("failed, expected: %s, got: %v", tc.wantError, err)
			}
			if tc.wantError == "" && err != nil {
				t.Errorf("got unexpected error: %v", err)
			}

			if tc.wantDesc != "" {
				got := &descpb.FileDescriptorSet{}
				// Not need to check error, gotByteDesc is just marshaled from the updateProtoDescriptor()
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
