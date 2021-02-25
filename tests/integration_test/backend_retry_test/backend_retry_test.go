// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0 //
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend_retry_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	defaultBackendRetryOns = "reset,connect-failure,refused-stream"
	defaultBackendRetryNum = 1
)

type backendType int32

const (
	remoteBackend backendType = iota
	localBackend
)

func TestBackendRetry(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc                       string
		requestHeader              map[string]string
		backendRespondRST          bool
		backendNotStart            bool
		backendRejectRequestNum    int
		backendRejectRequestStatus int
		backendRetryOnsFlag        string
		backendRetryNumFlag        int
		backendTypeUsed            backendType
		message                    string
		wantResp                   string
		wantError                  string
		wantSpanNames              []string
	}{
		{
			desc:                "Failed request for local backend, reproduce the case that upstream keeps sending RST in TCP connection and ESPv2 doesn't do retry",
			backendRespondRST:   true,
			backendRetryOnsFlag: "",
			backendRetryNumFlag: 0,
			backendTypeUsed:     localBackend,
			wantError:           `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: connection termination`,
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		// Hard to control making the successful TCP connection under certain retry
		// by sending certain RST so this test case is to see ESPv2 are doing retry
		// under reset.
		{
			desc:                "Failed request for local backend, simulate the scenario that upstream keeps sending RST in TCP connection and ESPv2 do 1 retry under the default retryOns(reset)",
			backendRespondRST:   true,
			backendRetryOnsFlag: defaultBackendRetryOns,
			backendRetryNumFlag: defaultBackendRetryNum,
			backendTypeUsed:     localBackend,
			// It could be `connection termination` or `connection failure` in the end
			//so don't specify it here.
			wantError: `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: connection failure`,
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                "Failed request for local backend, simulate the scenario that upstream cannot be connected and ESPv2 doesn't do retry",
			backendNotStart:     true,
			backendRetryOnsFlag: "",
			backendRetryNumFlag: 0,
			backendTypeUsed:     localBackend,
			wantError:           `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: connection failure`,
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		// Hard to control backend to launch right after certain retry so this test
		// case is to see ESPv2 are doing retry under connection failure.
		{
			desc:                "Failed request for local backend, simulate the scenario that upstream cannot be connected and and ESPv2 do 1 retry under the default retryOns(connect-failure)",
			backendNotStart:     true,
			backendRetryOnsFlag: defaultBackendRetryOns,
			backendRetryNumFlag: defaultBackendRetryNum,
			backendTypeUsed:     localBackend,
			wantError:           `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: connection failure`,
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for local backend, the retryNum is not enough",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        1,
			backendTypeUsed:            localBackend,
			wantError:                  "503 Service Unavailable",
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for local backend, invalid retryOns",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "this-is-random-retryOn",
			backendRetryNumFlag:        2,
			backendTypeUsed:            localBackend,
			wantError:                  "503 Service Unavailable",
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for local backend, no retry as 5xx doesn't cover 403",
			backendRejectRequestStatus: 403,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        2,
			backendTypeUsed:            localBackend,
			wantError:                  "403 Forbidden",
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Successful reuqest for local backend, sufficient retryNum and covered retryOn",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        2,
			backendTypeUsed:            localBackend,
			wantResp:                   `{"message":""}`,
			wantSpanNames: []string{
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"router backend-cluster-echo-api.endpoints.cloudesf-testing.cloud.goog_local egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for remote backend, the retryNum is not enough",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        1,
			backendTypeUsed:            remoteBackend,
			wantError:                  "503 Service Unavailable",
			wantSpanNames: []string{
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for remote backend, invalid retryOns",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "this-is-random-retryOn",
			backendRetryNumFlag:        2,
			backendTypeUsed:            remoteBackend,
			wantError:                  "503 Service Unavailable",
			wantSpanNames: []string{
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Failed request for remote backend, no retry as 5xx doesn't cover 403",
			backendRejectRequestStatus: 403,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        2,
			backendTypeUsed:            remoteBackend,
			wantError:                  "403 Forbidden",
			wantSpanNames: []string{
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"ingress Echo",
			},
		},
		{
			desc:                       "Successful request for remote backend, sufficient retryNum and covered retryOn",
			backendRejectRequestStatus: 503,
			backendRejectRequestNum:    2,
			backendRetryOnsFlag:        "5xx",
			backendRetryNumFlag:        2,
			backendTypeUsed:            remoteBackend,
			wantResp:                   `{"message":""}`,
			wantSpanNames: []string{
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"router backend-cluster-localhost:BACKEND_PORT egress",
				"ingress Echo",
			},
		},
	}
	for _, tc := range testData {
		func() {
			configId := "test-config-id"
			args := []string{
				"--service_config_id=" + configId,
				"--rollout_strategy=fixed",
				"--suppress_envoy_headers",
			}
			if tc.backendRetryOnsFlag != defaultBackendRetryOns {
				args = append(args, "--backend_retry_ons="+tc.backendRetryOnsFlag)
			}

			if tc.backendRetryNumFlag != defaultBackendRetryNum {
				args = append(args, fmt.Sprintf("--backend_retry_num=%v", tc.backendRetryNumFlag))
			}

			var backendUsed platform.Backend
			if tc.backendTypeUsed == localBackend {
				backendUsed = platform.EchoSidecar
			} else if tc.backendTypeUsed == remoteBackend {
				backendUsed = platform.EchoRemote
			}

			s := env.NewTestEnv(platform.TestBackendRetry, backendUsed)
			s.AppendUsageRules(
				[]*confpb.UsageRule{
					{
						Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
						AllowUnregisteredCalls: true,
					}})
			s.SetBackendAlwaysRespondRST(tc.backendRespondRST)
			s.SetBackendNotStart(tc.backendNotStart)
			s.SetBackendRejectRequestNum(tc.backendRejectRequestNum)
			s.SetBackendRejectRequestStatus(tc.backendRejectRequestStatus)
			s.SetupFakeTraceServer(1)
			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			resp, err := client.DoWithHeaders(fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo"), util.POST, tc.message, tc.requestHeader)
			respStr := string(resp)
			if !strings.Contains(respStr, tc.wantResp) {
				t.Errorf("Test (%s) failed, want resp %s, get resp %s", tc.desc, tc.wantResp, respStr)
			}

			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Test (%s) failed, want error %s, get error %v", tc.desc, tc.wantError, err)
				}
			} else if err != nil {
				t.Errorf("Test (%s) failed, get unexpected error %v", tc.desc, err)
			}

			time.Sleep(time.Second * 5)
			gotSpanNames, _ := s.FakeStackdriverServer.RetrieveSpanNames()
			realWantSpanNames := tc.wantSpanNames
			if tc.backendTypeUsed == remoteBackend {
				realWantSpanNames = replaceBackendPort(tc.wantSpanNames, strconv.Itoa(int(s.Ports().DynamicRoutingBackendPort)))
			}
			if !reflect.DeepEqual(gotSpanNames, realWantSpanNames) {
				t.Errorf("Test (%s) failed, got span names: %+q, want span names: %+q", tc.desc, gotSpanNames, realWantSpanNames)
			}

		}()
	}
}

func replaceBackendPort(spanNames []string, port string) []string {
	var replacedSpanNames []string
	for _, spanName := range spanNames {
		replacedSpanNames = append(replacedSpanNames, strings.ReplaceAll(spanName, "BACKEND_PORT", port))
	}

	return replacedSpanNames
}
