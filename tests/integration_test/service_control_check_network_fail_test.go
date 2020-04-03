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

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestServiceControlCheckNetworkFail(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"

	testdata := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		allocatedPort      uint16
		method             string
		serviceControlURL  string
		wantResp           string
		wantError          string
		wantScRequestCount int
	}{
		{
			desc:              "Failed. When the service control url is wrong, the request will be rejected by 500 Internal Server Error",
			clientProtocol:    "http",
			httpMethod:        "GET",
			method:            "/v1/shelves/100?key=api-key-1",
			serviceControlURL: "http://unavaliable_service_control_server_name",
			allocatedPort:     comp.TestServiceControlCheckWrongServerName,
			wantError:         "503 Service Unavailable, UNAVAILABLE:Calling ServiceControl failed with: 503",
		},
		{
			desc:              "Failed. When the service control is not set up, the request will be rejected by 500 Internal Server Error",
			clientProtocol:    "http",
			httpMethod:        "GET",
			method:            "/v1/shelves/100?key=api-key-2",
			serviceControlURL: "http://localhost:28753",
			allocatedPort:     comp.TestServiceControlCheckWrongServerName,
			wantError:         "503 Service Unavailable, UNAVAILABLE:Calling ServiceControl failed with: 503",
		},
	}

	for _, tc := range testdata {
		func() {
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed"}
			s := env.NewTestEnv(tc.allocatedPort, platform.GrpcBookstoreSidecar)
			s.ServiceControlServer.SetURL(tc.serviceControlURL)

			defer s.TearDown()
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
		}()
	}
}

type checkTimeoutServiceHandler struct {
	m *comp.MockServiceCtrl
}

func (h *checkTimeoutServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.Write([]byte(""))
}

func TestServiceControlCheckTimeout(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestServiceControlCheckTimeout, platform.GrpcBookstoreSidecar)
	s.ServiceControlServer.SetURL("http://wrong_service_control_server_name")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	type test struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		wantResp           string
		wantError          string
		wantScRequestCount int
	}
	tc := test{
		desc:           "Failed. When the check request is timeout, the request will be rejected by 500 Internal Server Error",
		clientProtocol: "http",
		httpMethod:     "GET",
		method:         "/v1/shelves/100?key=api-key-2",
		wantError:      "503 Service Unavailable, UNAVAILABLE:Calling ServiceControl failed with: 503",
	}

	s.ServiceControlServer.ResetRequestCount()
	addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
	resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

	if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
		t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
	} else if !strings.Contains(resp, tc.wantResp) {
		t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
	}
}

type localServiceHandler struct {
	m *comp.MockServiceCtrl
}

func (h *localServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 100)
	w.Write([]byte(""))
}

func TestServiceControlNetworkFailFlag(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	tests := []struct {
		desc            string
		networkFailOpen bool
		port            uint16
		clientProtocol  string
		httpMethod      string
		method          string
		token           string
		checkFailStatus int
		wantResp        string
		wantError       string
	}{
		{
			desc:            "Successful, since service_control_network_fail_open is set as true, the timeout of service control check response will be ignored.",
			networkFailOpen: true,
			port:            comp.TestServiceControlNetworkFailFlagOpen,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			token:           testdata.FakeCloudTokenMultiAudiences,
			wantResp:        `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:            "Failed, since service_control_network_fail_open is set as true default, the timeout of service control check response won't be ignored.",
			networkFailOpen: false,
			port:            comp.TestServiceControlNetworkFailFlagClosed,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			token:           testdata.FakeCloudTokenMultiAudiences,
			wantError:       "503 Service Unavailable, UNAVAILABLE:Calling ServiceControl failed with: 504",
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(tc.port, platform.GrpcBookstoreSidecar)
			s.ServiceControlServer.OverrideCheckHandler(&localServiceHandler{
				m: s.ServiceControlServer,
			})
			if tc.networkFailOpen {
				s.EnableScNetworkFailOpen()
			}

			defer s.TearDown()
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			s.ServiceControlServer.ResetRequestCount()
			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}
