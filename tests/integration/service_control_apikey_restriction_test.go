// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestServiceControlApiKeyRestriction(t *testing.T) {
	serviceName := "test-echo"
	configID := "test-config-id"

	args := []string{
		"--service=" + serviceName,
		"--version=" + configID,
		"--backend_protocol=http1",
		"--rollout_strategy=fixed",
	}

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceControl:    true,
		MockServiceManagement: true,
		MockJwtProviders:      []string{"google_jwt"},
	}

	if err := s.Setup(comp.TestServiceControlAPIKeyRestriction, "echo", args); err != nil {
		t.Fatalf("failed to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := []struct {
		desc    string
		url     string
		message string

		wantResp      string
		wantScRequest *utils.ExpectedCheck
	}{
		{
			desc:     "success, with android headers",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports.ListenerPort, "/echo", "?key=api-key"),
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequest: &utils.ExpectedCheck{
				Version:                utils.APIProxyVersion,
				ServiceName:            "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID:        "test-config-id",
				ConsumerID:             "api_key:api-key",
				OperationName:          "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				ApiKey:                 "api-key",
				AndroidCertFingerprint: "ABCDESF",
				AndroidPackageName:     "com.google.cloud",
				IosBundleID:            "5b40ad6af9a806305a0a56d7cb91b82a27c26909",
				Referer:                "referer",
				CallerIp:               "127.0.0.1",
			},
		},
	}

	for _, tc := range testData {
		wantReq := tc.wantScRequest

		// To set custom headers, use NewRequest and DefaultClient.Do.
		resp, err := client.DoPostWithHeaders(tc.url, tc.message, map[string]string{
			"Referer":                 wantReq.Referer,
			"X-Android-Package":       wantReq.AndroidPackageName,
			"X-Android-Cert":          wantReq.AndroidCertFingerprint,
			"X-Ios-Bundle-Identifier": wantReq.IosBundleID,
		})

		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected %s, got %s", tc.wantResp, string(resp))
		}

		scRequests, err := s.ServiceControlServer.GetRequests(1, 3*time.Second)
		if err != nil {
			t.Fatalf("GetRequest returns error: %v", err)
		}

		reqBody := scRequests[0].ReqBody

		if err := utils.VerifyCheck(reqBody, wantReq); err != nil {
			t.Error(err)
		}
	}
}
