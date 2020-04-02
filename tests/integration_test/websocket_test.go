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

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestWebsocket(t *testing.T) {
	t.Parallel()
	s := env.NewTestEnv(comp.TestWebsocket, platform.EchoSidecar)
	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc         string
		path         string
		messageCount int
		schema       string
		wantResp     string
	}{
		{
			desc:         "Websocket call succeed",
			path:         "/websocketecho",
			schema:       "ws",
			messageCount: 5,
			wantResp:     "hellohellohellohellohello",
		},
		{
			desc:     "normal http call succeed, not affected by websocket config",
			path:     "/echo?key=api_key",
			schema:   "http",
			wantResp: `{"message":"hello"}`,
		},
	}

	for _, tc := range testData {
		var resp []byte
		var err error
		if tc.schema == "ws" {
			resp, err = client.DoWS(fmt.Sprintf("localhost:%v", s.Ports().ListenerPort), tc.path, "hello", tc.messageCount)
		} else {
			resp, err = client.DoPost(fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path), "hello")
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
		}
	}
}
