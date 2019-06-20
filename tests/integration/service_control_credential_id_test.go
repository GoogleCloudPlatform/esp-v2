// Copyright 2019 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	bsClient "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlCredentialId(t *testing.T) {
	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers",
	}
	s := env.NewTestEnv(comp.TestServiceControlLogJwtPayloads, "bookstore", []string{"google_jwt"})

	s.OverrideAuthentication(&conf.Authentication{Rules: []*conf.AuthenticationRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
			Requirements: []*conf.AuthRequirement{
				{
					ProviderId: "google_jwt",
				},
			},
		},
	},
	})

	s.AppendUsageRules([]*conf.UsageRule{
		{
			Selector:               "endpoints.examples.bookstore.Bookstore.ListShelves",
			AllowUnregisteredCalls: true,
		},
	})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc                  string
		clientProtocol        string
		method                string
		httpMethod            string
		token                 string
		requestHeader         map[string]string
		message               string
		usageRules            []*conf.UsageRule
		authenticationRules   []*conf.AuthenticationRule
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:           "success; When api_key is unavaliable, the label credential_id is iss and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudToken,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
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
					RequestSize:       336,
					ResponseSize:      291,
					RequestBytes:      336,
					ResponseBytes:     291,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "success; When api_key is unavaliable, the label credential_id is iss plus aud and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudTokenSingleAudience1,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
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
					RequestSize:       390,
					ResponseSize:      291,
					RequestBytes:      390,
					ResponseBytes:     291,
					ResponseCode:      200,
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
