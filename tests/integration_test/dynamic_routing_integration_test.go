// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
)

func NewDynamicRoutingTestEnv(port uint16) *env.TestEnv {
	s := env.NewTestEnv(port, platform.EchoRemote)
	return s
}

func TestDynamicRouting(t *testing.T) {
	t.Parallel()
	s := NewDynamicRoutingTestEnv(platform.TestDynamicRouting)
	defer s.TearDown(t)

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		path          string
		method        string
		message       string
		wantResp      string
		httpCallError error
	}{
		{
			desc:     "Succeed, no path translation (no re-routing needed)",
			path:     "/echo?key=api-key",
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct",
			path:     "/pet/123/num/987",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/getpetbyid?pet_id=123&number=987"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct with escaped path segment",
			path:     "/pet/a%20b/num/9%3B8",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/getpetbyid?pet_id=a%20b&number=9%3B8"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation with empty path",
			path:     "/empty_path",
			method:   "POST",
			wantResp: `{"RequestURI":"/"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct, original URL has query parameters, original query parameters should appear first and query parameters converted from path parameters appear later",
			path:     "/pet/31/num/565?lang=US&zone=us-west1",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/getpetbyid?lang=US&zone=us-west1&pet_id=31&number=565"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct, original URL has query parameters, original query parameters should appear first and query parameters converted from path parameters appear later. Both have escaped characters",
			path:     "/pet/a%20b/num/9%3B8?lang=U%20S&zone=us%3Bwest",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/getpetbyid?lang=U%20S&zone=us%3Bwest&pet_id=a%20b&number=9%3B8"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation with snake case is correct",
			path:     "/shelves/123/books/info/987",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/bookinfo?SHELF=123&BOOK=987"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation with snake case is correct, supports {foo.bar} style path, if corresponding jsonName not found, origin snake case path is used.",
			path:     "/shelves/221/books/id/2019",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/bookid?SHELF.i_d=221&BOOK.id=2019"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct for cases that does not have path parameter",
			path:     "/shelves",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/shelves"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct for cases that does not have path parameter but has query parameter",
			path:     "/shelves?q=story",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/shelves?q=story"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct for cases that does not have path parameter but has query parameter and escaped characters",
			path:     "/shelves?q=story%3Bbooks",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/shelves?q=story%3Bbooks"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, appends original URL to backend address (https://domain/base/path)",
			path:     "/searchpet",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/searchpet/searchpet"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation with query parameter is correct, appends original URL to backend address (https://domain/base/path)",
			path:     "/searchpet?timezone=PST&lang=US",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/searchpet/searchpet?timezone=PST&lang=US"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, appends original URL to backend address that ends with slash (https://domain/base/path/)",
			path:     "/searchdog",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/searchdogs/searchdog"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation with query parameter is correct, appends original URL to backend address that ends with slash (https://domain/base/path/)",
			path:     "/searchdog?timezone=UTC",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/searchdogs/searchdog?timezone=UTC"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation with query parameter is correct, appends original URL to backend address that ends with slash (https://domain/base/path/), query parameter with escaped characters",
			path:     "/searchdog?timezone=U%20C%3BT",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/searchdogs/searchdog?timezone=U%20C%3BT"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, original URL has path parameters",
			path:     "/pets/cat/year/2018",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/listpet/pets/cat/year/2018"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, original URL has path parameters and escaped characters",
			path:     "/pets/c%20t/year/2018%3B2019",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/listpet/pets/c%20t/year/2018%3B2019"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, original URL has path parameters and query parameters",
			path:     "/pets/dog/year/2019?lang=US&zone=us-west1",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/listpet/pets/dog/year/2019?lang=US&zone=us-west1"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, original URL has path parameters and query parameters. Both have escaped characters",
			path:     "/pets/d%20g/year/2019%3B2020?lang=U%20S&zone=us%3Bwest",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/listpet/pets/d%20g/year/2019%3B2020?lang=U%20S&zone=us%3Bwest"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, backend address is root path with slash (https://domain/)",
			path:     "/searchrootwithslash",
			method:   "GET",
			wantResp: `{"RequestURI":"/searchrootwithslash"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation with query parameter is correct, backend address is root path with slash (https://domain/)",
			path:     "/searchroot?zone=us-central1&lang=en",
			method:   "GET",
			wantResp: `{"RequestURI":"/searchroot?zone=us-central1&lang=en"}`,
		},
		{
			desc:          "Fail, there is not backend rule specified for this path",
			path:          "/searchdogs",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS with single segments in double wildcards",
			path:     "/wildcard/a/1/b/2/c/3",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?name=2"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS with multiple segments in double wildcards",
			path:     "/wildcard/a/1/b/2/c/3/4/5",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?name=2"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS with no segments in double wildcards",
			path:     "/wildcard/a/1/b/2/c/",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?name=2"}`,
		},
		{
			desc:          "Fail, CONSTANT_ADDRESS with missing last segment in double wildcards",
			path:          "/wildcard/a/1/b/2/c",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		{
			desc:          "Fail, CONSTANT_ADDRESS with no segments in single wildcards",
			path:          "/wildcard/a/b/2/c/",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		{
			desc:          "Fail, CONSTANT_ADDRESS with multiple segments in single wildcards",
			path:          "/wildcard/a/1/2/3/b/4/c/",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS with query params for double wildcards",
			path:     "/wildcard/a/1/b/2/c/3/4/5?key=value",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?key=value&name=2"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS with query params for variable bindings with multiple segments",
			path:     "/field_path/a/1/b/2/x/9/8/7:upload",
			method:   "GET",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?s_1=a/1/b/2&s_2=x/9/8/7"}`,
		},
		{
			desc:          "Fail, CONSTANT_ADDRESS but verb is incorrect",
			path:          "/field_path/a/1/b/2/x/9/8/7:incorrect_verb",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		{
			desc:          "Fail, CONSTANT_ADDRESS but first variable binding is incorrect",
			path:          "/field_path/a/1/incorrect_segment/2/x/9/8/7:upload",
			method:        "GET",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
		},
		// The following three test cases cover when requests can be matched by multiple
		// http patterns, the most specific will be matched.
		{
			desc:     "Route match ordering - Match http pattern `POST /allow-all/exact-match`",
			path:     "/allow-all/exact-match",
			method:   "POST",
			message:  "hello",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard"}`,
		},
		{
			desc:     "Route match ordering - Match http pattern `POST /allow-all/{single_wildcard=*}`",
			path:     "/allow-all/should-match-single-wildcard",
			method:   "POST",
			message:  "hello",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?single_wildcard=should-match-single-wildcard"}`,
		},
		{
			desc:     "Route match ordering - Match http pattern `POST /allow-all/{double_wildcard=**}`",
			path:     "/allow-all/should-match/double-wildcard",
			method:   "POST",
			message:  "hello",
			wantResp: `{"RequestURI":"/dynamicrouting/const_wildcard?double_wildcard=should-match/double-wildcard"}`,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
			gotResp, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

			if tc.httpCallError == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if err == nil {
					t.Fatalf("got no error, expected err: %v", tc.httpCallError)
				}

				if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
					t.Fatalf("expected Http call error: %v, got: %v", tc.httpCallError, err)
				}
				return
			}
			gotRespStr := string(gotResp)
			if err := util.JsonEqual(tc.wantResp, gotRespStr); err != nil {
				t.Errorf("fail: \n %s", err)
			}
		})
	}
}

func TestDynamicRoutingWithAllowCors(t *testing.T) {
	t.Parallel()

	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsOrigin := "http://cloud.google.com"

	respHeaderMap := make(map[string]string)
	respHeaderMap["Access-Control-Allow-Origin"] = "*"
	respHeaderMap["Access-Control-Allow-Methods"] = "GET, OPTIONS"
	respHeaderMap["Access-Control-Allow-Headers"] = "Authorization"
	respHeaderMap["Access-Control-Expose-Headers"] = "Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	respHeaderMap["Access-Control-Allow-Credentials"] = "true"

	s := NewDynamicRoutingTestEnv(platform.TestDynamicRoutingWithAllowCors)
	s.SetAllowCors()

	defer s.TearDown(t)

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		url            string
		wantRequestUrl string
	}{
		{
			// when allowCors, passes preflight CORS request to backend.
			desc: "TestDynamicRoutingWithAllowCors Succeed, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simplegetcors"),
		},
		{
			// when allowCors, passes preflight CORS request without valid jwt token to backend,
			// even the origin method requires authentication.
			desc: "TestDynamicRoutingWithAllowCors Succeed without jwt token, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/auth/info/firebase"),
		},
		{
			// when allowCors, passes preflight CORS request to backend.
			// Test the path extraction for the auto-generated method for cors requests.
			// Original method:
			//   1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase
			//   Get: "/shelves/{s_h_e_l_f}/books/info/{b_o_o_k}",
			desc:           "TestDynamicRoutingWithAllowCors Succeed with path extraction, response has CORS headers",
			url:            fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/shelves/8/books/info/24"),
			wantRequestUrl: "/dynamicrouting/bookinfo?SHELF=8&BOOK=24",
		},
	}

	for idx, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader, "")
		if err != nil {
			t.Errorf("TestDynamicRoutingWithAllowCors Failed")
			t.Fatal(err)
		}

		for key, value := range respHeaderMap {
			if respHeader.Get(key) != value {
				t.Errorf("Test(%v, %s)%s expected: %s, got: %s", idx, tc.desc, key, value, respHeader.Get(key))
			}
		}

		if tc.wantRequestUrl != "" {
			if getRequestUrl := respHeader.Get("Request-Url"); getRequestUrl != tc.wantRequestUrl {
				t.Errorf("Test(%v, %s) expected request url: %s, got request url: %s", idx, tc.desc, tc.wantRequestUrl, getRequestUrl)
			}
		}
	}
}

