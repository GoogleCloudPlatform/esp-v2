// Copyright 2018 Google Cloud Platform Proxy Authors
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

package integration

import (
	"fmt"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
	"google.golang.org/genproto/googleapis/api/annotations"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlReportResponseCode(t *testing.T) {
	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestServiceControlReportResponseCode, "echo", nil)
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified",
			Pattern: &annotations.HttpRule_Get{
				Get: "/simpleget/304",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetUnauthorized",
			Pattern: &annotations.HttpRule_Get{
				Get: "/simpleget/401",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden",
			Pattern: &annotations.HttpRule_Get{
				Get: "/simpleget/403",
			},
		},
	})
	s.AppendUsageRules(
		[]*conf.UsageRule{
			{
				Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden",
				AllowUnregisteredCalls: true,
			},
		})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc                  string
		url                   string
		requestHeader         map[string]string
		message               string
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		// TODO(jcwang): add test cases for 304 and 403 to validate status in Check request
		{
			desc:          "succeed which has 304 response, no Jwt required, service control sends report request only with status code 304.",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simpleget/304"),
			httpCallError: fmt.Errorf("http response status is not 200 OK: 304 Not Modified"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/simpleget/304",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified is called",
					StatusCode:        "0",
					RequestSize:       170,
					ResponseSize:      84,
					RequestBytes:      170,
					ResponseBytes:     84,
					ResponseCode:      304,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:          "succeed which has 403 response, no Jwt required, service control sends report request only with status code 403.",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simpleget/403"),
			httpCallError: fmt.Errorf("http response status is not 200 OK: 403 Forbidden"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/simpleget/403",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden",
					ProducerProjectID: "producer-project",
					ErrorType:         "4xx",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden is called",
					StatusCode:        "0",
					RequestSize:       170,
					ResponseSize:      99,
					RequestBytes:      170,
					ResponseBytes:     99,
					ResponseCode:      403,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:          "succeed, service control still sends report when the request is rejected by the backend with 401 so the status code is 16",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simpleget/401"),
			httpCallError: fmt.Errorf("http response status is not 200 OK: 401 Unauthorized"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/simpleget/401",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetUnauthorized",
					ProducerProjectID: "producer-project",
					ErrorType:         "4xx",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetUnauthorized is called",
					StatusCode:        "16",
					RequestSize:       170,
					ResponseSize:      266,
					RequestBytes:      170,
					ResponseBytes:     266,
					ResponseCode:      401,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		resp, err := client.DoGet(tc.url)
		if tc.httpCallError == nil {
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test desc (%v) expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("Test desc (%v) expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1)
			if err1 == nil || strings.Contains(err1.Error(), tc.wantGetScRequestError.Error()) {
				t.Errorf("Test desc (%v) expected get service control request call error: %v, got: %v", tc.desc, tc.wantGetScRequestError, err1)
				t.Errorf("Test desc (%v) got service control requests: %v", tc.desc, scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test desc (%v) GetRequests returns error: %v", tc.desc, err1)
		}

		for i, wantScRequest := range tc.wantScRequests {
			reqBody := scRequests[i].ReqBody
			switch wantScRequest.(type) {
			case *utils.ExpectedCheck:
				if scRequests[i].ReqType != comp.CHECK_REQUEST {
					t.Errorf("Test desc (%v) service control request %v: should be Check", tc.desc, i)
				}
				if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
					t.Error(err)
				}
			case *utils.ExpectedReport:
				if scRequests[i].ReqType != comp.REPORT_REQUEST {
					t.Errorf("Test desc (%v) service control request %v: should be Report", tc.desc, i)
				}
				if err := utils.VerifyReport(reqBody, wantScRequest.(*utils.ExpectedReport)); err != nil {
					t.Error(err)
				}
			default:
				t.Fatalf("Test desc (%v) unknown service control response type", tc.desc)
			}
		}
	}
}
