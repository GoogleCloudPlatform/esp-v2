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

package client_cancellation_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

// Tests the SC report when a client cancels a request.
func TestCancellationReport(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestCancellationReport, platform.EchoRemote)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                 string
		backendSleepDuration time.Duration
		clientTimeout        time.Duration
		wantErr              string
		wantScRequests       []interface{}
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc:                 "Client timeout causes request cancellation before backend responds",
			backendSleepDuration: 10 * time.Second,
			clientTimeout:        5 * time.Second,
			wantErr:              `Client.Timeout exceeded`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/sleepDefault?duration=10s",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationDefault",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:        "1.0.0",
					ApiKeyState:       "NOT CHECKED",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationDefault is called",
					StatusCode:        "0",
					// Final status code is not present because request was cancelled.
					ResponseCode:   0,
					HttpStatusCode: 0,
					// Response code confirms cancellation.
					ResponseCodeDetail: "downstream_remote_disconnect",
					Platform:           util.GCE,
					Location:           "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

			path := fmt.Sprintf("/sleepDefault?duration=%v", tc.backendSleepDuration.String())
			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, path)

			_, err := client.DoWithHeadersAndTimeout(url, "GET", "", nil, tc.clientTimeout)

			if tc.wantErr == "" && err != nil {
				t.Errorf("Test (%s): failed, expected no err, got err (%v)", tc.desc, err)
			}

			if tc.wantErr != "" && err == nil {
				t.Errorf("Test (%s): failed, got no err, expected err (%v)", tc.desc, tc.wantErr)
			}

			if err != nil && !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Test (%s): failed, got err (%v), expected err (%v)", tc.desc, err, tc.wantErr)
			}

			scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err1 != nil {
				t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
			}
			utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
		})
	}
}
