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

package service_control_basic_test

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

func TestServiceControlBasic(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(platform.TestServiceControlBasic, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                  string
		url                   string
		method                string
		requestHeader         map[string]string
		message               string
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:     "SC does check and report for a basic GET request.",
			url:      fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/simpleget", "?key=api-key"),
			method:   "GET",
			message:  "",
			wantResp: "simple get message",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/simpleget?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "GET",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "SC does check and report for a basic POST request.",
			url:      fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:   "POST",
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
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
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
			desc:          "SC does NOT check (but does report) when API Key is missing in the request. Operation does NOT allow unregistered callers.",
			url:           fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo"),
			method:        "POST",
			message:       "hello",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 401 Unauthorized"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:            utils.ESPv2Version(),
					ServiceName:        "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:    "test-config-id",
					URL:                "/echo",
					StatusCode:         "16",
					ApiMethod:          "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ApiName:            "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiKeyState:        "NOT CHECKED",
					ErrorCause:         "Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API.",
					ApiVersion:         "1.0.0",
					ProducerProjectID:  "producer-project",
					FrontendProtocol:   "http",
					HttpMethod:         "POST",
					LogMessage:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					ResponseCode:       401,
					Platform:           util.GCE,
					Location:           "test-zone",
					ResponseCodeDetail: "service_control_bad_request{MISSING_API_KEY}",
				},
			},
		},
		{
			desc:     "SC does NOT check (but does report) when API Key is missing in the request. Operation does allow unregistered callers.",
			url:      fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo/nokey"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:        "1.0.0",
					ApiKeyState:       "NOT CHECKED",
					ProducerProjectID: "producer-project",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "SC does NOT check (but does report) when API Key is in the request, but operation does allow unregistered callers.",
			url:      fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo/nokey", "?key=api-key"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/echo/nokey?key=api-key",
					ApiName:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiMethod:       "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					// API Key is not verified by check, but still log it.
					ApiKeyInLogEntryOnly: "api-key",
					ApiKeyState:          "NOT CHECKED",
					ApiVersion:           "1.0.0",
					ProducerProjectID:    "producer-project",
					HttpMethod:           "POST",
					FrontendProtocol:     "http",
					LogMessage:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:           "0",
					ResponseCode:         200,
					Platform:             util.GCE,
					Location:             "test-zone",
				},
			},
		},
		{
			desc:    "Report with referrer header for an operation that does allow unregistered callers.",
			url:     fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo/nokey"),
			message: "hi",
			method:  "POST",
			requestHeader: map[string]string{
				"Referer": "http://google.com/bookstore/root",
			},
			wantResp: `{"message":"hi"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:        "1.0.0",
					ApiKeyState:       "NOT CHECKED",
					ProducerProjectID: "producer-project",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					Referer:           "http://google.com/bookstore/root",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed for unconfigured requests with any path (/**) and POST method, no JWT required, service control sends report request only",
			url:      fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/anypath/x/y/z"),
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/anypath/x/y/z",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiKeyState:       "NOT CHECKED",
					ApiVersion:        "1.0.0",
					ProducerProjectID: "producer-project",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath is called",
					StatusCode:        "0",
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
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
			if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1)
			if err1 == nil || err1.Error() != tc.wantGetScRequestError.Error() {
				t.Errorf("Test (%s): failed", tc.desc)
				t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err1)
				t.Errorf("got service control requests: %v", scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
