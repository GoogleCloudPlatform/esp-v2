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

package service_control_report_network_fail_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestServiceControlReportNetworkFail(t *testing.T) {

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--service_control_report_retries=0"}

	s := env.NewTestEnv(comp.TestServiceControlReportNetworkFail, "bookstore")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	time.Sleep(time.Duration(5 * time.Second))
	tests := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		reportFailed       bool
		wantResp           string
		wantError          string
		wantScRequestCount int
	}{
		{
			desc:               "Success, the request had a successful check, a successful report and a correct backend response normally",
			clientProtocol:     "http",
			httpMethod:         "GET",
			method:             "/v1/shelves/100?key=api-key-1",
			wantResp:           `{"id":"100","theme":"Kids"}`,
			wantScRequestCount: 2,
		},
		{
			desc:               "Succeed, the request had a failed report but still got the correct backend response",
			clientProtocol:     "http",
			httpMethod:         "GET",
			reportFailed:       true,
			method:             "/v1/shelves/100?key=api-key-2",
			wantResp:           `{"id":"100","theme":"Kids"}`,
			wantScRequestCount: 2,
		},
	}

	for _, tc := range tests {
		if tc.reportFailed {
			s.ServiceControlServer.SetReportResponseStatus(http.StatusInternalServerError)
		}
		s.ServiceControlServer.ResetRequestCount()
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsclient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", nil)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}

		err = s.ServiceControlServer.VerifyRequestCount(tc.wantScRequestCount)
		if err != nil {
			t.Fatalf("Test (%s): failed, %s", tc.desc, err.Error())
		}
	}
}
