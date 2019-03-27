// Copyright 2018 Google Cloud Platform Proxy Authors
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
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	testdata "cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestAsymmetricKeys(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestAsymmetricKeys, "bookstore", []string{"test_auth", "test_auth_1"})
	s.OverrideAuthentication(&conf.Authentication{
		Rules: []*conf.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*conf.AuthRequirement{
					{
						ProviderId: "test_auth",
						Audiences:  "ok_audience",
					},
					{
						ProviderId: "test_auth_1",
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		queryInToken       bool
		token              string
		wantResp           string
		wantError          string
		wantGRPCWebError   string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}{
		{
			desc:           "No JWT passed in.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantError:      "401 Unauthorized, Jwt is missing",
		},
		{
			desc:           "ES256Token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Es256Token,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "RS256-signed jwt token is passed in \"Authorization: Bearer\" header",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Rs256Token,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "RS256-signed jwt token is passed in query",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key&access_token=" + testdata.Rs256Token,
			queryInToken:   true,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
	}

	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		var resp string
		var err error
		if tc.queryInToken {
			resp, err = client.MakeTokenInQueryCall(addr, tc.httpMethod, tc.method, tc.token)
		} else {
			resp, err = client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})
		}

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
