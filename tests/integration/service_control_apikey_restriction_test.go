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

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

type testDataStruct struct {
	desc        string
	url         string
	message     string
	forwardedIp string

	wantResp      string
	wantScRequest *utils.ExpectedCheck
}

func TestServiceControlAPIKeyRestriction(t *testing.T) {
	configID := "test-config-id"
	args := []string{
		"--service_config_id=" + configID,
		"--backend_protocol=http1",
		"--rollout_strategy=fixed",
	}

	s := env.NewTestEnv(comp.TestServiceControlAPIKeyRestriction, "echo")
	if err := s.Setup(args); err != nil {
		t.Fatalf("failed to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []testDataStruct{
		{
			desc:     "success, for referrer, ios, android restrictions.",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
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
		{
			desc:        "success, for IP restrictions, the third from right side is the caller ip",
			url:         fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			message:     "hello",
			forwardedIp: "192.16.31.84, 172.17.131.252, 172.17.131.251",
			wantResp:    `{"message":"hello"}`,
			wantScRequest: &utils.ExpectedCheck{
				Version:         utils.APIProxyVersion,
				ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID: "test-config-id",
				ConsumerID:      "api_key:api-key",
				OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				ApiKey:          "api-key",
				CallerIp:        "192.16.31.84",
			},
		},
		{
			desc:        "success, for IP restrictions, the third from right side is the caller ip",
			url:         fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			message:     "hello",
			forwardedIp: "172.17.131.252, 192.16.31.84, 172.17.131.251",
			wantResp:    `{"message":"hello"}`,
			wantScRequest: &utils.ExpectedCheck{
				Version:         utils.APIProxyVersion,
				ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID: "test-config-id",
				ConsumerID:      "api_key:api-key",
				OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				ApiKey:          "api-key",
				CallerIp:        "172.17.131.252",
			},
		},
		{
			desc:        "success, for IP restrictions, the XFF contains fewer than 3 address, falls back to use immediate downstream source address",
			url:         fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			message:     "hello",
			forwardedIp: "192.16.31.84",
			wantResp:    `{"message":"hello"}`,
			wantScRequest: &utils.ExpectedCheck{
				Version:         utils.APIProxyVersion,
				ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID: "test-config-id",
				ConsumerID:      "api_key:api-key",
				OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				ApiKey:          "api-key",
				CallerIp:        "127.0.0.1",
			},
		},
	}

	for _, tc := range testData {
		runTest(t, s, tc)
	}
}

func TestServiceControlAPIKeyIpRestriction(t *testing.T) {
	serviceName := "test-echo"
	configID := "test-config-id"
	args := []string{
		"--service=" + serviceName,
		"--service_config_id=" + configID,
		"--backend_protocol=http1",
		"--rollout_strategy=fixed",
		"--envoy_use_remote_address",
		"--envoy_xff_num_trusted_hops=1",
	}

	s := env.NewTestEnv(comp.TestServiceControlAPIKeyRestriction, "echo")
	if err := s.Setup(args); err != nil {
		t.Fatalf("failed to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []testDataStruct{
		{
			desc:        "success, for IP restrictions, override envoy_use_remote_address and envoy_xff_num_trusted_hops.",
			url:         fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			message:     "hello",
			forwardedIp: "192.16.31.84",
			wantResp:    `{"message":"hello"}`,
			wantScRequest: &utils.ExpectedCheck{
				Version:         utils.APIProxyVersion,
				ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID: "test-config-id",
				ConsumerID:      "api_key:api-key",
				OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				ApiKey:          "api-key",
				CallerIp:        "192.16.31.84",
			},
		},
	}

	for _, tc := range testData {
		runTest(t, s, tc)
	}
}

func runTest(t *testing.T, env *env.TestEnv, tc testDataStruct) {
	wantReq := tc.wantScRequest
	// To set custom headers, use NewRequest and DefaultClient.Do.
	resp, err := client.DoPostWithHeaders(tc.url, tc.message, map[string]string{
		"Referer":                 wantReq.Referer,
		"X-Android-Package":       wantReq.AndroidPackageName,
		"X-Android-Cert":          wantReq.AndroidCertFingerprint,
		"X-Ios-Bundle-Identifier": wantReq.IosBundleID,
		"X-Forwarded-For":         tc.forwardedIp,
	})

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(resp), tc.wantResp) {
		t.Errorf("expected %s, got %s", tc.wantResp, string(resp))
	}

	scRequests, err := env.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("GetRequest returns error: %v", err)
	}

	reqBody := scRequests[0].ReqBody

	if err := utils.VerifyCheck(reqBody, wantReq); err != nil {
		t.Error(err)
	}
}
