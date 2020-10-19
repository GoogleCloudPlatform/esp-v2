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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestMakeRouteConfig(t *testing.T) {
	testData := []struct {
		desc                          string
		enableStrictTransportSecurity bool
		fakeServiceConfig             *confpb.Service
		wantedError                   string
		wantRouteConfig               string
	}{
		{
			desc:                          "Enable Strict Transport Security",
			enableStrictTransportSecurity: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
				},
				},
			},
			wantRouteConfig: `
{
  "name":"local_route",
  "virtualHosts":[
    {
      "domains":[
        "*"
      ],
      "name":"backend",
      "routes":[
        {
          "decorator":{
            "operation":"ingress Echo"
          },
          "match":{
            "headers":[
              {
                "exactMatch":"GET",
                "name":":method"
              }
            ],
            "path":"/echo"
          },
          "responseHeadersToAdd":[
            {
              "header":{
                "key":"Strict-Transport-Security",
                "value":"max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route":{
            "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "timeout":"15s"
          },
          "typedPerFilterConfig":{
            "com.google.espv2.filters.http.service_control":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName":"endpoints.examples.bookstore.Bookstore.Echo"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc:                          "Enable Strict Transport Security for remote backend",
			enableStrictTransportSecurity: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.Foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "foo",
							},
						},
					},
				},
			},
			wantRouteConfig: `
{
  "name": "local_route",
  "virtualHosts": [
    {
      "domains": [
        "*"
      ],
      "name": "backend",
      "routes": [
        {
          "decorator":{
            "operation":"ingress Foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "foo"
          },
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig":{
            "com.google.espv2.filters.http.backend_auth":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience":"bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath":{"path":"/foo"}
            },
            "com.google.espv2.filters.http.service_control":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName":"endpoints.examples.bookstore.Bookstore.Foo"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "Wildcard paths and wildcard http method for remote backend",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.Foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.Foo",
							Pattern: &annotationspb.HttpRule_Custom{
								Custom: &annotationspb.CustomHttpPattern{
									Path: "/v1/{book_name=*}/test/**",
									Kind: "*",
								},
							},
						},
					},
				},
			},
			wantRouteConfig: `
{
  "name":"local_route",
  "virtualHosts":[
    {
      "domains":[
        "*"
      ],
      "name":"backend",
      "routes":[
        {
          "decorator":{
            "operation":"ingress Foo"
          },
          "match":{
            "safeRegex":{
              "googleRe2":{},
              "regex":"^/v1/[^\\/]+/test/.*$"
            }
          },
          "route":{
            "cluster":"backend-cluster-testapipb.com:443",
            "hostRewriteLiteral":"testapipb.com",
            "timeout":"15s"
          },
          "typedPerFilterConfig":{
            "com.google.espv2.filters.http.backend_auth":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience":"bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
             "constantPath": {
                "path": "/foo",
                "urlTemplate": "/v1/{book_name=*}/test/**"
              }
            },
            "com.google.espv2.filters.http.service_control":{
              "@type":"type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName":"endpoints.examples.bookstore.Bookstore.Foo"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "Oversize wildcard path regex",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.Foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: getOverSizeRegexForTest(),
							},
						},
					},
				},
			},
			wantedError: "invalid route path regex: regex program size(1003) is larger than the max expected(1000)",
		},
		{
			desc: "path_rewrite: http rule url_templates with variable bindings.",
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
								Get: "/v1/shelves/{shelves}",
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelves}",
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
			wantRouteConfig: `{
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
                "exactMatch": "POST",
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/shelves/[^\\/]+$"
            }
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/foo",
                "urlTemplate": "/v1/shelves/{shelves}"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "endpoints.examples.bookstore.Bookstore.CreateShelf"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress ListShelves"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/shelves/[^\\/]+$"
            }
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "pathPrefix": "/foo"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "endpoints.examples.bookstore.Bookstore.ListShelves"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "path_rewrite: http rule url_templates without variable bindings.",
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
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "foo"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "foo.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/foo"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "bar"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "pathPrefix": "/foo"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.bar"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "http rule url_templates with allow Cors",
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
								Get: "foo/{x}",
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
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/foo?query=ignored",
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
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^foo/[^\\/]+$"
            }
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "foo.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/foo",
                "urlTemplate": "foo/{x}"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "bar"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "pathPrefix": "/bar"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.bar"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress ESPv2_Autogenerated_CORS_foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "OPTIONS",
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^foo/[^\\/]+$"
            }
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "foo.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/foo",
                "urlTemplate": "foo/{x}"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.ESPv2_Autogenerated_CORS_foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress ESPv2_Autogenerated_CORS_bar"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "OPTIONS",
                "name": ":method"
              }
            ],
            "path": "bar"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "bar.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "pathPrefix": "/bar"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.ESPv2_Autogenerated_CORS_bar"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "path_rewrite: empty path for APPEND, no path_rewrite",
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
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
						{
							Selector:        "testapi.bar",
							Address:         "https://testapipb.com/",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
					},
				},
			},
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "foo"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "bar"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.bar"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "path_rewrite: empty path for CONST always generates `/` prefix",
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
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
						{
							Selector:        "testapi.bar",
							Address:         "https://testapipb.com/",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "foo"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "bar"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.bar"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "path_rewrite: both url_template and path_prefix are `/` for CONST",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "/"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "Multiple http rules for one selector",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/",
							},
						},
						{
							Selector: "testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo/{abc}",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantRouteConfig: `{
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
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "path": "/"
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "exactMatch": "GET",
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/]+$"
            }
          },
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "timeout": "15s"
          },
          "typedPerFilterConfig": {
            "com.google.espv2.filters.http.backend_auth": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.backend_auth.PerRouteFilterConfig",
              "jwtAudience": "https://testapipb.com"
            },
            "com.google.espv2.filters.http.path_rewrite": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.path_rewrite.PerRouteFilterConfig",
              "constantPath": {
                "path": "/",
                "urlTemplate": "/foo/{abc}"
              }
            },
            "com.google.espv2.filters.http.service_control": {
              "@type": "type.googleapis.com/espv2.api.envoy.v9.http.service_control.PerRouteFilterConfig",
              "operationName": "testapi.foo"
            }
          }
        }
      ]
    }
  ]
}`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.EnableHSTS = tc.enableStrictTransportSecurity
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gotRoute, err := MakeRouteConfig(fakeServiceInfo)
			if tc.wantedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
					t.Fatalf("expected err: %v, got: %v", tc.wantedError, err)
				}
				return
			} else if err != nil {
				t.Fatalf("expected err: %v, got: %v", tc.wantedError, err)
			}

			marshaler := &jsonpb.Marshaler{}
			gotConfig, err := marshaler.MarshalToString(gotRoute)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(tc.wantRouteConfig, gotConfig); err != nil {
				t.Errorf("MakeRouteConfig failed, \n %v", err)
			}
		})
	}
}

