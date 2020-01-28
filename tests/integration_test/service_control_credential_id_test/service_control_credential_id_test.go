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
package service_control_credential_id_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsClient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlCredentialId(t *testing.T) {

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers",
	}
	s := env.NewTestEnv(comp.TestServiceControlCredentialId, platform.GrpcBookstoreSidecar)

	s.OverrideAuthentication(&confpb.Authentication{Rules: []*confpb.AuthenticationRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
			Requirements: []*confpb.AuthRequirement{
				{
					ProviderId: testdata.GoogleJwtProvider,
				},
			},
		},
	},
	})

	s.AppendUsageRules([]*confpb.UsageRule{
		{
			Selector:               "endpoints.examples.bookstore.Bookstore.ListShelves",
			AllowUnregisteredCalls: true,
		},
	})

	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                  string
		clientProtocol        string
		method                string
		httpMethod            string
		token                 string
		requestHeader         map[string]string
		message               string
		usageRules            []*confpb.UsageRule
		authenticationRules   []*confpb.AuthenticationRule
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:           "success; When api_key is unavailable, the label credential_id is iss and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudToken,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves",
					JwtAuth:           "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "success; When api_key is unavailable, the label credential_id is iss plus aud and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudTokenSingleAudience1,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves",
					JwtAuth:           "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw&audience=Ym9va3N0b3JlX3Rlc3RfY2xpZW50LmNsb3VkLmdvb2c",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if tc.httpCallError.Error() != err.Error() {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1)
			if err1.Error() != tc.wantGetScRequestError.Error() {
				t.Errorf("Test (%s): failed", tc.desc)
				t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err1)
				t.Errorf("got service control requests: %v", scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
