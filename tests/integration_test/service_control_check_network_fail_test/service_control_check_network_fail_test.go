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

package service_control_check_network_fail_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
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
			allocatedPort:     platform.TestServiceControlCheckWrongServerName,
			wantError:         `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:Calling Google Service Control API failed with: 503 and body: no healthy upstream"}`,
		},
		{
			desc:              "Failed. When the service control is not set up, the request will be rejected by 500 Internal Server Error",
			clientProtocol:    "http",
			httpMethod:        "GET",
			method:            "/v1/shelves/100?key=api-key-2",
			serviceControlURL: fmt.Sprintf("http://%v:28753", platform.GetLoopbackAddress()),
			allocatedPort:     platform.TestServiceControlCheckWrongServerName,
			wantError:         `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:Calling Google Service Control API failed with: 503 and body: upstream connect error or disconnect/reset before headers. reset reason: connection failure, transport failure reason: delayed connect error: 111"}`,
		},
	}

	for _, tc := range testdata {
		func() {
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed"}
			s := env.NewTestEnv(tc.allocatedPort, platform.GrpcBookstoreSidecar)
			s.ServiceControlServer.SetURL(tc.serviceControlURL)

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			s.ServiceControlServer.ResetRequestCount()
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}

func TestServiceControlCheckTimeout(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestServiceControlCheckTimeout, platform.GrpcBookstoreSidecar)
	s.ServiceControlServer.SetURL("http://wrong_service_control_server_name")
	defer s.TearDown(t)
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
		wantError:      `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:Calling Google Service Control API failed with: 503 and body: no healthy upstream"}`,
	}

	s.ServiceControlServer.ResetRequestCount()
	addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
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
	_, _ = w.Write([]byte(""))
}

func TestServiceControlNetworkFailFlagForTimeout(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	tests := []struct {
		desc            string
		networkFailOpen bool
		clientProtocol  string
		httpMethod      string
		method          string
		token           string
		checkFailStatus int
		wantResp        string
		wantError       string
		wantScRequests  []interface{}
	}{
		{
			desc:            "Successful, since service_control_network_fail_open is set as true, the timeout of service control check response will be ignored.",
			networkFailOpen: true,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			token:           testdata.FakeCloudTokenLongClaims,
			wantResp:        `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`, wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/v1/shelves?key=api-key",
					// API Key is not trusted due to SC network failure.
					ApiKeyInLogEntryOnly: "api-key",
					ApiKeyState:          "NOT CHECKED",
					// API Key is not trusted, so JWT is used as credential_id instead.
					JwtAuthCredentialId: "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw",
					ApiMethod:           "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:             "endpoints.examples.bookstore.Bookstore",
					ApiVersion:          "1.0.0",
					ProducerProjectID:   "producer project",
					FrontendProtocol:    "http",
					BackendProtocol:     "grpc",
					HttpMethod:          "GET",
					LogMessage:          "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:          "0",
					ResponseCode:        200,
					Platform:            util.GCE,
					Location:            "test-zone",
				},
			},
		},
		{
			desc:            "Failed, since service_control_network_fail_open is set as false, the timeout of service control check response won't be ignored.",
			networkFailOpen: false,
			clientProtocol:  "http",
			httpMethod:      "GET",
			method:          "/v1/shelves?key=api-key",
			token:           testdata.FakeCloudTokenLongClaims,
			wantError:       `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:Calling Google Service Control API failed with: 504 and body: upstream request timeout"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/v1/shelves?key=api-key",
					// API Key is not trusted due to SC network failure.
					ApiKeyInLogEntryOnly: "api-key",
					ApiKeyState:          "NOT CHECKED",
					// API Key is not trusted, so JWT is used as credential_id instead.
					JwtAuthCredentialId: "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw",
					ApiMethod:           "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:             "endpoints.examples.bookstore.Bookstore",
					ApiVersion:          "1.0.0",
					ErrorCause:          "Calling Google Service Control API failed with: 504 and body: upstream request timeout",
					ProducerProjectID:   "producer project",
					FrontendProtocol:    "http",
					BackendProtocol:     "grpc",
					HttpMethod:          "GET",
					LogMessage:          "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:          "14",
					ResponseCode:        503,
					Platform:            util.GCE,
					Location:            "test-zone",
					ResponseCodeDetail:  "service_control_check_network_failure{UNAVAILABLE}",
				},
			},
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(platform.TestServiceControlNetworkFailFlagForTimeout, platform.GrpcBookstoreSidecar)
			s.ServiceControlServer.OverrideCheckHandler(&localServiceHandler{
				m: s.ServiceControlServer,
			})
			if tc.networkFailOpen {
				s.EnableScNetworkFailOpen()
			}

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			s.ServiceControlServer.ResetRequestCount()
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err)
			}
			utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
		}()
	}
}

func TestServiceControlNetworkFailFlagForUnavailableCheckResponse(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	tests := []struct {
		desc            string
		networkFailOpen bool
		checkResponse   scpb.CheckResponse
		clientProtocol  string
		httpMethod      string
		method          string
		token           string
		checkFailStatus int
		wantResp        string
		wantError       string
	}{
		{
			desc:            "Successful, since service_control_network_fail_open is set as true, the unavailable check error will be ignored.",
			networkFailOpen: true,
			checkResponse: scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_NAMESPACE_LOOKUP_UNAVAILABLE,
					},
				},
			},
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenLongClaims,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:            "Failed, since service_control_network_fail_open is set as false, the unavailable check error won't be ignored.",
			networkFailOpen: false,
			checkResponse: scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_NAMESPACE_LOOKUP_UNAVAILABLE,
					},
				},
			},
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenLongClaims,
			wantError:      `503 Service Unavailable, {"code":503,"message":"UNAVAILABLE:One or more Google Service Control backends are unavailable."}`,
		},
		{
			desc:            "Failed, even though service_control_network_fail_open is set as true, non-5xx check error won't be ignored.",
			networkFailOpen: true,
			checkResponse: scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_PROJECT_INVALID,
					},
				},
			},
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenLongClaims,
			wantError:      `400 Bad Request, {"code":400,"message":"INVALID_ARGUMENT:Client project not valid. Please pass a valid project."}`,
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(platform.TestServiceControlNetworkFailFlagForUnavailableCheckResponse, platform.GrpcBookstoreSidecar)
			s.ServiceControlServer.SetCheckResponse(&tc.checkResponse)
			if tc.networkFailOpen {
				s.EnableScNetworkFailOpen()
			}

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, nil)

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}
