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

func TestNewDenyInvalidMethodRouteGenFromOPConfig(t *testing.T) {
	testdata := []struct {
		wrappedGens []routegen.RouteGeneratorOPFactory
		*routegentest.SuccessOPTestCase
	}{
		{
			wrappedGens: nil,
			SuccessOPTestCase: &routegentest.SuccessOPTestCase{
				Desc: "No routes generated when no underlying wrapped route gens",
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
				OptsIn: options.ConfigGeneratorOptions{
					Healthz: "/healthz",
				},
				WantHostConfig: `{}`,
			},
		},
		{
			wrappedGens: []routegen.RouteGeneratorOPFactory{
				routegen.NewDirectResponseHealthCheckRouteGenFromOPConfig,
			},
			SuccessOPTestCase: &routegentest.SuccessOPTestCase{
				Desc: "Routes generated for healthz (only healthz route gen provided)",
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
				OptsIn: options.ConfigGeneratorOptions{
					Healthz: "/healthz",
				},
				WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/healthz"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/healthz\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/healthz"
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/healthz"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/healthz\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/healthz/"
      }
    }
  ]
}
`,
			},
		},
		{
			wrappedGens: []routegen.RouteGeneratorOPFactory{
				routegen.NewProxyBackendRouteGenFromOPConfig,
			},
			SuccessOPTestCase: &routegentest.SuccessOPTestCase{
				Desc: "Routes generated for single HTTP path (only backend route gen provided)",
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
        "operation":"ingress UnknownHttpMethodForPath_/echo"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/echo"
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/echo"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/echo\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/echo/"
      }
    }
  ]
}
`,
			},
		},
		{
			// In this test, the route configs will be in the order of:
			//    GET /foo/bar
			//    * /foo/bar, -- Not generated (no unknown method)
			//    GET /foo/*
			//    GET /foo/*/bar
			//    GET /foo/**/bar
			//    GET /foo/**:verb
			//    GET /foo/**
			//		GET /healthz
			wrappedGens: []routegen.RouteGeneratorOPFactory{
				routegen.NewProxyBackendRouteGenFromOPConfig,
				routegen.NewDirectResponseHealthCheckRouteGenFromOPConfig,
			},
			SuccessOPTestCase: &routegentest.SuccessOPTestCase{
				Desc: "Order route match config for backend routes and healthz route",
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
										Pattern: &annotationspb.HttpRule_Get{
											Get: "/foo/*",
										},
									},
									{
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
										Pattern: &annotationspb.HttpRule_Get{
											Get: "/foo/bar",
										},
									},
									{
										Pattern: &annotationspb.HttpRule_Get{
											Get: "/foo/*/bar",
										},
									},
									{
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
				OptsIn: options.ConfigGeneratorOptions{
					Healthz: "healthz",
				},
				WantHostConfig: `
{
  "routes":[
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/bar"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/bar\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/foo/bar"
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/bar"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/bar\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/foo/bar/"
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/*"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/*\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "safeRegex":{
          "regex":"^/foo/[^\\/]+\\/?$"
        }
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/*/bar"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/*/bar\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "safeRegex":{
          "regex":"^/foo/[^\\/]+/bar\\/?$"
        }
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/**/bar"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/**/bar\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "safeRegex":{
          "regex":"^/foo/.*/bar\\/?$"
        }
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/**:verb"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/**:verb\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "safeRegex":{
          "regex":"^/foo/.*\\/?:verb$"
        }
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/foo/**"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/foo/**\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "safeRegex":{
          "regex":"^/foo/.*\\/?$"
        }
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/healthz"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/healthz\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/healthz"
      }
    },
    {
      "decorator":{
        "operation":"ingress UnknownHttpMethodForPath_/healthz"
      },
      "directResponse":{
        "body":{
          "inlineString":"The current request is matched to the defined url template \"/healthz\" but its http method is not allowed"
        },
        "status":405
      },
      "match":{
        "path":"/healthz/"
      }
    }
  ]
}
`,
			},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (routegen.RouteGenerator, error) {
			return routegen.NewDenyInvalidMethodRouteGenFromOPConfig(serviceConfig, opts, tc.wrappedGens)
		})
	}
}