func TestDynamicRoutingCorsByEnvoy(t *testing.T) {
	t.Parallel()
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsAllowOriginValue := "http://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "DNT,User-Agent,Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	respHeaderMap := make(map[string]string)
	respHeaderMap["Access-Control-Allow-Origin"] = corsAllowOriginValue
	respHeaderMap["Access-Control-Allow-Methods"] = corsAllowMethodsValue
	respHeaderMap["Access-Control-Allow-Headers"] = corsAllowHeadersValue
	respHeaderMap["Access-Control-Expose-Headers"] = corsExposeHeadersValue
	respHeaderMap["Access-Control-Allow-Credentials"] = "true"
	dynamicRoutingArgs := utils.CommonArgs()
	dynamicRoutingArgs = append(dynamicRoutingArgs, []string{
		"--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue,
		"--cors_allow_credentials"}...)

	s := NewDynamicRoutingTestEnv(platform.TestDynamicRoutingCorsByEnvoy)
	defer s.TearDown(t)

	if err := s.Setup(dynamicRoutingArgs); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc string
		url  string
	}{
		{
			// preflight CORS request handled by envoy CORS filter
			desc: "Succeed, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simplegetcors"),
		},
		{
			// preflight CORS request handled by envoy CORS filter
			// it is before jwt_authn filter even the origin method requires authentication.
			desc: "Succeed without jwt token, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/auth/info/firebase"),
		},
	}

	for _, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsAllowOriginValue, corsRequestMethod, corsRequestHeader, "")
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range respHeaderMap {
			if respHeader.Get(key) != value {
				t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
			}
		}
	}
}

