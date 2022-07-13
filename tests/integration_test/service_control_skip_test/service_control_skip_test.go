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

package service_control_skip_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlSkipUsage(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(platform.TestServiceControlSkipUsage, platform.EchoSidecar)
	s.AppendUsageRules(
		[]*confpb.UsageRule{
			{
				Selector:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				SkipServiceControl: true,
			},
		},
	)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc               string
		url                string
		method             string
		requestHeader      map[string]string
		message            string
		wantResp           string
		wantScRequestCount int
	}{
		{
			desc:               "succeed, just show the service control works for normal request",
			url:                fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/simplegetcors", "?key=api-key"),
			method:             "GET",
			wantResp:           `simple get message`,
			wantScRequestCount: 2,
		},
		{
			desc:               "succeed, the api with SkipServiceControl set true will skip service control",
			url:                fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:             "POST",
			message:            "hello",
			wantResp:           `{"message":"hello"}`,
			wantScRequestCount: 0,
		},
	}
	for _, tc := range testData {
		s.ServiceControlServer.ResetRequestCount()
		resp, err := client.DoWithHeaders(tc.url, tc.method, tc.message, tc.requestHeader)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}

		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		err = s.ServiceControlServer.VerifyRequestCount(tc.wantScRequestCount)
		if err != nil {
			t.Fatalf("Test (%s): failed, %s", tc.desc, err.Error())
		}
	}
}

func TestHealthCheckSkipServiceControl(t *testing.T) {
	testData := []struct {
		desc                  string
		method                string
		enforceServiceControl bool
		wantResp              string
		wantScRequestCount    int
	}{
		{
			desc:                  "health check should bypass service control by default",
			enforceServiceControl: false,
			wantResp:              "SERVING",
			wantScRequestCount:    0,
		},
		{
			desc:                  "service control on health check can be enforced setting SkipServiceControl to be false",
			enforceServiceControl: true,
			wantResp:              "UNAUTHENTICATED",
			wantScRequestCount:    1,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			configId := "test-config-id"
			args := []string{"--service_config_id=" + configId,
				"--rollout_strategy=fixed", "--suppress_envoy_headers"}
			s := env.NewTestEnv(platform.TestServiceControlSkipUsage, platform.GrpcBookstoreSidecar)
			if tc.enforceServiceControl {
				s.AppendUsageRules(
					[]*confpb.UsageRule{
						{
							Selector:           "grpc.health.v1.Health.Check",
							SkipServiceControl: false,
						},
					},
				)
			}
			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			s.ServiceControlServer.ResetRequestCount()

			url := fmt.Sprintf("http://%v:%v/health/check", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("failed to health check url, %v", err)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body from response: %v", err)
			}

			if !strings.Contains(string(body), tc.wantResp) {
				t.Errorf("failed to obtain expected message, expected: %s, got: %s", tc.wantResp, string(body))
			}
			err = s.ServiceControlServer.VerifyRequestCount(tc.wantScRequestCount)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}
		})
	}
}
