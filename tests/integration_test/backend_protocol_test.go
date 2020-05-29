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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestBackendHttpProtocol(t *testing.T) {

	testData := []struct {
		desc           string
		backendIsHttp2 bool
		configHttp2    bool
		wantResp       string
		httpCallError  error
	}{
		{
			desc:           "Success when backend is http/1 only and envoy is configured for http/1 backend",
			backendIsHttp2: false,
			configHttp2:    false,
			wantResp:       `{"message":"hello"}`,
		},
		{
			desc:           "Success when backend is http/2 and envoy is configured for http/1 backend",
			backendIsHttp2: true,
			configHttp2:    false,
			wantResp:       `{"message":"hello"}`,
		},
		{
			desc:           "Success when backend is http/2 and envoy is configured for http/2 backend",
			backendIsHttp2: true,
			configHttp2:    true,
			wantResp:       `{"message":"hello"}`,
		},
		{
			desc:           "Failure when backend is http/1 only and envoy is configured for http/2 backend",
			backendIsHttp2: false,
			configHttp2:    true,
			httpCallError:  fmt.Errorf("503 Service Unavailable, upstream connect error or disconnect/reset before headers. reset reason: connection termination"),
		},
	}
	for _, tc := range testData {
		// Place in closure to allow deferring in loop.
		func() {

			httpProtocol := "http/1.1"
			if tc.configHttp2 {
				httpProtocol = "h2"
			}

			// Setup the protocol in the backend rule for the endpoint under test.
			s := env.NewTestEnv(comp.TestBackendHttpProtocol, platform.EchoRemote)
			s.RemoveAllBackendRules()
			s.AppendBackendRules([]*confpb.BackendRule{
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Address:         "https://localhost:-1/echo",
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Protocol:        httpProtocol,
				},
			})

			// Explicitly setup which protocol the echo backend operates under.
			if !tc.backendIsHttp2 {
				s.DisableHttp2ForHttpsBackend()
			}

			// Setup test env.
			defer s.TearDown(t)
			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			// Do test.
			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo?key=api-key")
			gotResp, err := client.DoWithHeaders(url, "POST", "hello", nil)

			// Assertions.
			if tc.httpCallError != nil {
				// Expect an error.
				if err == nil {
					t.Errorf("Test(%s) expected error: %v, got: none", tc.desc, tc.httpCallError)
				} else if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
					t.Errorf("Test(%s) expected error: %v, got: %v", tc.desc, tc.httpCallError, err)
				}
			} else {
				// Expect success.
				if err != nil {
					t.Errorf("Test(%s) expected success, got err: %v", tc.desc, err)
				} else if err := util.JsonEqual(tc.wantResp, string(gotResp)); err != nil {
					t.Errorf("Test(%s) expected success: \n %s", tc.desc, err)
				}
			}

		}()
	}
}
