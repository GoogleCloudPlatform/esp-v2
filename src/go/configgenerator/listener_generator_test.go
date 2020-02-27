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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
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
      "autoMapping":true,
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
		if gotFilter != want {
			t.Errorf("Test Desc(%d): %s, makeTranscoderFilter failed, got: %s, want: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func TestBackendRoutingFilter(t *testing.T) {
	testdata := []struct {
		desc                     string
		protocol                 string
		fakeServiceConfig        *confpb.Service
		wantBackendRoutingFilter string
	}{
		{
			desc:     "Success, generate backend routing filter for gRPC",
			protocol: "grpc",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateShelf",
							},
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves",
							},
							Body: "shelf",
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.ListShelves",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantBackendRoutingFilter: `{
        "name": "envoy.filters.http.backend_routing",
        "typedConfig": {
          "@type":"type.googleapis.com/google.api.envoy.http.backend_routing.FilterConfig",
          "rules": [
            {
              "isConstAddress": true,
              "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
              "pathPrefix": "/foo"
            },
            {
              "operation":  "endpoints.examples.bookstore.Bookstore.ListShelves",
              "pathPrefix": "/foo"
            }
          ]
        }
      }`,
		},
		{
			desc:     "Success, generate backend routing filter for http",
			protocol: "http",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "bar",
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
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "testapi.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantBackendRoutingFilter: `{
        "name": "envoy.filters.http.backend_routing",
        "typedConfig": {
          "@type":"type.googleapis.com/google.api.envoy.http.backend_routing.FilterConfig",
          "rules": [
            {
              "operation": "testapi.bar",
              "pathPrefix": "/foo"
            },
            {
              "isConstAddress": true,
              "operation":"testapi.foo",
              "pathPrefix": "/foo"
            }
          ]
        }
      }`,
		},
		{
			desc:     "Success, generate backend routing filter with allow Cors",
			protocol: "http",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "bar",
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
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "testapi.bar",
							Address:         "https://testapipb.com/bar",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantBackendRoutingFilter: `{
        "name": "envoy.filters.http.backend_routing",
        "typedConfig": {
          "@type":"type.googleapis.com/google.api.envoy.http.backend_routing.FilterConfig",
          "rules": [
            {
              "operation":"testapi.CORS_bar",
              "pathPrefix": "/bar"
            },
            {
              "isConstAddress": true,
              "operation":"testapi.CORS_foo",
              "pathPrefix": "/foo"
            },
            {
              "operation": "testapi.bar",
              "pathPrefix": "/bar"
            },
            {
              "isConstAddress": true,
              "operation":"testapi.foo",
              "pathPrefix": "/foo"
            }
          ]
        }
      }`,
		},
	}

	for i, tc := range testdata {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = tc.protocol
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		filter, err := makeBackendRoutingFilter(fakeServiceInfo)
		if err != nil {
			t.Fatal(err)
		}

		gotFilter, err := marshaler.MarshalToString(filter)
		if err != nil {
			t.Fatal(err)
		}

		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantBackendRoutingFilter)

		if !strings.Contains(gotFilter, want) {
			t.Errorf("Test Desc(%d): %s, makeBackendAuthFilter failed,\ngot: %s, \nwant: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func TestBackendAuthFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		iamServiceAccount     string
		fakeServiceConfig     *confpb.Service
		delegates             []string
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
          "cluster":"metadata-cluster",
          "timeout":"5s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "get_testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
						{
							Selector: "get_testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "bar",
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
			wantBackendAuthFilter: `{
        "name":"envoy.filters.http.backend_auth",
        "typedConfig":{
          "@type":"type.googleapis.com/google.api.envoy.http.backend_auth.FilterConfig",
          "imdsToken":{
            "cluster":"metadata-cluster",
            "timeout":"5s",
            "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
          },
          "rules":[
            {
            	"jwtAudience": "bar.com",
            	"operation": "get_testapi.CORS_bar"
            },
            {
            	"jwtAudience": "foo.com",
            	"operation":"get_testapi.CORS_foo"
            },
            {
              "jwtAudience": "bar.com",
              "operation": "get_testapi.bar"
            },
            {
              "jwtAudience": "foo.com",
              "operation": "get_testapi.foo"
            }
          ]
        }
      }`,
		},
		{
			desc:              "Success, set iamIdToken when iam service account is set",
			iamServiceAccount: "service-account@google.com",
			delegates:         []string{"delegate_foo", "delegate_bar", "delegate_baz"},
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
         "delegates":["delegate_foo","delegate_bar","delegate_baz"],
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

	for i, tc := range testdata {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendProtocol = "grpc"
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

		marshaler := &jsonpb.Marshaler{}
		gotFilter, err := marshaler.MarshalToString(makeBackendAuthFilter(fakeServiceInfo))
		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantBackendAuthFilter)

		if !strings.Contains(gotFilter, want) {
			t.Errorf("Test Desc(%d): %s, makeBackendAuthFilter failed,\ngot: %s, \nwant: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func TestPathMatcherFilter(t *testing.T) {
	testData := []struct {
		desc                  string
		fakeServiceConfig     *confpb.Service
		backendProtocol       string
		healthz               string
		wantPathMatcherFilter string
	}{
		{
			desc: "Path Matcher filter with Healthz - gRPC backend",
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
			healthz:         "healthz",
			wantPathMatcherFilter: `
{
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"ESPv2.HealthCheck",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"/healthz"
            }
         },
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
			desc: "Path Matcher filter with Healthz - HTTP backend",
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
			backendProtocol: "HTTP",
			healthz:         "/",
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
         },
         {
            "operation":"ESPv2.HealthCheck",
            "pattern":{
               "httpMethod":"GET",
               "uriTemplate":"/"
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
			backendProtocol: "HTTP",
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
			backendProtocol: "HTTP",
			wantPathMatcherFilter: `
			        {
   "name":"envoy.filters.http.path_matcher",
   "typedConfig":{
      "@type":"type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig",
      "rules":[
         {
            "operation":"1.cloudesf_testing_cloud_goog.CORS_foo",
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
			backendProtocol: "http",
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
		opts.Healthz = tc.healthz
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

func TestHealthCheckFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		protocol              string
		healthz               string
		fakeServiceConfig     *confpb.Service
		wantHealthCheckFilter string
	}{
		{
			desc:     "Success, generate health check filter for gRPC",
			protocol: "grpc",
			healthz:  "healthz",
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
        "name": "envoy.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.config.filter.http.health_check.v2.HealthCheck",
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
			desc:     "Success, generate health check filter for http",
			protocol: "http",
			healthz:  "/",
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
								Get: "foo/{id}",
							},
						},
					},
				},
			},
			wantHealthCheckFilter: `{
        "name": "envoy.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.config.filter.http.health_check.v2.HealthCheck",
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
		opts.BackendProtocol = tc.protocol
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

		gotFilter = normalizeJson(gotFilter)
		want := normalizeJson(tc.wantHealthCheckFilter)

		if !strings.Contains(gotFilter, want) {
			t.Errorf("Test Desc(%d): %s, makeHealthCheckFilter failed,\ngot: %s, \nwant: %s", i, tc.desc, gotFilter, want)
		}
	}
}

func normalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}
