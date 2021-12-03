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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
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
		enableOperationNameHeader     bool
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
            "operation": "ingress Echo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
		},
		{
			desc:                          "Enable Strict Transport Security for remote backend",
			enableStrictTransportSecurity: true,
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
								Get: "/foo",
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
          "decorator": {
            "operation": "ingress Foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
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
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
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
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
		},
		{
			desc: "Wildcard paths and wildcard http method for remote backend",
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
            "operation": "ingress Foo"
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/[^\\/:]+/test/[^:]*\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
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
								Get: "/v1/shelves/{shelves=*}",
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelves=*}",
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
          "decorator": {
            "operation": "ingress ListShelves"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/shelves/[^\\/:]+\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.ListShelves",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress CreateShelf"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/shelves/[^\\/:]+\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.CreateShelf",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/v1/shelves/{shelves=*}"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/v1/shelves/{shelves=*}\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/v1/shelves/[^\\/:]+\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
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
								Get: "/foo",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
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
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar/"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
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
								Get: "/foo/{x=*}",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
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
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar/"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress ESPv2_Autogenerated_CORS_bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "path": "/bar"
          },
          "name": "testapi.ESPv2_Autogenerated_CORS_bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress ESPv2_Autogenerated_CORS_bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "path": "/bar/"
          },
          "name": "testapi.ESPv2_Autogenerated_CORS_bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress ESPv2_Autogenerated_CORS_foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          },
          "name": "testapi.ESPv2_Autogenerated_CORS_foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/{x=*}"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/{x=*}\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
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
								Get: "/foo",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
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
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar/"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
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
								Get: "/foo",
							},
						},
						{
							Selector: "testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
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
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/bar/"
          },
          "name": "testapi.bar",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/bar/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
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
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
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
								Get: "/foo/{abc=*}",
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
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/"
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          },
          "name": "testapi.foo",
          "route": {
            "cluster": "backend-cluster-testapipb.com:443",
            "hostRewriteLiteral": "testapipb.com",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/{abc=*}"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/{abc=*}\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
		},
		// In this test, the route configs will be in the order of
		//    GET /foo/bar
		//    * /foo/bar,
		//    GET /foo/*
		//    GET /foo/*/bar
		//    GET /foo/**/bar
		//    GET /foo/**:verb
		//    GET /foo/**
		{
			desc:                          "Order route match config",
			enableStrictTransportSecurity: true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
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
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Foo", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/**",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Foo", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/*",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Bar", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/**:verb",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Bar", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/bar",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Bar", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/*/bar",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Foo", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/foo/**/bar",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Bar", testApiName),
						Pattern: &annotationspb.HttpRule_Custom{
							Custom: &annotationspb.CustomHttpPattern{
								Path: "/foo/bar",
								Kind: "*",
							},
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
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/bar"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/foo/bar/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "path": "/foo/bar"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "path": "/foo/bar/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+/bar\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*/bar\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Bar"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*\\/?:verb$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Bar",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Foo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Foo",
          "responseHeadersToAdd": [
            {
              "header": {
                "key": "Strict-Transport-Security",
                "value": "max-age=31536000; includeSubdomains"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/bar"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/foo/bar/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/*"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/*\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/*/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/*/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^\\/:]+/bar\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/**/bar"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/**/bar\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*/bar\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/**:verb"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/**:verb\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*\\/?:verb$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/foo/**"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/foo/**\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/foo/[^:]*\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc:                          "Use duplicate http template",
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
							Get: "/{echoSize=*}",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/{echoId=*}",
						},
					},
				},
				},
			},
			wantedError: "fail to sort route match, endpoints.examples.bookstore.Bookstore.Echo has duplicate http pattern `GET /{echoId=*}`",
		},
		{
			desc:                      "Enable operation name header",
			enableOperationNameHeader: true,
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
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
            "operation": "ingress Echo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo",
          "requestHeadersToAdd": [
            {
              "append": false,
              "header": {
                "key": "X-Endpoint-Api-Operation-Name",
                "value": "endpoints.examples.bookstore.Bookstore.Echo"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo",
          "requestHeadersToAdd": [
            {
              "append": false,
              "header": {
                "key": "X-Endpoint-Api-Operation-Name",
                "value": "endpoints.examples.bookstore.Bookstore.Echo"
              }
            }
          ],
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.EnableHSTS = tc.enableStrictTransportSecurity
			opts.EnableOperationNameHeader = tc.enableOperationNameHeader
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gotRoute, err := makeRouteConfig(fakeServiceInfo)
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
				t.Errorf("makeRouteConfig failed, \n %v", err)
			}
		})
	}
}

func TestMakeFallbackRoute(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		params            []string
		wantRouteConfig   string
	}{
		{
			desc: "generate 404/405 fallback routes for exact path",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo_Get",
							},
							{
								Name: "Echo_Post",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo_Post", testApiName),
						Pattern: &annotationspb.HttpRule_Post{
							Post: "/echo",
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
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "generate 404/405 fallback routes for regex",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo_Get",
							},
							{
								Name: "Echo_Post",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo/{id}",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo_Post", testApiName),
						Pattern: &annotationspb.HttpRule_Post{
							Post: "/echo/{id}",
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
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/echo/[^\\/:]+\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/echo/[^\\/:]+\\/?$"
            }
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo/{id}"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo/{id}\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "safeRegex": {
              "googleRe2": {},
              "regex": "^/echo/[^\\/:]+\\/?$"
            }
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}
`,
		},
		{
			desc: "ensure the order of backend routes, 405 routes, cors default allow_origin=* routes, and 404 route",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo_Get",
							},
							{
								Name: "Echo_Post",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo_Post", testApiName),
						Pattern: &annotationspb.HttpRule_Post{
							Post: "/echo",
						},
					},
				},
				},
			},
			params: []string{"basic", "*", "", "GET,POST,PUT,OPTIONS", "", "", "2m"},
			wantRouteConfig: `
{
  "name": "local_route",
  "virtualHosts": [
    {
      "cors": {
        "allowCredentials": false,
        "allowMethods": "GET,POST,PUT,OPTIONS",
        "allowOriginStringMatch": [
          {
            "exact": "*"
          }
        ],
        "maxAge": "120"
      },
      "domains": [
        "*"
      ],
      "name": "backend",
      "routes": [
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              },
              {
                "name": "origin",
                "presentMatch": true
              },
              {
                "name": "access-control-request-method",
                "presentMatch": true
              }
            ],
            "prefix": "/"
          },
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "directResponse": {
            "body": {
              "inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
            },
            "status": 400
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "prefix": "/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "ensure the order of backend routes, 405 routes, cors exact origin routes, and 404 route",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo_Get",
							},
							{
								Name: "Echo_Post",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo_Post", testApiName),
						Pattern: &annotationspb.HttpRule_Post{
							Post: "/echo",
						},
					},
				},
				},
			},
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", "", "2m"},
			wantRouteConfig: `
{
  "name": "local_route",
  "virtualHosts": [
    {
      "cors": {
        "allowCredentials": false,
        "allowMethods": "GET,POST,PUT,OPTIONS",
        "allowOriginStringMatch": [
          {
            "exact": "http://example.com"
          }
        ],
        "maxAge": "120"
      },
      "domains": [
        "*"
      ],
      "name": "backend",
      "routes": [
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              },
              {
                "name": "origin",
                "stringMatch":{"exact":"http://example.com"}
              },
              {
                "name": "access-control-request-method",
                "presentMatch": true
              }
            ],
            "prefix": "/"
          },
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "directResponse": {
            "body": {
              "inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
            },
            "status": 400
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "prefix": "/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "ensure the order of backend routes, 405 routes, cors regex origin routes, and 404 route",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Echo_Get",
							},
							{
								Name: "Echo_Post",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Echo_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Echo_Post", testApiName),
						Pattern: &annotationspb.HttpRule_Post{
							Post: "/echo",
						},
					},
				},
				},
			},
			params: []string{"cors_with_regex", "", ".*", "GET,POST,PUT,OPTIONS", "", "", "2m"},
			wantRouteConfig: `
{
  "name": "local_route",
  "virtualHosts": [
    {
      "cors": {
        "allowCredentials": false,
        "allowMethods": "GET,POST,PUT,OPTIONS",
        "allowOriginStringMatch": [
          {
            "safeRegex": {
              "googleRe2": {},
              "regex": ".*"
            }
          }
        ],
        "maxAge": "120"
      },
      "domains": [
        "*"
      ],
      "name": "backend",
      "routes": [
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Echo_Post"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"POST"},
                "name": ":method"
              }
            ],
            "path": "/echo/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Echo_Post",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              },
              {
                "name": "origin",
                "safeRegexMatch": {
                  "googleRe2": {},
                  "regex": ".*"
                }
              },
              {
                "name": "access-control-request-method",
                "presentMatch": true
              }
            ],
            "prefix": "/"
          },
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
          }
        },
        {
          "decorator": {
            "operation": "ingress"
          },
          "directResponse": {
            "body": {
              "inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
            },
            "status": 400
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"OPTIONS"},
                "name": ":method"
              }
            ],
            "prefix": "/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/echo"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/echo/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
          }
        }
      ]
    }
  ]
}`,
		},
		{
			desc: "span name length check",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "Long_Get",
							},
							{
								Name: "Short_Get",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: fmt.Sprintf("%s.Long_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template",
						},
					},
					{
						Selector: fmt.Sprintf("%s.Short_Get", testApiName),
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/this-is-short-uri-template",
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
          "decorator": {
            "operation": "ingress Short_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/this-is-short-uri-template"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Short_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Short_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/this-is-short-uri-template/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Short_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Long_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Long_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress Long_Get"
          },
          "match": {
            "headers": [
              {
                "stringMatch":{"exact":"GET"},
                "name": ":method"
              }
            ],
            "path": "/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/"
          },
          "name": "endpoints.examples.bookstore.Bookstore.Long_Get",
          "route": {
            "cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
            "idleTimeout": "300s",
            "retryPolicy": {
              "numRetries": 1,
              "retryOn": "reset,connect-failure,refused-stream"
            },
            "timeout": "15s"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/this-is-short-uri-template"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/this-is-short-uri-template\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/this-is-short-uri-template"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/this-is-short-uri-template"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/this-is-short-uri-template\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/this-is-short-uri-template/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-temp..."
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownHttpMethodForPath_/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-temp..."
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is matched to the defined url template \"/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template\" but its http method is not allowed"
            },
            "status": 405
          },
          "match": {
            "path": "/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/this-is-super-long-uri-template/"
          }
        },
        {
          "decorator": {
            "operation": "ingress UnknownOperationName"
          },
          "directResponse": {
            "body": {
              "inlineString": "The current request is not defined by this API."
            },
            "status": 404
          },
          "match": {
            "prefix": "/"
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

			if tc.params != nil {
				opts.CorsPreset = tc.params[0]
				opts.CorsAllowOrigin = tc.params[1]
				opts.CorsAllowOriginRegex = tc.params[2]
				opts.CorsAllowMethods = tc.params[3]
				opts.CorsAllowHeaders = tc.params[4]
				opts.CorsExposeHeaders = tc.params[5]
				var err error
				opts.CorsMaxAge, err = time.ParseDuration(tc.params[6])
				if err != nil {
					t.Fatal(err)
				}
			}
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gotRoute, err := makeRouteConfig(fakeServiceInfo)
			if err != nil {
				t.Fatalf("got error: %v", err)
			}

			marshaler := &jsonpb.Marshaler{}
			gotConfig, err := marshaler.MarshalToString(gotRoute)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(tc.wantRouteConfig, gotConfig); err != nil {
				t.Errorf("Test(%s): makeRouteConfig failed, \n %v", tc.desc, err)
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
			params:      []string{"basic", "", `^https?://.+\\.example\\.com\/?$`, "", "", "", "2m"},
			wantedError: "cors_allow_origin cannot be empty when cors_preset=basic",
		},
		{
			desc:        "Incorrect configured  Cors",
			params:      []string{"", "", "", "GET", "", "", "2m"},
			wantedError: "cors_preset must be set in order to enable CORS support",
		},
		{
			desc:        "Incorrect configured regex Cors",
			params:      []string{"cors_with_regexs", "", `^https?://.+\\.example\\.com\/?$`, "", "", "", "2m"},
			wantedError: `cors_preset must be either "basic" or "cors_with_regex"`,
		},
		{
			desc:        "Oversize cors origin regex",
			params:      []string{"cors_with_regex", "", getOverSizeRegexForTest(), "", "Origin,Content-Type,Accept", "", "2m"},
			wantedError: `invalid cors origin regex: regex program size`,
		},
		{
			desc:   "Correct configured basic Cors, with allow methods",
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", "", "2m"},
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
				MaxAge:           "120",
			},
		},
		{
			desc:   "Correct configured regex Cors, with allow headers",
			params: []string{"cors_with_regex", "", `^https?://.+\\.example\\.com\/?$`, "", "Origin,Content-Type,Accept", "", "2m"},
			wantCorsPolicy: &routepb.CorsPolicy{
				AllowOriginStringMatch: []*matcher.StringMatcher{
					{
						MatchPattern: &matcher.StringMatcher_SafeRegex{
							SafeRegex: &matcher.RegexMatcher{
								EngineType: &matcher.RegexMatcher_GoogleRe2{
									GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
								},
								Regex: `^https?://.+\\.example\\.com\/?$`,
							},
						},
					},
				},
				AllowHeaders:     "Origin,Content-Type,Accept",
				AllowCredentials: &wrapperspb.BoolValue{Value: false},
				MaxAge:           "120",
			},
		},
		{
			desc:             "Correct configured regex Cors, with expose headers",
			params:           []string{"cors_with_regex", "", `^https?://.+\\.example\\.com\/?$`, "", "", "Content-Length", "2m"},
			allowCredentials: true,
			wantCorsPolicy: &routepb.CorsPolicy{
				AllowOriginStringMatch: []*matcher.StringMatcher{
					{
						MatchPattern: &matcher.StringMatcher_SafeRegex{
							SafeRegex: &matcher.RegexMatcher{
								EngineType: &matcher.RegexMatcher_GoogleRe2{
									GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
								},
								Regex: `^https?://.+\\.example\\.com\/?$`,
							},
						},
					},
				},
				ExposeHeaders:    "Content-Length",
				AllowCredentials: &wrapperspb.BoolValue{Value: true},
				MaxAge:           "120",
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
			var err error
			opts.CorsMaxAge, err = time.ParseDuration(tc.params[6])
			if err != nil {
				t.Fatal(err)
			}
		}
		opts.CorsAllowCredentials = tc.allowCredentials

		gotRoute, err := makeRouteConfig(&configinfo.ServiceInfo{
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

func TestHeadersToAdd(t *testing.T) {
	testData := []struct {
		desc                  string
		addRequestHeaders     string
		appendRequestHeaders  string
		addResponseHeaders    string
		appendResponseHeaders string
		wantedError           string
		wantedRequestHeaders  []*corepb.HeaderValueOption
		wantedResponseHeaders []*corepb.HeaderValueOption
	}{
		{
			desc:              "error case: wrong format",
			addRequestHeaders: "k1",
			wantedError:       "invalid header: k1. should be in key=value format",
		},
		{
			desc:              "error case: empty key",
			addRequestHeaders: "=value",
			wantedError:       "header key should not be empty for: =value",
		},
		{
			desc:              "OK case, empty value",
			addRequestHeaders: "k1=",
			wantedRequestHeaders: []*corepb.HeaderValueOption{
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key: "k1",
					},
					Append: &wrapperspb.BoolValue{
						Value: false,
					},
				},
			},
		},
		{
			desc:                  "basic case: 3 headers for request and response",
			addRequestHeaders:     "k1=v1;k2=v2",
			appendRequestHeaders:  "k3=v3",
			addResponseHeaders:    "kk1=vv1",
			appendResponseHeaders: "kk3=vv3;kk4=vv4",
			wantedRequestHeaders: []*corepb.HeaderValueOption{
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "k1",
						Value: "v1",
					},
					Append: &wrapperspb.BoolValue{
						Value: false,
					},
				},
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "k2",
						Value: "v2",
					},
					Append: &wrapperspb.BoolValue{
						Value: false,
					},
				},
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "k3",
						Value: "v3",
					},
					Append: &wrapperspb.BoolValue{
						Value: true,
					},
				},
			},
			wantedResponseHeaders: []*corepb.HeaderValueOption{
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "kk1",
						Value: "vv1",
					},
					Append: &wrapperspb.BoolValue{
						Value: false,
					},
				},
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "kk3",
						Value: "vv3",
					},
					Append: &wrapperspb.BoolValue{
						Value: true,
					},
				},
				&corepb.HeaderValueOption{
					Header: &corepb.HeaderValue{
						Key:   "kk4",
						Value: "vv4",
					},
					Append: &wrapperspb.BoolValue{
						Value: true,
					},
				},
			},
		},
	}

	for _, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.AddRequestHeaders = tc.addRequestHeaders
		opts.AppendRequestHeaders = tc.appendRequestHeaders
		opts.AddResponseHeaders = tc.addResponseHeaders
		opts.AppendResponseHeaders = tc.appendResponseHeaders

		gotRoute, err := makeRouteConfig(&configinfo.ServiceInfo{
			Name:    "test-api",
			Options: opts,
		})
		if tc.wantedError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
				t.Errorf("Test (%s): expected err: %v, got: %v", tc.desc, tc.wantedError, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Test (%s): makeRouteConfig got error: %v", tc.desc, err)
		}

		if len(tc.wantedRequestHeaders) != len(gotRoute.RequestHeadersToAdd) {
			t.Errorf("Test (%v): makeRouteConfig failed, RequestHeadersAdd diff len: %v, want: %v", tc.desc, len(gotRoute.RequestHeadersToAdd), len(tc.wantedRequestHeaders))
		} else {
			for idx, want := range tc.wantedRequestHeaders {
				if !proto.Equal(gotRoute.RequestHeadersToAdd[idx], want) {
					t.Errorf("Test (%v): makeRouteConfig failed, RequestHeadersAdd(%v): %v, want: %v", tc.desc, idx, gotRoute.RequestHeadersToAdd[idx], want)
				}
			}
		}
		if len(tc.wantedResponseHeaders) != len(gotRoute.ResponseHeadersToAdd) {
			t.Errorf("Test (%v): makeRouteConfig failed, ResponseHeadersAdd diff len: %v, want: %v", tc.desc, len(gotRoute.ResponseHeadersToAdd), len(tc.wantedResponseHeaders))
		} else {
			for idx, want := range tc.wantedResponseHeaders {
				if !proto.Equal(gotRoute.ResponseHeadersToAdd[idx], want) {
					t.Errorf("Test (%v): makeRouteConfig failed, ResponeHeadersToAdd(%v): %v, want: %v", tc.desc, idx, gotRoute.ResponseHeadersToAdd[idx], want)
				}
			}
		}
	}
}

// Used to generate a oversize cors origin regex or a oversize uri template.
func getOverSizeRegexForTest() string {
	overSizeRegex := ""
	for i := 0; i < 333; i += 1 {
		// Form regex in a way that it cannot be simplified.
		overSizeRegex += "[abc]+123"
	}
	return overSizeRegex
}
