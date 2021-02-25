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

package http_method_override_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestMethodOverrideBackendMethod(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestMethodOverrideBackendMethod, platform.EchoSidecar)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		path          string
		method        string
		headers       map[string]string
		wantResp      string
		httpCallError error
	}{
		{
			desc:   "Overridden POST is received as GET",
			path:   "/echoMethod?key=api-key",
			method: "POST",
			headers: map[string]string{
				"X-HTTP-Method-Override": "GET",
			},
			wantResp: `{"RequestMethod":"GET"}`,
		},
		{
			desc:   "Overridden DELETE is received as POST",
			path:   "/echoMethod?key=api-key",
			method: "DELETE",
			headers: map[string]string{
				"X-HTTP-Method-Override": "POST",
			},
			wantResp: `{"RequestMethod":"POST"}`,
		},
		{
			desc:   "Overridden POST is rejected because DELETE is not defined in the service config",
			path:   "/echoMethod?key=api-key",
			method: "POST",
			headers: map[string]string{
				"X-HTTP-Method-Override": "DELETE",
			},
			httpCallError: fmt.Errorf("{\"code\":405,\"message\":\"The current request is matched to the defined url template \"/echoMethod\" but its http method is not allowed\"}"),
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, tc.path)
		gotResp, err := client.DoWithHeaders(url, tc.method, "test-body", tc.headers)

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			if err == nil || !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("expected Http call error: %v, got: %v", tc.httpCallError, err)
			}
			continue
		}
		gotRespStr := string(gotResp)
		if err := util.JsonEqual(tc.wantResp, gotRespStr); err != nil {
			t.Errorf("Test(%s) fails: \n %s", tc.desc, err)
		}
	}
}

func TestMethodOverrideBackendBody(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestMethodOverrideBackendBody, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		path          string
		method        string
		headers       map[string]string
		body          string
		wantResp      string
		httpCallError error
	}{
		{
			desc:   "Overridden PUT has body sent from client",
			path:   "/echo?key=api-key",
			method: "PUT",
			headers: map[string]string{
				"X-HTTP-Method-Override": "POST",
			},
			body:     `hello`,
			wantResp: `{"message":"hello"}`,
		},
		{
			desc:   "Overridden GET has no body for the backend to handle, as client did not send it",
			path:   "/echo?key=api-key",
			method: "GET",
			headers: map[string]string{
				"X-HTTP-Method-Override": "POST",
			},
			body:          "This body will not be sent in the request",
			httpCallError: fmt.Errorf(`{"code":500,"message":"Could not get body: EOF"}`),
		},
		{
			desc:   "Overridden POST has body sent from client",
			path:   "/echo?key=api-key",
			method: "POST",
			headers: map[string]string{
				"X-HTTP-Method-Override": "GET",
			},
			body:     "hello",
			wantResp: `{"message":"hello"}`,
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, tc.path)
		gotResp, err := client.DoWithHeaders(url, tc.method, tc.body, tc.headers)

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			if err == nil || !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("expected Http call error: %v, got: %v", tc.httpCallError, err)
			}
			continue
		}
		gotRespStr := string(gotResp)
		if err := util.JsonEqual(tc.wantResp, gotRespStr); err != nil {
			t.Errorf("Test(%s) fails: \n %s", tc.desc, err)
		}
	}
}

func TestMethodOverrideScReport(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestMethodOverrideScReport, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		url            string
		method         string
		requestHeader  map[string]string
		message        string
		wantResp       string
		httpCallError  error
		wantScRequests []interface{}
	}{
		{
			desc:    "Overridden POST is displayed as GET in SC Report, success case",
			url:     fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo?key=api-key"),
			message: "hello",
			method:  "POST",
			requestHeader: map[string]string{
				"X-HTTP-Method-Override": "GET",
			},
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoGetWithBody",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echo?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoGetWithBody",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					HttpMethod:                   "GET",
					FrontendProtocol:             "http",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoGetWithBody is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		resp, err := client.DoWithHeaders(tc.url, tc.method, tc.message, tc.requestHeader)
		if tc.httpCallError == nil {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if err == nil || !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
