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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	echo_client "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestAsymmetricKeys(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestAsymmetricKeys, platform.GrpcBookstoreSidecar)
	if err := s.FakeJwtService.SetupValidOpenId(); err != nil {
		t.Fatalf("fail to setup open id servers: %v", err)
	}
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
					{
						ProviderId: testdata.TestAuth1Provider,
						Audiences:  "ok_audience",
					},
					{
						ProviderId: testdata.InvalidProvider,
						Audiences:  "bookstore_test_client.cloud.goog",
					},
					{
						ProviderId: testdata.NonexistentProvider,
						Audiences:  "bookstore_test_client.cloud.goog",
					},
					{
						ProviderId: testdata.OpenIdProvider,
						Audiences:  "ok_audience",
					},
					{
						ProviderId: testdata.X509Provider,
						Audiences:  "fake.audience",
					},
				},
			},
		},
	})
	defer s.TearDown(t)
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
		{
			// Regression test for b/146942680
			desc:           "Succeeded for x509 public keys",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.X509Token,
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
			resp, err = client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)
		}

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

// Tests that config translation will fail when the OpenID Connect Discovery protocol is not followed.
func TestInvalidOpenIDConnectDiscovery(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	tests := []struct {
		desc        string
		providerId  string
		expectedErr string
	}{
		{
			desc:        "Fail with provider with invalid response",
			providerId:  testdata.OpenIdInvalidProvider,
			expectedErr: "health check response was not healthy",
		},
		{
			desc:        "Fail with provider that does not exist",
			providerId:  testdata.OpenIdNonexistentProvider,
			expectedErr: "health check response was not healthy",
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(comp.TestInvalidOpenIDConnectDiscovery, platform.GrpcBookstoreSidecar)
			if err := s.FakeJwtService.SetupInvalidOpenId(); err != nil {
				t.Fatalf("fail to setup open id servers: %v", err)
			}

			s.OverrideAuthentication(&confpb.Authentication{
				Rules: []*confpb.AuthenticationRule{
					{
						Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
						Requirements: []*confpb.AuthRequirement{
							{
								ProviderId: tc.providerId,
								Audiences:  "ok_audience",
							},
						},
					},
				},
			})

			err := s.Setup(args)

			// LIFO ordering. Disable health checks before teardown, we expect a failure.
			defer s.TearDown(t)
			defer s.SkipHealthChecks()

			if err == nil {
				t.Errorf("Test (%s): failed, expected error, got no err", tc.desc)
			} else if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("Test (%s): failed, expected err: %v, got err: %v", tc.desc, tc.expectedErr, err)
			}
		}()
	}
}

func TestFrontendAndBackendAuthHeaders(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(comp.TestFrontendAndBackendAuthHeaders, platform.EchoRemote)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
			{
				Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToTrue",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.BackendAuthToken",
		}, 0)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc        string
		method      string
		path        string
		headers     map[string]string
		wantHeaders map[string]string
	}{
		{
			desc: "Frontend auth preserves `Authorization` and creates `X-Endpoint-API-UserInfo`." +
				"Backend auth is disabled, so no further header modifications.",
			method: "GET",
			path:   "/disableauthsettotrue/constant/disableauthsettotrue",
			headers: map[string]string{
				"Authorization": "Bearer " + testdata.Es256Token,
			},
			wantHeaders: map[string]string{
				"Authorization":           "Bearer " + testdata.Es256Token,
				"X-Endpoint-API-UserInfo": testdata.Es256TokenPayloadBase64,
			},
		},
		{
			desc: "Frontend auth preserves `Authorization` and creates `X-Endpoint-API-UserInfo`." +
				"Backend auth then modifies `Authorization` and creates `X-Forwarded-Authorization`.",
			method: "GET",
			path:   "/bearertoken/constant/0",
			headers: map[string]string{
				"Authorization": "Bearer " + testdata.Es256Token,
			},
			wantHeaders: map[string]string{
				"Authorization":             "Bearer ya29.BackendAuthToken",
				"X-Endpoint-API-UserInfo":   testdata.Es256TokenPayloadBase64,
				"X-Forwarded-Authorization": "Bearer " + testdata.Es256Token,
			},
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		resp, err := echo_client.DoWithHeaders(url, tc.method, "", tc.headers)

		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		gotResp := string(resp)
		for wantKey, wantValue := range tc.wantHeaders {
			wantHeader := fmt.Sprintf(`"%v": "%v"`, wantKey, wantValue)
			if !strings.Contains(gotResp, wantHeader) {
				t.Fatalf("Test Desc(%s) failed,\n  got: %v\n want: %v", tc.desc, gotResp, tc.wantHeaders)
			}
		}
	}
}
