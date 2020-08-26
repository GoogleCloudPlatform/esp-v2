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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestWebsocket(t *testing.T) {
	t.Parallel()
	s := env.NewTestEnv(comp.TestWebsocket, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                 string
		path                 string
		query                string
		header               map[string][]string
		messageCount         int
		schema               string
		wantResp             string
		wantScRequests       []interface{}
		wantSkipScRequestNum int
	}{
		{
			desc:  "Websocket call succeed with service control check and jwt authn",
			path:  "/websocketecho",
			query: "key=api-key",
			header: map[string][]string{
				"Authorization": {
					"Bearer " + testdata.FakeCloudTokenMultiAudiences,
				},
			},
			schema:       "ws",
			messageCount: 5,
			wantResp:     "hellohellohellohellohello",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/websocketecho?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "GET",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho is called",
					StatusCode:                   "0",
					ResponseCode:                 101,
					Platform:                     util.GCE,
					Location:                     "test-zone",
					ApiVersion:                   "1.0.0",
				},
			},
		},
		{
			desc:                 "normal http call succeed, not affected by websocket config",
			path:                 "/echo",
			query:                "key=api_key",
			schema:               "http",
			wantResp:             `{"message":"hello"}`,
			wantSkipScRequestNum: 2,
		},
	}

	for _, tc := range testData {
		var resp []byte
		var err error
		if tc.schema == "ws" {
			resp, err = client.DoWS(fmt.Sprintf("localhost:%v", s.Ports().ListenerPort), tc.path, tc.query, tc.header, "hello", tc.messageCount)
		} else {
			resp, err = client.DoPost(fmt.Sprintf("http://localhost:%v%v?%s", s.Ports().ListenerPort, tc.path, tc.query), "hello")
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
		}

		if tc.wantSkipScRequestNum != 0 {
			_, _ = s.ServiceControlServer.GetRequests(tc.wantSkipScRequestNum)
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
