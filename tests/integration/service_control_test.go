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
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
)

func TestServiceControlBasic(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--skip_jwt_authn_filter", "--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestServiceControlBasic, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := []struct {
		desc     string
		method   string
		wantResp string
	}{
		{
			desc:     "succeed, no Jwt required",
			wantResp: `{"message":"hello"}`,
		},
	}
	for _, tc := range testData {
		host := fmt.Sprintf("http://localhost:%v", s.Ports.ListenerPort)
		resp, err := client.DoEcho(host, "api-key", "hello")
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
		}

		sc_requests, err1 := s.ServiceControlServer.GetRequests(2, 3*time.Second)
		if err1 != nil {
			t.Errorf("GetRequests returns error: %v", err1)
		}
		if len(sc_requests) != 2 {
			t.Errorf("Expected number of requests is 2 ,but got: %d", len(sc_requests))
		}
		if sc_requests[0].ReqType != comp.CHECK_REQUEST {
			t.Errorf("service control request 0: should be Check")
		}
		if sc_requests[1].ReqType != comp.REPORT_REQUEST {
			t.Errorf("service control request 1: should be Report")
		}
		if !utils.VerifyCheck(sc_requests[0].ReqBody, &utils.ExpectedCheck{
			Version:         "0.1",
			ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:api-key",
			OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
		}) {
			t.Errorf("Check request data doesn't match.")
		}
		if !utils.VerifyReport(sc_requests[1].ReqBody, &utils.ExpectedReport{
			Version:           "0.1",
			ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:   "test-config-id",
			URL:               "/echo?key=api-key",
			ApiKey:            "api-key",
			ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			ProducerProjectID: "producer-project",
			HttpMethod:        "POST",
			LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
			RequestSize:       20,
			ResponseSize:      19,
			ResponseCode:      200,
		}) {
			t.Errorf("Report request data doesn't match.")
		}
	}

}
