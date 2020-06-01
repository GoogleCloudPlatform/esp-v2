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
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bookstore "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	echoclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestServiceControlProtocolWithGRPCBackend(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"

	args := []string{
		"--service_config_id=" + configID,

		"--rollout_strategy=fixed",
	}

	headerWithAPIKey := http.Header{bookstore.APIKeyHeaderKey: []string{"foobar"}}

	s := env.NewTestEnv(comp.TestServiceControlProtocolWithGRPCBackend, platform.GrpcBookstoreSidecar)

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc                 string
		method               string
		protocol             string
		wantFrontendProtocol string
		// if frontend and backend match, the service_control filter does not add
		// `backend_protocol` to the ReportRequest. Only check for this
		// if we know service_control will provide it.
		wantBackendProtocol string
		numRequestsToSkip   int
	}{
		{
			desc:                 "http for frontend protocol",
			method:               "/v1/shelves/100",
			protocol:             "http",
			wantFrontendProtocol: "http",
			wantBackendProtocol:  "grpc",
			// HTTP requests go through CheckRequest before ReportRequest
			numRequestsToSkip: 1,
		},
		{
			desc:                 "grpc for frontend protocol",
			method:               "GetShelf", // makeGRPCCall sets shelf=100 automatically
			protocol:             "grpc",
			wantFrontendProtocol: "grpc",
		},
		{
			desc:                 "grpc-web for frontend protocol",
			method:               "GetShelf", // MakeGRPCWebCall sets shelf=100 automatically
			protocol:             "grpc-web",
			wantFrontendProtocol: "grpc",
		},
	}

	for _, tc := range tests {
		wantResp := `{"id":"100","theme":"Kids"}`
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)

		var resp string
		var err error
		if tc.protocol == "grpc-web" {
			wantTrailer := bookstore.GRPCWebTrailer{"grpc-message": "", "grpc-status": "0"}
			var trailer bookstore.GRPCWebTrailer
			resp, trailer, err = bookstore.MakeGRPCWebCall(addr, tc.method, "", headerWithAPIKey)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(trailer, wantTrailer) {
				t.Errorf("Test (%s): GRPCWebRequest failed, expected: %s, got: %s", tc.desc, wantTrailer, trailer)
			}
		} else {
			resp, err = bookstore.MakeCall(tc.protocol, addr, "GET", tc.method, "", headerWithAPIKey)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !strings.Contains(resp, wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, wantResp, resp)
			break
		}

		var body []byte
		scRequests, err := s.ServiceControlServer.GetRequests(1 + tc.numRequestsToSkip)
		if err != nil {
			t.Fatal(err)
		}

		if scRequests[tc.numRequestsToSkip].ReqType != utils.ReportRequest {
			t.Fatalf("Test (%s): Expected but did not get a ReportRequest", tc.desc)
		}

		body = scRequests[tc.numRequestsToSkip].ReqBody

		if err := utils.VerifyReportRequestOperationLabel(body, "/protocol", tc.wantFrontendProtocol); err != nil {
			t.Errorf("Test (%s): Failed to verify frontend protocol, %v", tc.desc, err)
		}

		err = utils.VerifyReportRequestOperationLabel(body,
			"servicecontrol.googleapis.com/backend_protocol", "grpc")

		if tc.wantBackendProtocol == "" {
			if err == nil ||
				err.Error() != "No operations contained label servicecontrol.googleapis.com/backend_protocol" {
				t.Errorf("Test (%s): Expected no backend protocol, got, %v", tc.desc, err)
			}
		} else if err != nil {
			t.Errorf("Test (%s): Failed to verify backend protocol, %v", tc.desc, err)
		}
	}
}

func TestServiceControlProtocolWithHTTPBackend(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configID := "test-config-id"

	args := []string{
		"--service=" + serviceName,
		"--service_config_id=" + configID,

		"--rollout_strategy=fixed",
	}

	s := env.NewTestEnv(comp.TestServiceControlProtocolWithHTTPBackend, platform.EchoSidecar)

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	desc := "http for frontend protocol"
	protocol := "http"
	message := "hello"
	wantResp := `{"message":"hello"}`
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey")

	resp, err := echoclient.DoPost(url, message)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resp), wantResp) {
		t.Errorf("expected: %s, got: %s", wantResp, string(resp))
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatal(err)
	}

	if scRequests[0].ReqType != utils.ReportRequest {
		t.Fatalf("Test (%s): Expected but did not get a ReportRequest", desc)
	}

	body := scRequests[0].ReqBody

	if err := utils.VerifyReportRequestOperationLabel(body, "/protocol", protocol); err != nil {
		t.Errorf("Test (%s): Failed to verify frontend protocol, %v", desc, err)
	}

	// if frontend and backend match, the service_control filter does not add
	// `backend_protocol` to the ReportRequest. Since only the http frontend can
	// communicate with an http backend, they must match, so this is not set.
	err = utils.VerifyReportRequestOperationLabel(body,
		"servicecontrol.googleapis.com/backend_protocol", "http")
	if err == nil {
		t.Errorf("Test (%s): Expected no backend protocol, but got one, %v", desc, err)
	}
	if err.Error() != "No operations contained label servicecontrol.googleapis.com/backend_protocol" {
		t.Errorf("Test (%s): Wrong error. Expected No operations contained label, got, %v", desc, err)
	}
}