func TestMakeRouteConfigForCors(t *testing.T) {
	testData := []struct {
		desc string
		// Test parameters, in the order of "cors_preset", "cors_allow_origin"
		// "cors_allow_origin_regex", "cors_allow_methods", "cors_allow_headers"
		// "cors_expose_headers"
		params           []string
		allowCredentials bool
		wantedError      string
		wantCorsPolicy   *routepb.CorsPolicy
	}{
		{
			desc:           "No Cors",
			wantCorsPolicy: nil,
		},
		{
			desc:        "Incorrect configured basic Cors",
			params:      []string{"basic", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: "cors_allow_origin cannot be empty when cors_preset=basic",
		},
		{
			desc:        "Incorrect configured  Cors",
			params:      []string{"", "", "", "GET", "", ""},
			wantedError: "cors_preset must be set in order to enable CORS support",
		},
		{
			desc:        "Incorrect configured regex Cors",
			params:      []string{"cors_with_regexs", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: `cors_preset must be either "basic" or "cors_with_regex"`,
		},
		{
			desc:        "Oversize cors origin regex",
			params:      []string{"cors_with_regex", "", getOverSizeRegexForTest(), "", "Origin,Content-Type,Accept", ""},
			wantedError: `invalid cors origin regex: regex program size(1001) is larger than the max expected(1000)`,
		},
		{
			desc:   "Correct configured basic Cors, with allow methods",
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", ""},
			wantCorsPolicy: &routepb.CorsPolicy{
				AllowOriginStringMatch: []*matcher.StringMatcher{
					{
						MatchPattern: &matcher.StringMatcher_Exact{
							Exact: "http://example.com",
						},
					},
				},
				AllowMethods:     "GET,POST,PUT,OPTIONS",
				AllowCredentials: &wrapperspb.BoolValue{Value: false},
			},
		},
		{
			desc:   "Correct configured regex Cors, with allow headers",
			params: []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "Origin,Content-Type,Accept", ""},
			wantCorsPolicy: &routepb.CorsPolicy{
				AllowOriginStringMatch: []*matcher.StringMatcher{
					{
						MatchPattern: &matcher.StringMatcher_SafeRegex{
							SafeRegex: &matcher.RegexMatcher{
								EngineType: &matcher.RegexMatcher_GoogleRe2{
									GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
								},
								Regex: `^https?://.+\\.example\\.com$`,
							},
						},
					},
				},
				AllowHeaders:     "Origin,Content-Type,Accept",
				AllowCredentials: &wrapperspb.BoolValue{Value: false},
			},
		},
		{
			desc:             "Correct configured regex Cors, with expose headers",
			params:           []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "", "Content-Length"},
			allowCredentials: true,
			wantCorsPolicy: &routepb.CorsPolicy{
				AllowOriginStringMatch: []*matcher.StringMatcher{
					{
						MatchPattern: &matcher.StringMatcher_SafeRegex{
							SafeRegex: &matcher.RegexMatcher{
								EngineType: &matcher.RegexMatcher_GoogleRe2{
									GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
								},
								Regex: `^https?://.+\\.example\\.com$`,
							},
						},
					},
				},
				ExposeHeaders:    "Content-Length",
				AllowCredentials: &wrapperspb.BoolValue{Value: true},
			},
		},
	}

	for _, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		if tc.params != nil {
			opts.CorsPreset = tc.params[0]
			opts.CorsAllowOrigin = tc.params[1]
			opts.CorsAllowOriginRegex = tc.params[2]
			opts.CorsAllowMethods = tc.params[3]
			opts.CorsAllowHeaders = tc.params[4]
			opts.CorsExposeHeaders = tc.params[5]
		}
		opts.CorsAllowCredentials = tc.allowCredentials

		gotRoute, err := MakeRouteConfig(&configinfo.ServiceInfo{
			Name:    "test-api",
			Options: opts,
		})
		if tc.wantedError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
				t.Errorf("Test (%s): expected err: %v, got: %v", tc.desc, tc.wantedError, err)
			}
			continue
		}

		gotHost := gotRoute.GetVirtualHosts()
		if len(gotHost) != 1 {
			t.Errorf("Test (%v): got expected number of virtual host", tc.desc)
		}
		gotCors := gotHost[0].GetCors()
		if !proto.Equal(gotCors, tc.wantCorsPolicy) {
			t.Errorf("Test (%v): makeRouteConfig failed, got Cors: %v, want: %v", tc.desc, gotCors, tc.wantCorsPolicy)
		}
	}
}

// Used to generate a oversize cors origin regex or a oversize wildcard uri template.
func getOverSizeRegexForTest() string {
	overSizeRegex := ""
	for i := 0; i < 333; i += 1 {
		// Use "/**" as it is a replacement token for wildcard uri template.
		overSizeRegex += "/**"
	}
	return overSizeRegex
}
