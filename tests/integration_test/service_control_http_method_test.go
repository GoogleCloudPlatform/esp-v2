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
)

func TestServiceControlAllHTTPMethods(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlAllHTTPMethod, platform.EchoSidecar)
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoGET",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/echoMethod",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPOST",
			Pattern: &annotationspb.HttpRule_Post{
				Post: "/echoMethod",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPUT",
			Pattern: &annotationspb.HttpRule_Put{
				Put: "/echoMethod",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPATCH",
			Pattern: &annotationspb.HttpRule_Patch{
				Patch: "/echoMethod",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoDELETE",
			Pattern: &annotationspb.HttpRule_Delete{
				Delete: "/echoMethod",
			},
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
			desc:     "Succeed, test httpMethod GET",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoMethod", "?key=api-key"),
			method:   "GET",
			wantResp: `{"RequestMethod": "GET"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoGET",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echoMethod?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoGET",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "GET",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoGET is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, test httpMethod POST",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoMethod", "?key=api-key"),
			method:   "POST",
			message:  "",
			wantResp: `{"RequestMethod": "POST"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPOST",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echoMethod?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPOST",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "POST",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPOST is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, test httpMethod PUT",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoMethod", "?key=api-key"),
			method:   "PUT",
			message:  "",
			wantResp: `{"RequestMethod": "PUT"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPUT",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echoMethod?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPUT",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "PUT",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPUT is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, test httpMethod PATCH",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoMethod", "?key=api-key"),
			method:   "PATCH",
			message:  "",
			wantResp: `{"RequestMethod": "PATCH"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPATCH",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echoMethod?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPATCH",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "PATCH",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPATCH is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:     "Succeed, test httpMethod DELETE",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoMethod", "?key=api-key"),
			method:   "DELETE",
			wantResp: `{"RequestMethod": "DELETE"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoDELETE",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/echoMethod?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoDELETE",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "DELETE",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoDELETE is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		resp, err := client.DoWithHeaders(tc.url, tc.method, tc.message, nil)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
