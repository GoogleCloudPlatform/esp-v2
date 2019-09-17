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
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestAsymmetricKeys(t *testing.T) {
	t.Parallel()
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestAsymmetricKeys, "bookstore")
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
					{
						ProviderId: "invalid_jwks_provider",
						Audiences:  "bookstore_test_client.cloud.goog",
					},
					{
						ProviderId: "nonexist_jwks_provider",
						Audiences:  "bookstore_test_client.cloud.goog",
					},
					{
						ProviderId: "openID_provider",
						Audiences:  "ok_audience",
					},
					{
						ProviderId: "openID_invalid_provider",
						Audiences:  "ok_audience",
					},
					{
						ProviderId: "openID_nonexist_provider",
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	time.Sleep(time.Duration(5 * time.Second))
	tests := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		queryInToken       bool
		token              string
		headers            map[string][]string
		wantResp           string
		wantError          string
		wantGRPCWebError   string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}{
		{
			desc:           "Failed, no JWT passed in.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantError:      "401 Unauthorized, Jwt is missing",
		},
		{
			desc:           "Succeeded, with right token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Es256Token,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Succeeded, wth jwt token passed in \"Authorization: Bearer\" header",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Rs256Token,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Succeeded, wth jwt token passed in \"x-goog-iap-jwt-assertion\" header",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			headers: map[string][]string{
				"x-goog-iap-jwt-assertion": []string{testdata.Rs256Token},
			},
			wantResp: `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Succeeded, with jwt token passed in query",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key&access_token=" + testdata.Rs256Token,
			queryInToken:   true,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Failed, provider providing wrong-format jwks",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeInvalidJwksProviderToken,
			wantError:      "401 Unauthorized, Jwks remote fetch is failed",
		},
		{
			desc:           "Failed, provider not existing",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeNonexistJwksProviderToken,
			wantError:      "401 Unauthorized, Jwks remote fetch is failed",
		},
		{
			desc:           "Succeeded, using openID discovery",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeOpenIDToken,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Failed, the provider found by openID discovery providing invalid jwks",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeInvalidOpenIDToken,
			// Note: The detailed error should be Jwks remote fetch is failed while envoy may inaccurate
			// detailed error(issuer is not configured).
			wantError: "401 Unauthorized",
		},
		{
			desc:           "Failed, the provider got by openID discover not existing",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeNonexistOpenIDToken,
			// Note: The detailed error should be Jwks remote fetch is failed while envoy may inaccurate
			// detailed error(issuer is not configured).
			wantError: "401 Unauthorized",
		},
	}

	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		var resp string
		var err error
		if tc.queryInToken {
			resp, err = client.MakeTokenInQueryCall(addr, tc.httpMethod, tc.method, tc.token)
		} else {
			resp, err = client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)
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
