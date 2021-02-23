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

package jwt_locations_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestJwtLocations(t *testing.T) {
	t.Parallel()
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestJwtLocations, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuth1Provider,
						Audiences:  "ok_audience",
					},
				},
			},
			{
				Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.CustomJwtLocationProvider,
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

	var tests = []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		headers            map[string][]string
		wantResp           string
		wantError          string
		wantGRPCWebError   string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}{
		{
			desc:           "Success. Jwt token is passed in default \"Authorization: Bearer\" header",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			headers: map[string][]string{
				"Authorization": {"Bearer " + testdata.Rs256Token},
			},
			wantResp: `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Success. Jwt token is passed in default \"x-goog-iap-jwt-assertion\" header",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			headers: map[string][]string{
				"x-goog-iap-jwt-assertion": {testdata.Rs256Token},
			},
			wantResp: `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Success. Jwt token is passed in default in query param \"access_token\"",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key&access_token=" + testdata.Rs256Token,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Success. Jwt token is passed in custom location jwt-header-foo",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key",
			headers: map[string][]string{
				"jwt-header-foo": {"jwt-prefix-foo " + testdata.Rs256Token},
			},
			wantResp: `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "Success. Jwt token is passed in custom location jwt-header-bar",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key",
			headers: map[string][]string{
				"jwt-header-bar": {"jwt-prefix-bar " + testdata.Rs256Token},
			},
			wantResp: `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "Success. Jwt token is passed in custom location jwt-param-foo",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key&jwt-param-foo=" + testdata.Rs256Token,
			wantResp:       `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "Success. Jwt token is passed in custom location jwt-param-foo",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key&jwt-param-bar=" + testdata.Rs256Token,
			wantResp:       `{"id":"100","theme":"Kids"}`,
		},
		{
			desc:           "Failure. Jwt token is passed in default \"Authorization: Bearer\" header for the customized jwt locations",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key",
			headers: map[string][]string{
				"Authorization": {"Bearer " + testdata.Rs256Token},
			},
			wantError: `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
		},
		{
			desc:           "Failure. Jwt token is passed in default \"x-goog-iap-jwt-assertion\" header for the customized jwt locations",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key",
			headers: map[string][]string{
				"x-goog-iap-jwt-assertion": {testdata.Rs256Token},
			},
			wantError: `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
		},
		{
			desc:           "Failure. Jwt token is passed in default in query param \"access_token\" for the customized jwt locations",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key&access_token=" + testdata.Rs256Token,
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
		},
	}
	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, "", tc.headers)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else if tc.wantError == "" && err != nil {
			t.Errorf("Test (%s): failed, expected no error, got error: %s", tc.desc, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