func TestServiceControlRequestForDynamicRouting(t *testing.T) {
	t.Parallel()

	s := NewDynamicRoutingTestEnv(platform.TestServiceControlRequestForDynamicRouting)
	defer s.TearDown(t)

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		path           string
		message        string
		wantResp       string
		wantScRequests []interface{}
	}{
		{
			desc:     "Succeed, no path translation (no re-routing needed)",
			path:     "/echo?key=api-key",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echo?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ApiVersion:                   "1.0.0",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct, service control check request and report request are correct",
			path:     "/sc/searchpet?key=api-key&timezone=EST",
			message:  "hello",
			wantResp: `{"RequestURI":"/dynamicrouting/sc/searchpet?key=api-key&timezone=EST"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/sc/searchpet?key=api-key&timezone=EST",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ApiVersion:                   "1.0.0",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct, service control check request and report request are correct",
			path:     "/sc/pet/0325/num/2019?key=api-key&lang=en",
			message:  "hello",
			wantResp: `{"RequestURI":"/dynamicrouting?key=api-key&lang=en&pet_id=0325&number=2019"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/sc/pet/0325/num/2019?key=api-key&lang=en",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					ApiVersion:                   "1.0.0",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		var gotResp []byte
		var err error
		gotResp, err = client.DoPost(url, tc.message)

		if err != nil {
			t.Fatalf("Test (%v): %v", tc.desc, err)
		}

		gotRespStr := string(gotResp)

		if err := util.JsonEqual(tc.wantResp, gotRespStr); err != nil {
			t.Errorf("Test (%v) fails: \n %s", tc.desc, err)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err != nil {
			t.Fatalf("Test (%v): GetRequests returns error: %v", tc.desc, err)
		}

		if err := utils.VerifyServiceControlResp(tc.desc, tc.wantScRequests, scRequests); err != nil {
			t.Fatalf("Test (%v): %v", tc.desc, err)
		}
	}
}

func TestDynamicBackendRoutingTLS(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc           string
		path           string
		useWrongCert   bool
		message        string
		wantError      string
		wantScRequests []interface{}
	}{
		{
			desc:         "Success for correct cert ",
			path:         "/sc/searchpet?key=api-key&timezone=EST",
			useWrongCert: false,
			message:      "hello",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/sc/searchpet?key=api-key&timezone=EST",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:         "Fail for incorrect cert ",
			path:         "/sc/searchpet?key=api-key&timezone=EST",
			useWrongCert: true,
			message:      "hello",
			wantError:    "503 Service Unavailable",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/sc/searchpet?key=api-key&timezone=EST",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 503,
					Platform:                     util.GCE,
					Location:                     "test-zone",
					ResponseCodeDetail:           "upstream_reset_before_response_started{connection_failure,TLS_error:_268435581:SSL_routines:OPENSSL_internal:CERTIFICATE_VERIFY_FAILED}",
				},
			},
		},
	}

	for _, tc := range testData {
		func() {
			s := env.NewTestEnv(platform.TestDynamicBackendRoutingTLS, platform.EchoRemote)
			s.UseWrongBackendCertForDR(tc.useWrongCert)
			defer s.TearDown(t)

			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
			var err error
			_, err = client.DoPost(url, tc.message)

			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("Test (%s): failed, %v", tc.desc, err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Test (%s): failed, want error: %v, got error: %v", tc.desc, tc.wantError, err)
				}
			}

			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("Test Desc(%s): GetRequests returns error: %v", tc.desc, err)
			}
			if err := utils.VerifyServiceControlResp(tc.desc, tc.wantScRequests, scRequests); err != nil {
				t.Error(err)
			}
		}()
	}
}

