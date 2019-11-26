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

package configgenerator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/configinfo"
	"github.com/GoogleCloudPlatform/api-proxy/src/go/options"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

	anypb "github.com/golang/protobuf/ptypes/any"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
	ptypepb "google.golang.org/genproto/protobuf/ptype"
)

var (
	fakeProtoDescriptor = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))

	sourceFile = &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: []byte("rawDescriptor"),
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, _ = ptypes.MarshalAny(sourceFile)
)

func TestTranscoderFilter(t *testing.T) {
	testData := []struct {
		desc                 string
		fakeServiceConfig    *confpb.Service
		wantTranscoderFilter string
	}{
		{
			desc: "Success for gRPC backend with transcoding",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
			},
			wantTranscoderFilter: fmt.Sprintf(`
{
   "name":"envoy.grpc_json_transcoder",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.config.filter.http.transcoder.v2.GrpcJsonTranscoder",
      "convertGrpcStatus":true,
      "ignoredQueryParameters":[
         "api_key",
         "key",
         "access_token"
      ],
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
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = "gRPC"
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotFilter, err := marshaler.MarshalToString(makeTranscoderFilter(fakeServiceInfo))

		// Normalize both path matcher filter and gotListeners.
		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantTranscoderFilter)
		if !strings.Contains(gotFilter, want) {
			t.Errorf("Test Desc(%d): %s, makeTranscoderFilter failed, got: %s, want: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func TestBackendAuthFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		iamServiceAccount     string
		fakeServiceConfig     *confpb.Service
		wantBackendAuthFilter string
	}{
		{
			desc: "Success, generate backend auth filter in general",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
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
							Selector: "ignore_me",
						},
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
			wantBackendAuthFilter: `
{
   "name":"envoy.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.backend_auth.FilterConfig",
      "imdsToken":{
         "imdsServerUri":{
            "cluster":"metadata-cluster",
            "timeout":"5s",
            "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
         }
      },
      "rules":[
         {
            "jwtAudience":"bar.com",
            "operation":"testapipb.bar"
         },
         {
            "jwtAudience":"foo.com",
            "operation":"testapipb.foo"
         }
      ]
   }
}
`,
		},
		{
			desc:              "Success, set iamIdToken when iam service account is set",
			iamServiceAccount: "service-account@google.com",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
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
   "name":"envoy.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.backend_auth.FilterConfig",
      "iamToken":{
         "accessToken":{
            "remoteToken":{
               "cluster":"metadata-cluster",
               "timeout":"5s",
               "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
            }
         },
         "iamUri":{
            "cluster":"iam-cluster",
            "timeout":"5s",
            "uri":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/service-account@google.com:generateIdToken"
         },
         "serviceAccountEmail":"service-account@google.com"
      },
      "rules":[
         {
            "jwtAudience":"bar.com",
            "operation":"testapipb.bar"
         }
      ]
   }
}
`,
		},
	}

	for _, tc := range testdata {

		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = "grpc"
		opts.EnableBackendRouting = true
		opts.IamServiceAccount = tc.iamServiceAccount
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotFilter, err := marshaler.MarshalToString(makeBackendAuthFilter(fakeServiceInfo))
		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantBackendAuthFilter)

		if !strings.Contains(gotFilter, want) {
			t.Errorf("makeBackendAuthFilter failed,\ngot: %s, \nwant: %s", gotFilter, want)
		}
	}
}

func TestPathMatcherFilter(t *testing.T) {
	testData := []struct {
		desc                  string
		fakeServiceConfig     *confpb.Service
		backendProtocol       string
		wantPathMatcherFilter string
	}{
		{
			desc: "Path Matcher filter - gRPC backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
			},
			backendProtocol: "GRPC",
			wantPathMatcherFilter: `
{
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"endpoints.examples.bookstore.Bookstore.CreateShelf",
            "pattern":{
               "httpMethod":"POST",
               "uriTemplate":"/endpoints.examples.bookstore.Bookstore/CreateShelf"
            }
         },
         {
            "operation":"endpoints.examples.bookstore.Bookstore.ListShelves",
            "pattern":{
               "httpMethod":"POST",
               "uriTemplate":"/endpoints.examples.bookstore.Bookstore/ListShelves"
            }
         }
      ]
   }
}
			      `,
		},
		{
			desc: "Path Matcher filter - HTTP backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Echo_Auth_Jwt",
							},
							{
								Name: "Echo",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/echo",
							},
							Body: "message",
						},
					},
				},
			},
			backendProtocol: "HTTP1",
			wantPathMatcherFilter: `
			        {
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
            "pattern":{
               "httpMethod":"POST",
               "uriTemplate":"/echo"
            }
         },
         {
            "operation":"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"/auth/info/googlejwt"
            }
         }
      ]
   }
}
			      `,
		},
		{
			desc: "Path Matcher filter - HTTP backend with path parameters",
			fakeServiceConfig: &confpb.Service{
				Name: "foo.endpoints.bar.cloud.goog",
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
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo/{id}",
							},
						},
						{
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
					},
				},
			},
			backendProtocol: "HTTP1",
			wantPathMatcherFilter: `
			        {
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"1.cloudesf_testing_cloud_goog.Bar",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"foo"
            }
         },
         {
            "extractPathParameters":true,
            "operation":"1.cloudesf_testing_cloud_goog.Foo",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"foo/{id}"
            }
         }
      ]
   }
}
			      `,
		},
		{
			desc: "Path Matcher filter - CORS support",
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
				Endpoints: []*confpb.Endpoint{
					{
						Name:      "foo.endpoints.bar.cloud.goog",
						AllowCors: true,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
					},
				},
			},
			backendProtocol: "HTTP1",
			wantPathMatcherFilter: `
			        {
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"1.cloudesf_testing_cloud_goog.CORS_0",
            "pattern":{
               "httpMethod":"OPTIONS",
               "uriTemplate":"foo"
            }
         },
         {
            "operation":"1.cloudesf_testing_cloud_goog.Foo",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"foo"
            }
         }
      ]
   }
}
			      `,
		},
		{
			desc: "Path Matcher filter - Segment Name Mapping for snake-case field",
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
				Types: []*ptypepb.Type{
					{
						Fields: []*ptypepb.Field{
							&ptypepb.Field{
								JsonName: "fooBar",
								Name:     "foo_bar",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo/{foo_bar}",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantPathMatcherFilter: `
			        {
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "extractPathParameters":true,
            "operation":"1.cloudesf_testing_cloud_goog.Foo",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"foo/{foo_bar}"
            }
         }
      ],
      "segmentNames":[
         {
            "jsonName":"fooBar",
            "snakeName":"foo_bar"
         }
      ]
   }
}`,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.backendProtocol
		opts.EnableBackendRouting = true
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}
		marshaler := &jsonpb.Marshaler{}
		gotFilter, err := marshaler.MarshalToString(makePathMatcherFilter(fakeServiceInfo))

		// Normalize both path matcher filter and gotListeners.
		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantPathMatcherFilter)
		if !strings.Contains(gotFilter, want) {
			t.Errorf("Test Desc(%d): %s, makePathMatcherFilter failed, got: %s, want: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func normalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}
