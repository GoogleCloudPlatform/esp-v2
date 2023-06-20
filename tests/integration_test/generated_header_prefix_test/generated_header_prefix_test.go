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

package generated_header_prefix_test

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

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
				"Echo-X-Endpoint-Api-Consumer-Type":   "PROJECT",
				"Echo-X-Endpoint-Api-Consumer-Number": "123456",
				"Echo-X-Endpoint-Api-Userinfo":        testdata.Es256TokenPayloadBase64,
			},
		},
		{
			desc:                     "use customized --generated_header_prefix=X-Apigateway-",
			generatedHeaderPrefixArg: "--generated_header_prefix=x-apigateway-",
			requestHeader: map[string]string{
				"Authorization": "Bearer " + testdata.Es256Token,
			},
			wantRespHeader: map[string]string{
				"Echo-X-Apigateway-Api-Consumer-Type":   "PROJECT",
				"Echo-X-Apigateway-Api-Consumer-Number": "123456",
				"Echo-X-Apigateway-Api-Userinfo":        testdata.Es256TokenPayloadBase64,
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			args := []string{"--service_config_id=test-config-id",
				"--rollout_strategy=fixed", "--suppress_envoy_headers"}
			if tc.generatedHeaderPrefixArg != "" {
				args = append(args, tc.generatedHeaderPrefixArg)
			}

			s := env.NewTestEnv(platform.TestGeneratedHeaders, platform.EchoSidecar)
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

			url := fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader", "?key=api-key-2")
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
		})
	}
}

func TestOperationNameHeader(t *testing.T) {
	t.Parallel()
	operationName := "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader"

	testData := []struct {
		desc           string
		confArgs       []string
		requestHeader  map[string]string
		wantRespHeader map[string]string
	}{
		{
			desc: "Enable generated operation name header",
			confArgs: append([]string{
				"--enable_operation_name_header",
			}, utils.CommonArgs()...),
			wantRespHeader: map[string]string{
				"Echo-X-Endpoint-Api-Operation-Name": operationName,
			},
		},
		{
			desc: "Enable generated operation name header with customized prefix",
			confArgs: append([]string{
				"--enable_operation_name_header",
				"--generated_header_prefix=x-apigateway-",
			}, utils.CommonArgs()...),
			wantRespHeader: map[string]string{
				"Echo-X-Apigateway-Api-Operation-Name": operationName,
			},
		},
		{
			desc: "Enable generated operation name header, overwrites existing one",
			confArgs: append([]string{
				"--enable_operation_name_header",
			}, utils.CommonArgs()...),
			requestHeader: map[string]string{
				"Echo-X-Endpoint-Api-Operation-Name": "bad-value-set-by-client",
			},
			wantRespHeader: map[string]string{
				"Echo-X-Endpoint-Api-Operation-Name": operationName,
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestOperationNameHeader, platform.EchoSidecar)

			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://%v:%v%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echoHeader", "?key=api-key-2")
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
		})
	}
}
