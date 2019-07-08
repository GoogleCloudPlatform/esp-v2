// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/api"
	"google.golang.org/genproto/protobuf/ptype"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

var (
	fakeProtoDescriptor = base64.StdEncoding.EncodeToString([]byte("rawDescriptor"))

	sourceFile = &sm.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: []byte("rawDescriptor"),
		FileType:     sm.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, _ = ptypes.MarshalAny(sourceFile)
)

func TestTranscoderFilter(t *testing.T) {
	testData := []struct {
		desc                 string
		fakeServiceConfig    *conf.Service
		wantTranscoderFilter string
	}{
		{
			desc: "Success for gRPC backend with transcoding",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				SourceInfo: &conf.SourceInfo{
					SourceFiles: []*any.Any{content},
				},
			},
			wantTranscoderFilter: fmt.Sprintf(`
        {
          "config":{
            "ignored_query_parameters": [
                "api_key",
                "key",
                "access_token"
              ],
            "proto_descriptor_bin":"%s",
            "services":[
              "%s"
            ]
          },
          "name":"envoy.grpc_json_transcoder"
        }
      `, fakeProtoDescriptor, testApiName),
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", "gRPC")
		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
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
	fakeServiceConfig := &conf.Service{
		Name: testProjectName,
		Apis: []*api.Api{
			{
				Name: "testapi",
				Methods: []*api.Method{
					{
						Name: "foo",
					},
					{
						Name: "bar",
					},
				},
			},
		},
		Backend: &conf.Backend{
			Rules: []*conf.BackendRule{
				{
					Selector: "ignore_me",
				},
				{
					Selector:        "testapi.foo",
					Address:         "https://testapi.com/foo",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "foo.com",
					},
				},
				{
					Selector:        "testapi.bar",
					Address:         "https://testapi.com/foo",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "bar.com",
					},
				},
			},
		},
	}
	wantBackendAuthFilter :=
		`{
        "config": {
          "rules": [
            {
              "jwt_audience": "bar.com",
              "operation": "testapi.bar"
            },
            {
              "jwt_audience": "foo.com",
              "operation": "testapi.foo"
            }
          ],
          "access_token": {
            "remote_token": {
              "cluster": "metadata-cluster",
              "uri": "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity",
              "timeout":"5s"
            }
          }
        },
      "name": "envoy.filters.http.backend_auth"
    }`

	flag.Set("backend_protocol", "http2")
	flag.Set("enable_backend_routing", "true")
	fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID)
	if err != nil {
		t.Fatal(err)
	}

	marshaler := &jsonpb.Marshaler{}
	gotFilter, err := marshaler.MarshalToString(makeBackendAuthFilter(fakeServiceInfo))
	gotFilter = normalizeJson(gotFilter)
	want := normalizeJson(wantBackendAuthFilter)

	if !strings.Contains(gotFilter, want) {
		t.Errorf("makeBackendAuthFilter failed, got: %s, \n\twant: %s", gotFilter, want)
	}
}

