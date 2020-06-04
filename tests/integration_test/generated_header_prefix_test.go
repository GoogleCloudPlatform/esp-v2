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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func checkHeaderExist(headers http.Header, wantHeaderName, wantHeaderVal string) bool {
	for headerName, headerVals := range headers {
		if headerName == wantHeaderName {
			if len(headerVals) > 0 || headerVals[0] == wantHeaderVal {
				return true
			}
		}
	}
	return false
}

func TestGeneratedHeaders(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc                     string
		generatedHeaderPrefixArg string
		requestHeader            map[string]string
		wantRespHeader           map[string]string
	}{
		{
			desc: "use default --generated_header_prefix=X-Endpoint-",
			requestHeader: map[string]string{
				"Authorization": "Bearer " + testdata.Es256Token,
			},
			wantRespHeader: map[string]string{
				"Echo-X-Endpoint-Api-Project-Number": "123456",
				"Echo-X-Endpoint-Api-Userinfo":       testdata.Es256TokenPayloadBase64,
			},
		},
		{
			desc:                     "use customized --generated_header_prefix=X-Apigateway-",
			generatedHeaderPrefixArg: "--generated_header_prefix=x-apigateway-",
			requestHeader: map[string]string{
				"Authorization": "Bearer " + testdata.Es256Token,
			},
			wantRespHeader: map[string]string{
				"Echo-X-Apigateway-Api-Project-Number": "123456",
				"Echo-X-Apigateway-Api-Userinfo":       testdata.Es256TokenPayloadBase64,
			},
		},
	}
	for _, tc := range testData {
		func() {
			args := []string{"--service_config_id=test-config-id",
				"--rollout_strategy=fixed", "--suppress_envoy_headers"}
			if tc.generatedHeaderPrefixArg != "" {
				args = append(args, tc.generatedHeaderPrefixArg)
			}

			s := env.NewTestEnv(comp.TestGeneratedHeaders, platform.EchoSidecar)
			s.OverrideAuthentication(&confpb.Authentication{
				Rules: []*confpb.AuthenticationRule{
					{
						Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader",
						Requirements: []*confpb.AuthRequirement{
							{
								ProviderId: testdata.TestAuthProvider,
								Audiences:  "ok_audience",
							},
						},
					},
				},
			})

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echoHeader", "?key=api-key-2")
			headers, _, err := utils.DoWithHeaders(url, "GET", "", tc.requestHeader)
			if err != nil {
				t.Errorf("fail to make request: %v", err)
			}

			for wantHeaderName, wantHeaderVal := range tc.wantRespHeader {
				if !checkHeaderExist(headers, wantHeaderName, wantHeaderVal) {
					t.Errorf("Test (%s): get headers %v, not find expected header %s:%s,  ", tc.desc, headers, wantHeaderName, wantHeaderVal)
				}
			}
		}()

	}
}
