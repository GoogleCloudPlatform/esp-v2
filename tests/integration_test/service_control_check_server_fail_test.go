// Copyright 2020 Google LLC
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

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

type serverFailCheckHandler struct {
	m          *comp.MockServiceCtrl
	retryCount int
	respCode   int
	respBody   string
}

func (h *serverFailCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.retryCount++
	http.Error(w, h.respBody, h.respCode)
}

func TestServiceControlCheckServerFailFlag(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	tests := []struct {
		desc            string
		networkFailOpen bool
		method          string
		respCode        int
		respBody        string
		wantRetry       int
		wantResp        string
		wantError       string
	}{
		{
			desc:            "Failed, 403 check error should not retry and fail_open is false.",
			networkFailOpen: false,
			method:          "/v1/shelves/100/books?key=api-key",
			respCode:        403,
			respBody:        "service control service is not enabled",
			wantRetry:       1,
			wantError:       "403 Forbidden, PERMISSION_DENIED:Calling Google Service Control API failed with: 403 and body: service control service is not enabled",
		},
		{
			desc:            "Failed, 403 check error should not retry and fail_open should not apply.",
			networkFailOpen: true,
			method:          "/v1/shelves/100/books?key=api-key",
			respCode:        403,
			respBody:        "service control service is not enabled",
			wantRetry:       1,
			wantError:       "403 Forbidden, PERMISSION_DENIED:Calling Google Service Control API failed with: 403 and body: service control service is not enabled",
		},
		{
			desc:            "Failed, 503 check error should retry and fail_open is false.",
			networkFailOpen: false,
			method:          "/v1/shelves/100/books?key=api-key",
			respCode:        503,
			respBody:        "gateway error",
			wantRetry:       4,
			wantError:       "503 Service Unavailable, UNAVAILABLE:Calling Google Service Control API failed with: 503 and body: gateway error",
		},
		{
			desc:            "Success, 503 check error should retry and fail_open should apply.",
			networkFailOpen: true,
			method:          "/v1/shelves/100/books?key=api-key",
			respCode:        503,
			respBody:        "gateway error",
			wantRetry:       4,
			wantResp:        `{"books":[{"id":"1001","title":"Alphabet"}]}`,
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(comp.TestServiceControlCheckServerFail, platform.GrpcBookstoreSidecar)
			handler := &serverFailCheckHandler{
				m:        s.ServiceControlServer,
				respCode: tc.respCode,
				respBody: tc.respBody,
			}
			s.ServiceControlServer.OverrideCheckHandler(handler)
			if tc.networkFailOpen {
				s.EnableScNetworkFailOpen()
			}

			defer s.TearDown()
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall("http", addr, "GET", tc.method, "", nil)

			if tc.wantRetry != handler.retryCount {
				t.Errorf("Test (%s): failed, expected retry count: %d, got: %d", tc.desc, tc.wantRetry, handler.retryCount)
			}

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}
