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
	"net/http"
	"strings"
	"testing"
	"time"

	bsclient "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

type localServiceHandler struct {
	m *comp.MockServiceCtrl
}

func (h *localServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 100)
	w.Write([]byte(""))
}

func TestServiceControlCheckNetworkFail(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	tests := []struct {
		desc            string
		networkFailOpen bool
		port            uint16
		clientProtocol  string
		httpMethod      string
		method          string
		checkFailStatus int
		wantResp        string
		wantError       string
	}{
		{
			desc:            "Successful, since service_control_network_fail_open is set as true, the timeout of service control check response will be ignored.",
			networkFailOpen: true,
			port:            comp.TestServiceControlCheckNetworkFailOpen,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			wantResp:        `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:            "Failed, since service_control_network_fail_open is set as true default, the timeout of service control check response won't be ignored.",
			networkFailOpen: false,
			port:            comp.TestServiceControlCheckNetworkFailClosed,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			wantError:       "500 Internal Server Error, INTERNAL:Failed to call service control",
		},
	}

	for _, tc := range tests {
		s := env.NewTestEnv(tc.port, "bookstore", nil)
		s.ServiceControlServer.OverrideCheckHandler(&localServiceHandler{
			m: s.ServiceControlServer,
		})
		if tc.networkFailOpen {
			s.EnableScNetworkFailOpen()
		}

		if err := s.Setup(args); err != nil {
			t.Fatalf("fail to setup test env, %v", err)
		}

		s.ServiceControlServer.ResetRequestCount()
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}

		s.TearDown()
	}
}