func TestDynamicBackendRoutingMutualTLS(t *testing.T) {
	t.Parallel()

	args := utils.CommonArgs()
	args = append(args, "--ssl_backend_client_cert_path=../env/testdata/")

	testData := []struct {
		desc           string
		path           string
		mtlsCertFile   string
		message        string
		wantError      string
		wantScRequests []interface{}
	}{
		{
			desc:         "Success for correct cert, with same self signed cert",
			path:         "/sc/searchpet?key=api-key&timezone=EST",
			mtlsCertFile: platform.GetFilePath(platform.ServerCert),
			message:      "hello",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/sc/searchpet?key=api-key&timezone=EST",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ApiVersion:                   "1.0.0",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:         "Fail for incorrect cert, backend uses an unmatch cert",
			path:         "/sc/searchpet?key=api-key&timezone=EST",
			mtlsCertFile: platform.GetFilePath(platform.ProxyCert),
			message:      "hello",
			wantError:    "503 Service Unavailable",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					ApiKeyState:                  "VERIFIED",
					URL:                          "/sc/searchpet?key=api-key&timezone=EST",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ApiVersion:                   "1.0.0",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:                   "0",
					ResponseCode:                 503,
					Platform:                     util.GCE,
					Location:                     "test-zone",
					ResponseCodeDetail:           "upstream_reset_before_response_started{connection_failure,TLS_error:_268436498:SSL_routines:OPENSSL_internal:SSLV3_ALERT_BAD_CERTIFICATE}",
				},
			},
		},
	}

	for _, tc := range testData {
		func() {
			s := env.NewTestEnv(platform.TestDynamicBackendRoutingMutualTLS, platform.EchoRemote)
			s.SetBackendMTLSCert(tc.mtlsCertFile)
			defer s.TearDown(t)

			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
			var err error
			_, err = client.DoPost(url, tc.message)

			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("Test (%s): failed, %v", tc.desc, err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Test (%s): failed, want error: %v, got error: %v", tc.desc, tc.wantError, err)
				}
			}

			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("Test Desc(%s): GetRequests returns error: %v", tc.desc, err)
			}
			if err := utils.VerifyServiceControlResp(tc.desc, tc.wantScRequests, scRequests); err != nil {
				t.Errorf("Test Desc(%s): %v", tc.desc, err)
			}
		}()
	}
}

