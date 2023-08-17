package routegen_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen/routegentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestNewBackendRouteGensFromOPConfig(t *testing.T) {
	testdata := []routegentest.SuccessOPTestCase{
		{
			Desc: "Happy path simple OpenAPI service",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
							Selector: "endpoints.examples.bookstore.Bookstore.Echo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/echo",
							},
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"GET"
            }
          }
        ],
        "path":"/echo"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"GET"
            }
          }
        ],
        "path":"/echo/"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "Wildcard paths and wildcard http method for remote backend",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
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
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.Foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &servicepb.BackendRule_JwtAudience{
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
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress Foo"
      },
      "match":{
        "safeRegex":{
          "regex":"^/v1/[^\\/]+/test/.*\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "Wildcard paths with disallowing colon in wildcard segment",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
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
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.Foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &servicepb.BackendRule_JwtAudience{
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
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					DisallowColonInWildcardPathSegment: true,
				},
			},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress Foo"
      },
      "match":{
        "safeRegex":{
          "regex":"^/v1/[^\\/:]+/test/[^:]*\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "path_rewrite: multiple http rule url_templates with variable bindings.",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
						{
							Selector:        "endpoints.examples.bookstore.Bookstore.ListShelves",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &servicepb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress ListShelves"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/v1/shelves/[^\\/]+\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.ListShelves",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress CreateShelf"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"POST"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/v1/shelves/[^\\/]+\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.CreateShelf",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "path_rewrite: multiple http rule url_templates without variable bindings.",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
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
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &servicepb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "testapi.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &servicepb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/bar"
      },
      "name":"testapi.bar",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/bar/"
      },
      "name":"testapi.bar",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/foo"
      },
      "name":"testapi.foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/foo/"
      },
      "name":"testapi.foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "Multiple http rules for one selector",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
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
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									Selector: "testapi.foo",
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/foo/{abc=*}",
									},
								},
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "testapi.foo",
							Address:         "https://testapipb.com/",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/"
      },
      "name":"testapi.foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/[^\\/]+\\/?$"
        }
      },
      "name":"testapi.foo",
      "route":{
        "cluster":"backend-cluster-testapipb.com:443",
        "hostRewriteLiteral":"testapipb.com",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			// In this test, the route configs will be in the order of
			//    GET /foo/bar
			//    * /foo/bar,
			//    GET /foo/*
			//    GET /foo/*/bar
			//    GET /foo/**/bar
			//    GET /foo/**:verb
			//    GET /foo/**
			Desc: "Order route match config",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
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
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.Foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo/**",
							},
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									Selector: "endpoints.examples.bookstore.Bookstore.Foo",
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/foo/*",
									},
								},
								{
									Selector: "endpoints.examples.bookstore.Bookstore.Foo",
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/foo/**/bar",
									},
								},
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.Bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo/**:verb",
							},
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									Selector: "endpoints.examples.bookstore.Bookstore.Bar",
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/foo/bar",
									},
								},
								{
									Selector: "endpoints.examples.bookstore.Bookstore.Bar",
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/foo/*/bar",
									},
								},
								{
									Selector: "endpoints.examples.bookstore.Bookstore.Bar",
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
				},
			},
			OptsIn: options.ConfigGeneratorOptions{},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/foo/bar"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "path":"/foo/bar/"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "path":"/foo/bar"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "path":"/foo/bar/"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/[^\\/]+\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Foo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/[^\\/]+/bar\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/.*/bar\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Foo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Bar"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/.*\\/?:verb$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Bar",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Foo"
      },
      "match":{
        "headers":[
          {
            "stringMatch":{
              "exact":"GET"
            },
            "name":":method"
          }
        ],
        "safeRegex":{
          "regex":"^/foo/.*\\/?$"
        }
      },
      "name":"endpoints.examples.bookstore.Bookstore.Foo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
		{
			Desc: "gRPC support required",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
						},
					},
				},
				Http: &annotationspb.Http{Rules: []*annotationspb.HttpRule{
					{
						Selector: "endpoints.examples.bookstore.Bookstore.Echo",
						Pattern: &annotationspb.HttpRule_Get{
							Get: "/echo",
						},
					},
				},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://grpc-backend:8080",
			},
			WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"GET"
            }
          }
        ],
        "path":"/echo"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"GET"
            }
          }
        ],
        "path":"/echo/"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"POST"
            }
          }
        ],
        "path":"/endpoints.examples.bookstore.Bookstore/Echo"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    },
    {
      "decorator":{
        "operation":"ingress Echo"
      },
      "match":{
        "headers":[
          {
            "name":":method",
            "stringMatch":{
              "exact":"POST"
            }
          }
        ],
        "path":"/endpoints.examples.bookstore.Bookstore/Echo/"
      },
      "name":"endpoints.examples.bookstore.Bookstore.Echo",
      "route":{
        "cluster":"backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
        "idleTimeout":"300s",
        "retryPolicy":{
          "numRetries":1,
          "retryOn":"reset,connect-failure,refused-stream"
        },
        "timeout":"15s"
      }
    }
  ]
}
`,
		},
	}
	for _, tc := range testdata {
		tc.RunTest(t, routegen.NewBackendRouteGensFromOPConfig)
	}
}
