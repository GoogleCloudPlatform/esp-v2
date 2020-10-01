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