// Tests both TLS and mTLS for gRPC backends.
func TestDynamicGrpcBackendTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc                string
		clientProtocol      string
		methodOrUrl         string
		mtlsCertFile        string
		useWrongBackendCert bool
		header              http.Header
		wantResp            string
		wantError           string
	}{
		{
			desc:           "gRPC client calling gRPCs remote backend succeed",
			clientProtocol: "grpc",
			methodOrUrl:    "GetShelf",
			header:         http.Header{"x-api-key": []string{"api-key"}},
			wantResp:       `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "Http client calling gRPCs remote backend succeed",
			clientProtocol: "http",
			methodOrUrl:    "/v1/shelves/200?key=api-key",
			wantResp:       `{"id":"200","theme":"Classic"}`,
		},
		{
			desc:                "gRPC client calling gRPCs remote backend fail with incorrect cert",
			clientProtocol:      "grpc",
			methodOrUrl:         "GetShelf",
			useWrongBackendCert: true,
			header:              http.Header{"x-api-key": []string{"api-key"}},
			wantError:           "Unavailable",
		},
		{
			desc:                "Http2 client calling gRPCs remote backend fail with incorrect cert",
			clientProtocol:      "http2",
			useWrongBackendCert: true,
			methodOrUrl:         "/v1/shelves/200?key=api-key",
			wantError:           "503 Service Unavailable",
		},
		{
			desc:           "gRPC client calling gRPCs remote backend with mTLS succeed",
			clientProtocol: "grpc",
			methodOrUrl:    "GetShelf",
			mtlsCertFile:   platform.GetFilePath(platform.ServerCert),
			header:         http.Header{"x-api-key": []string{"api-key"}},
			wantResp:       `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "HTTP2 client calling gRPCs remote backend through mTLS failed with incorrect client root cert",
			clientProtocol: "http2",
			methodOrUrl:    "/v1/shelves/200?key=api-key",
			mtlsCertFile:   platform.GetFilePath(platform.ProxyCert),
			header:         http.Header{"x-api-key": []string{"api-key"}},
			wantError:      "503 Service Unavailable",
		},
	}

	for _, tc := range tests {
		args := utils.CommonArgs()
		func() {
			s := env.NewTestEnv(platform.TestDynamicGrpcBackendTLS, platform.GrpcBookstoreRemote)
			defer s.TearDown(t)
			s.UseWrongBackendCertForDR(tc.useWrongBackendCert)
			if tc.mtlsCertFile != "" {
				s.SetBackendMTLSCert(tc.mtlsCertFile)
				args = append(args, "--ssl_backend_client_cert_path=../env/testdata/")
			}

			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall(tc.clientProtocol, addr, "GET", tc.methodOrUrl, "", tc.header)
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected: %s, got: %v", tc.desc, tc.wantError, err)
			}

			if tc.wantError == "" && err != nil {
				t.Errorf("Test (%s): got unexpected error: %s", tc.desc, resp)
			}

			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}