func TestPathMatcherFilter(t *testing.T) {
	testData := []struct {
		desc                  string
		fakeServiceConfig     *conf.Service
		backendProtocol       string
		wantPathMatcherFilter string
	}{
		{
			desc: "Path Matcher filter - gRPC backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
						Methods: []*api.Method{
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
			          "config": {
			            "rules": [
			              {
			                "operation": "endpoints.examples.bookstore.Bookstore.CreateShelf",
			                "pattern": {
			                  "http_method": "POST",
			                  "uri_template": "/endpoints.examples.bookstore.Bookstore/CreateShelf"
			                }
			              },
			              {
			                "operation": "endpoints.examples.bookstore.Bookstore.ListShelves",
			                "pattern": {
			                  "http_method": "POST",
			                  "uri_template": "/endpoints.examples.bookstore.Bookstore/ListShelves"
			                }
			              }
			            ]
			          },
			          "name": "envoy.filters.http.path_matcher"
			        }
			      `,
		},
		{
			desc: "Path Matcher filter - HTTP backend",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Echo_Auth_Jwt",
							},
							{
								Name: "Echo",
							},
						},
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotations.HttpRule_Get{
								Get: "/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotations.HttpRule_Post{
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
			          "config": {
			            "rules": [
			              {
			                "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			                "pattern": {
			                  "http_method": "POST",
			                  "uri_template": "/echo"
			                }
			              },
			              {
			                "operation": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
			                "pattern": {
			                  "http_method": "GET",
			                  "uri_template": "/auth/info/googlejwt"
			                }
			              }
			            ]
			          },
			          "name": "envoy.filters.http.path_matcher"
			        }
			      `,
		},
		{
			desc: "Path Matcher filter - HTTP backend with path parameters",
			fakeServiceConfig: &conf.Service{
				Name: "foo.endpoints.bar.cloud.goog",
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
							{
								Name: "Bar",
							},
						},
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Bar",
							PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotations.HttpRule_Get{
								Get: "foo/{id}",
							},
						},
						{
							Selector: "1.cloudesf_testing_cloud_goog.Bar",
							Pattern: &annotations.HttpRule_Get{
								Get: "foo",
							},
						},
					},
				},
			},
			backendProtocol: "HTTP1",
			wantPathMatcherFilter: `
			        {
			          "config": {
			            "rules": [
			              {
			                "operation": "1.cloudesf_testing_cloud_goog.Bar",
			                "pattern": {
			                  "http_method": "GET",
			                  "uri_template": "foo"
			                }
			              },
			              {
			                "extract_path_parameters": true,
			                "operation": "1.cloudesf_testing_cloud_goog.Foo",
			                "pattern": {
			                  "http_method": "GET",
			                  "uri_template": "foo/{id}"
			                }
			              }
			            ]
			          },
			          "name": "envoy.filters.http.path_matcher"
			        }
			      `,
		},
		{
			desc: "Path Matcher filter - CORS support",
			fakeServiceConfig: &conf.Service{
				Name: "foo.endpoints.bar.cloud.goog",
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      "foo.endpoints.bar.cloud.goog",
						AllowCors: true,
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotations.HttpRule_Get{
								Get: "foo",
							},
						},
					},
				},
			},
			backendProtocol: "HTTP1",
			wantPathMatcherFilter: `
			        {
			         "config": {
			            "rules": [
			              {
			                "operation": "1.cloudesf_testing_cloud_goog.CORS_0",
			                "pattern": {
			                  "http_method": "OPTIONS",
			                  "uri_template": "foo"
			                }
			              },
			              {
			                "operation": "1.cloudesf_testing_cloud_goog.Foo",
			                "pattern": {
			                  "http_method": "GET",
			                  "uri_template": "foo"
			                }
			              }
			            ]
			          },
			          "name": "envoy.filters.http.path_matcher"
			        }
			      `,
		},
		{
			desc: "Path Matcher filter - Segment Name Mapping for snake-case field",
			fakeServiceConfig: &conf.Service{
				Name: "foo.endpoints.bar.cloud.goog",
				Apis: []*api.Api{
					{
						Name: "1.cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Foo",
							},
						},
					},
				},
				Types: []*ptype.Type{
					{
						Fields: []*ptype.Field{
							&ptype.Field{
								JsonName: "fooBar",
								Name:     "foo_bar",
							},
						},
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://mybackend.com",
							Selector:        "1.cloudesf_testing_cloud_goog.Foo",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
							Authentication: &conf.BackendRule_JwtAudience{
								JwtAudience: "mybackend.com",
							},
						},
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.cloudesf_testing_cloud_goog.Foo",
							Pattern: &annotations.HttpRule_Get{
								Get: "foo/{foo_bar}",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantPathMatcherFilter: `
			        {
			          "config": {
			          "segment_names": [
			            {
			              "json_name": "fooBar",
			              "snake_name": "foo_bar"
			            }
			          ],
			          "rules": [
			            {
			              "extract_path_parameters": true,
			              "operation": "1.cloudesf_testing_cloud_goog.Foo",
			              "pattern": {
			                "http_method": "GET",
			                "uri_template": "foo/{foo_bar}"
			              }
			            }
			          ]
			        },
			        "name": "envoy.filters.http.path_matcher"
			      }`,
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", tc.backendProtocol)
		flag.Set("enable_backend_routing", "true")
		fakeServiceInfo, err := sc.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
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
