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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlBasic(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlBasic, platform.EchoSidecar)
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/simpleget",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/echo/nokey/OverrideAsGet",
			},
		},
	})
	s.AppendUsageRules(
		[]*confpb.UsageRule{
			{
				Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
				AllowUnregisteredCalls: true,
			},
		})

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
			desc:     "succeed GET, no Jwt required, service control sends check request and report request for GET request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/simpleget", "?key=api-key"),
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
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
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
			desc:     "succeed, no Jwt required, service control sends check request and report request for POST request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
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
			desc:          "fail, no Jwt required, service control sends report request only for request that does not have API key but is actually required",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:        "POST",
			message:       "hello",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 401 Unauthorized"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo",
					StatusCode:        "16",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ErrorCause:        "Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API.",
					ApiVersion:        "1.0.0",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					ResponseCode:      401,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, allow no api key (unregistered request), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
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
			desc:     "succeed, no Jwt required, allow no api key (there is API key in request though it is not required), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo/nokey", "?key=api-key"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					ApiKeyInOperationAndLogEntry: "api-key",
					URL:                          "/echo/nokey?key=api-key",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID:            "producer-project",
					HttpMethod:                   "POST",
					FrontendProtocol:             "http",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:    "succeed with request with referer header, no Jwt required, allow no api key (unregistered request), service control sends report (with referer information) only",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
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
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/anypath/x/y/z"),
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
