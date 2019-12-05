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

package dynamic_routing_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

var testDynamicRoutingArgs = []string{
	"--service_config_id=test-config-id",
	"--backend_protocol=http1",
	"--rollout_strategy=fixed",
	"--enable_backend_routing",
	"--backend_dns_lookup_family=v4only",
	"--suppress_envoy_headers",
}

func NewDynamicRoutingTestEnv(port uint16) *env.TestEnv {
	s := env.NewTestEnv(port, "echoForDynamicRouting")
	s.EnableDynamicRoutingBackend()
	return s
}

func TestDynamicRouting(t *testing.T) {

	s := NewDynamicRoutingTestEnv(comp.TestDynamicRouting)

	defer s.TearDown()

	if err := s.Setup(testDynamicRoutingArgs); err != nil {
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
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		gotResp, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("expected Http call error: %v, got: %v", tc.httpCallError, err)
			}
			continue
		}
		gotRespStr := string(gotResp)
		if !utils.JsonEqual(gotRespStr, tc.wantResp) {
			t.Errorf("response want: %s, got: %s", tc.wantResp, gotRespStr)
		}
	}
}

func TestDynamicRoutingWithAllowCors(t *testing.T) {
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsOrigin := "http://cloud.google.com"

	respHeaderMap := make(map[string]string)
	respHeaderMap["Access-Control-Allow-Origin"] = "*"
	respHeaderMap["Access-Control-Allow-Methods"] = "GET, OPTIONS"
	respHeaderMap["Access-Control-Allow-Headers"] = "Authorization"
	respHeaderMap["Access-Control-Expose-Headers"] = "Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	respHeaderMap["Access-Control-Allow-Credentials"] = "true"

	s := NewDynamicRoutingTestEnv(comp.TestDynamicRoutingWithAllowCors)
	s.SetAllowCors()

	defer s.TearDown()

	if err := s.Setup(testDynamicRoutingArgs); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc string
		url  string
	}{
		{
			// when allowCors, passes preflight CORS request to backend.
			desc: "Succeed, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simplegetcors"),
		},
		{
			// when allowCors, passes preflight CORS request without valid jwt token to backend,
			// even the origin method requires authentication.
			desc: "Succeed without jwt token, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/auth/info/firebase"),
		},
	}

	for _, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader, "")
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

	s := NewDynamicRoutingTestEnv(comp.TestServiceControlRequestForDynamicRouting)

	defer s.TearDown()

	if err := s.Setup(testDynamicRoutingArgs); err != nil {
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
					Version:         utils.ESPv2Version,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
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
					Version:         utils.ESPv2Version,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/sc/searchpet?key=api-key&timezone=EST",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
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
					Version:         utils.ESPv2Version,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/sc/pet/0325/num/2019?key=api-key&lang=en",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
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
			t.Fatal(err)
		}

		gotRespStr := string(gotResp)

		if !utils.JsonEqual(gotRespStr, tc.wantResp) {
			t.Errorf("Test Desc(%s): response want: %s, got: %s", tc.desc, tc.wantResp, gotRespStr)
		}

		scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err != nil {
			t.Fatalf("Test Desc(%s): GetRequests returns error: %v", tc.desc, err)
		}

		for i, wantScRequest := range tc.wantScRequests {
			reqBody := scRequests[i].ReqBody
			switch wantScRequest.(type) {
			case *utils.ExpectedCheck:
				if scRequests[i].ReqType != comp.CHECK_REQUEST {
					t.Errorf("Test Desc(%s): service control request %v: should be Check", tc.desc, i)
				}
				if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
					t.Error(err)
				}
			case *utils.ExpectedReport:
				if scRequests[i].ReqType != comp.REPORT_REQUEST {
					t.Errorf("Test Desc(%s): service control request %v: should be Report", tc.desc, i)
				}
				if err := utils.VerifyReport(reqBody, wantScRequest.(*utils.ExpectedReport)); err != nil {
					t.Error(err)
				}
			default:
				t.Fatalf("Test Desc(%s): unknown service control response type", tc.desc)
			}
		}
	}
}
