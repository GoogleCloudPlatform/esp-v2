// Copyright 2021 Google LLC
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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestAddHeaders(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc           string
		headerFlags    []string
		requestHeader  map[string]string
		wantRespHeader map[string]string
	}{
		{
			desc: "add single request header",
			headerFlags: []string{
				"--add_request_headers=key1=value1",
			},
			wantRespHeader: map[string]string{
				"Echo-Key1": "value1",
			},
		},
		{
			desc: "add two request headers, second one exists",
			headerFlags: []string{
				"--add_request_headers=key1=value1;key2=new-value2",
			},
			requestHeader: map[string]string{
				"key2": "old-value2",
			},
			wantRespHeader: map[string]string{
				"Echo-Key1": "value1",
				"Echo-Key2": "new-value2",
			},
		},
		{
			desc: "append two request headers, second one exists",
			headerFlags: []string{
				"--append_request_headers=key1=value1;key2=new-value2",
			},
			requestHeader: map[string]string{
				"key2": "old-value2",
			},
			wantRespHeader: map[string]string{
				"Echo-Key1": "value1",
				"Echo-Key2": "old-value2;new-value2",
			},
		},
		{
			desc: "add single response header",
			headerFlags: []string{
				"--add_response_headers=key1=value1",
			},
			wantRespHeader: map[string]string{
				"Key1": "value1",
			},
		},
		{
			desc: "add two response headers, second one exists",
			headerFlags: []string{
				"--add_response_headers=key1=value1;Echo-Key2=new-value2",
			},
			requestHeader: map[string]string{
				"key2": "old-value2",
			},
			wantRespHeader: map[string]string{
				"Key1":      "value1",
				"Echo-Key2": "new-value2",
			},
		},
		{
			desc: "append two response headers, second one exists",
			headerFlags: []string{
				"--append_response_headers=key1=value1;Echo-Key2=new-value2",
			},
			requestHeader: map[string]string{
				"key2": "old-value2",
			},
			wantRespHeader: map[string]string{
				"Key1":      "value1",
				"Echo-Key2": "old-value2;new-value2",
			},
		},
	}
	for _, tc := range testData {
		func() {
			args := append(utils.CommonArgs(), tc.headerFlags...)

			s := env.NewTestEnv(platform.TestAddHeaders, platform.EchoSidecar)
			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoHeader", "?key=api-key")
			headers, _, err := utils.DoWithHeaders(url, "GET", "", tc.requestHeader)
			if err != nil {
				t.Errorf("fail to make request: %v", err)
			}

			for wantHeaderName, wantHeaderVal := range tc.wantRespHeader {
				if !utils.CheckHeaderExist(headers, wantHeaderName, func(gotHeaderVal string) bool {
					return wantHeaderVal == gotHeaderVal
				}) {
					t.Errorf("Test (%s): get headers %v, not find expected header %s:%s,  ", tc.desc, headers, wantHeaderName, wantHeaderVal)
				}
			}
		}()

	}
}
