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
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/golang/protobuf/proto"

	echoClient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestManagedServiceConfig(t *testing.T) {
	t.Parallel()

	args := []string{"--rollout_strategy=managed", "--check_rollout_interval=500ms"}
	s := env.NewTestEnv(comp.TestManagedServiceConfig, platform.GrpcBookstoreSidecar)
	s.SetEnvoyDrainTimeInSec(1)

	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc           string
		clientProtocol string
		httpMethod     string
		method         string
		token          string
		headers        map[string][]string
		wantResp       string
		wantError      string
	}{
		{
			desc:           "Fail, since the request doesn't have token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
		},
		{
			desc:           "Success, the new service config doesn't require JWT for this API",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
	}

	for idx, tc := range tests {
		// Remove the authentication in service config and wait envoy to update.
		if idx == 1 {
			s.OverrideAuthentication(&confpb.Authentication{})
			s.OverrideRolloutIdAndConfigId("new-service-rollout-id", "new-service-config-id")
			time.Sleep(time.Second * 3)
		}

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			}
		}
	}
}

type configsHandler struct {
	m                  *comp.MockServiceMrg
	rejectWith429Times int
	curFailCnt         int
}

func (h *configsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.curFailCnt < h.rejectWith429Times {
		h.curFailCnt += 1
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	serviceConfigByte, _ := proto.Marshal(h.m.ServiceConfig)
	h.m.LastServiceConfig = serviceConfigByte
	_, _ = w.Write(serviceConfigByte)
}

func TestRetryCallServiceManagement(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"

	testCases := []struct {
		desc     string
		retryNum int
	}{
		{
			desc:     "fail, retry 2 times for servicemanagement server rejects 2 times",
			retryNum: 2,
		},
		{
			desc:     "success, retry 3 times while servicemanagement server rejects 2 times",
			retryNum: 3,
		},
	}
	for _, tc := range testCases {

		_test := func() {
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed", "--healthz=/healthz", fmt.Sprintf(`--service_management_call_retry_configs={"429":{"RetryNum":%v,"RetryInterval":100000000,}}`, tc.retryNum)}

			s := env.NewTestEnv(comp.TestRetryCallServiceManagement, platform.EchoSidecar)
			defer s.TearDown(t)

			s.MockServiceManagementServer.ConfigsHandler = &configsHandler{
				m:                  s.MockServiceManagementServer,
				rejectWith429Times: 2,
			}

			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v/echo", s.Ports().ListenerPort)

			resp, err := echoClient.DoPost(fmt.Sprintf("%s?key=api-key", url), echo)
			if err != nil {
				t.Errorf("got unexpected error: %v", err)
			}
			wantResp := `{"message":"hello"}`
			if string(resp) != wantResp {
				t.Errorf("expected resp: %s, got response: %s", wantResp, string(resp))
			}
		}

		_test()
	}
}
