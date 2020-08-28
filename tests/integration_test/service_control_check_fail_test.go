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
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func TestServiceControlCheckError(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	provider := testdata.GoogleJwtProvider

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlCheckError, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	type expectedRequestCount struct {
		key string
		cnt int
	}
	testData := []struct {
		desc                     string
		url                      string
		path                     string
		method                   string
		token                    string
		apiKey                   string
		requestHeader            map[string]string
		message                  string
		mockedCheckResponse      *scpb.CheckResponse
		wantRequestsToMetaServer *expectedRequestCount
		wantRequestsToProvider   *expectedRequestCount
		wantResp                 string
		wantError                string
		wantScRequests           []interface{}
	}{
		{
			desc:    "Failed, the check return SERVICE_NOT_ACTIVATED and no consumer project id",
			url:     fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key-1"),
			method:  "POST",
			message: "",
			mockedCheckResponse: &scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_SERVICE_NOT_ACTIVATED,
					},
				},
			},
			wantError: `403 Forbidden, {"code":403,"message":"PERMISSION_DENIED:API echo-api.endpoints.cloudesf-testing.cloud.goog is not enabled for the project."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-1",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo?key=api-key-1",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ApiVersion:        "1.0.0",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ErrorCause:        "API echo-api.endpoints.cloudesf-testing.cloud.goog is not enabled for the project.",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:        "7",
					ResponseCode:      403,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:    "Failed, the check return API_KEY_INVALID and no consumer project id",
			url:     fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key-2"),
			method:  "POST",
			message: "",
			mockedCheckResponse: &scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_API_KEY_INVALID,
					},
				},
			},
			wantError: `400 Bad Request, {"code":400,"message":"INVALID_ARGUMENT:API key not valid. Please pass a valid API key."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-2",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo?key=api-key-2",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ErrorCause:        "API key not valid. Please pass a valid API key.",
					ApiVersion:        "1.0.0",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:        "3",
					ResponseCode:      400,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:   "Failed, the request passed auth but failed in check with PROJECT_INVALID",
			path:   "/auth/info/auth0",
			apiKey: "api-key",
			method: "GET",
			token:  testdata.FakeCloudTokenMultiAudiences,
			mockedCheckResponse: &scpb.CheckResponse{
				CheckInfo: &scpb.CheckResponse_CheckInfo{
					ConsumerInfo: &scpb.CheckResponse_ConsumerInfo{
						ProjectNumber: 123456,
					},
				},
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_PROJECT_INVALID,
					},
				},
			},
			// Note: first request is from Config Manager, second is from ESPv2
			wantRequestsToMetaServer: &expectedRequestCount{util.AccessTokenPath, 2},
			wantRequestsToProvider:   &expectedRequestCount{provider, 1},
			wantError:                `400 Bad Request, {"code":400,"message":"INVALID_ARGUMENT:Client project not valid. Please pass a valid project."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:     utils.ESPv2Version(),
					ServiceName: "echo-api.endpoints.cloudesf-testing.cloud.goog", ServiceConfigID: "test-config-id",
					URL:                          "/auth/info/auth0?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					ErrorCause:                   "Client project not valid. Please pass a valid project.",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "GET",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0 is called",
					StatusCode:                   "3",
					ResponseCode:                 400,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		if tc.mockedCheckResponse != nil {
			s.ServiceControlServer.SetCheckResponse(tc.mockedCheckResponse)
		}
		var resp []byte
		var err error
		if tc.token != "" {
			resp, err = client.DoJWT(fmt.Sprintf("http://localhost:%v", s.Ports().ListenerPort), tc.method, tc.path, tc.apiKey, "", tc.token)
		} else {
			resp, err = client.DoWithHeaders(tc.url, tc.method, tc.message, nil)
		}

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed\nexpected: %v\ngot: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed\nexpected: %s\ngot: %s", tc.desc, tc.wantResp, string(resp))
		}

		if tc.wantRequestsToMetaServer != nil {
			if realCnt := s.MockMetadataServer.GetReqCnt(tc.wantRequestsToMetaServer.key); realCnt != tc.wantRequestsToMetaServer.cnt {
				t.Errorf("Test (%s): failed, %s on MetadataServer should be requested %v times not %v times.", tc.desc, tc.wantRequestsToProvider.key, tc.wantRequestsToMetaServer.cnt, realCnt)
			}
		}

		if tc.wantRequestsToProvider != nil {
			provider, ok := s.FakeJwtService.ProviderMap[tc.wantRequestsToProvider.key]
			if !ok {
				t.Errorf("Test (%s): failed, the provider is not inited.", tc.desc)
			} else if realCnt := provider.GetReqCnt(); realCnt != tc.wantRequestsToProvider.cnt {
				t.Errorf("Test (%s): failed, pubkey of %s shoud be fetched %v times instead of %v times.", tc.desc, tc.wantRequestsToProvider.key, tc.wantRequestsToProvider.cnt, realCnt)
			}
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
