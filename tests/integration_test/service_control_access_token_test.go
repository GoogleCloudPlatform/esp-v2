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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestServiceControlAccessTokenFromIam(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlAccessTokenFromIam, platform.EchoSidecar)
	serviceAccount := "ServiceAccount@google.com"
	s.SetServiceControlIamServiceAccount(serviceAccount)
	s.SetServiceControlIamDelegates("delegate_foo,delegate_bar,delegate_baz")

	s.SetIamResps(
		map[string]string{
			fmt.Sprintf("/v1/projects/-/serviceAccounts/%s:generateAccessToken", serviceAccount): `{"accessToken":  "access-token-from-iam", "expireTime": "2022-10-02T15:01:23.045123456Z"}`,
		}, 0, 0)

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	testData := []struct {
		desc    string
		url     string
		method  string
		message string

		wantIamReqToken          string
		wantIamReqBody           string
		wantScRequestAccessToken string
	}{
		{
			desc:                     "succeed, fetching access token from IAM using access token got from IMDS",
			url:                      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:                   "POST",
			message:                  "this-is-messgae",
			wantIamReqToken:          "Bearer ya29.new",
			wantIamReqBody:           `{"scope":["https://www.googleapis.com/auth/servicecontrol"],"delegates":["projects/-/serviceAccounts/delegate_foo","projects/-/serviceAccounts/delegate_bar","projects/-/serviceAccounts/delegate_baz"]}`,
			wantScRequestAccessToken: "Bearer access-token-from-iam",
		},
	}
	for _, tc := range testData {
		_, err := client.DoWithHeaders(tc.url, tc.method, tc.message, nil)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}

		// The check call and the report call will be sent.
		scRequests, err1 := s.ServiceControlServer.GetRequests(2)
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}

		for _, scRequest := range scRequests {
			if gotAccessToken := scRequest.ReqHeader.Get("Authorization"); gotAccessToken != tc.wantScRequestAccessToken {
				t.Errorf("Test (%s): failed, different access token received by service controller, expected: %v, but got: %v", tc.desc, tc.wantScRequestAccessToken, gotAccessToken)
			}
		}

		if iamReqToken, err := s.MockIamServer.GetRequestToken(); err != nil {
			t.Errorf("Test Desc(%s): failed to get request header", tc.desc)
		} else if tc.wantIamReqToken != iamReqToken {
			t.Errorf("Test Desc(%s), different iam request token, wanted: %s, got: %s", tc.desc, tc.wantIamReqToken, iamReqToken)
		}

		if iamReqBody, err := s.MockIamServer.GetRequestBody(); err != nil {
			t.Errorf("Test Desc(%s): failed to get request body", tc.desc)
		} else if tc.wantIamReqBody != "" && tc.wantIamReqBody != iamReqBody {
			t.Errorf("Test Desc(%s), different request body received by iam, expected: %s, got: %s", tc.desc, tc.wantIamReqBody, iamReqBody)
		}
	}
}

func TestServiceControlAccessTokenFromTokenAgent(t *testing.T) {
	t.Parallel()

	customSa, err := utils.NewServiceAccountForTest()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	defer customSa.MockTokenServer.Close()

	args := []string{"--service_config_id=test-config-id",
		"--rollout_strategy=fixed", "--suppress_envoy_headers", "--service_account_key=" + customSa.FileName}

	s := env.NewTestEnv(comp.TestServiceControlAccessTokenFromTokenAgent, platform.EchoSidecar)

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	testData := []struct {
		desc                     string
		url                      string
		method                   string
		wantScRequestAccessToken string
	}{
		{
			desc:                     "succeed, fetching access token from local server",
			url:                      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:                   "POST",
			wantScRequestAccessToken: "Bearer this-is-sa_gen_token",
		},
	}
	for _, tc := range testData {
		_, err := client.DoWithHeaders(tc.url, tc.method, "message", nil)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}

		// The check call and the report call will be sent.
		scRequests, err := s.ServiceControlServer.GetRequests(2)
		if err != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err)
		}

		for _, scRequest := range scRequests {
			if gotAccessToken := scRequest.ReqHeader.Get("Authorization"); gotAccessToken != tc.wantScRequestAccessToken {
				t.Errorf("Test (%s): failed, different access token received by service controller, expected: %v, but got: %v", tc.desc, tc.wantScRequestAccessToken, gotAccessToken)
			}
		}
	}
}
