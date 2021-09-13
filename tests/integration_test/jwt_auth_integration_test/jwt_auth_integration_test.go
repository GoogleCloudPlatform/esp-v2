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

package jwt_auth_integration_test

import (
	"encoding/json"
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
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestAsymmetricKeys(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestAsymmetricKeys, platform.GrpcBookstoreSidecar)
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
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
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
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwks remote fetch is failed"}`,
		},
		{
			desc:           "Failed, provider not existing",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeNonexistJwksProviderToken,
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwks remote fetch is failed"}`,
		},
		{
			desc:           "Succeeded, using OpenID Connect Discovery",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeOpenIDToken,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:           "Failed, the provider found by OpenID Connect Discovery providing invalid jwks",
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
		t.Run(tc.desc, func(t *testing.T) {
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
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
		})
	}
}

func TestAuthAllowMissing(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestAuthAllowMissing, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector:               "endpoints.examples.bookstore.Bookstore.ListShelves",
				AllowWithoutCredential: true,
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
					{
						ProviderId: testdata.InvalidProvider,
						Audiences:  "bookstore_test_client.cloud.goog",
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
			desc:           "Succeeded with allow_missing. no JWT passed in.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
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
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwks remote fetch is failed"}`,
		},
		{
			// Ref: https://groups.google.com/g/envoy-announce/c/VkqM-5MlUeY
			desc:           "Failed, token with wrong issuer is rejected, even with allow missing.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.Rs256Token,
			wantError:      `401 Unauthorized, {"code":401,"message":"Jwt issuer is not configured"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
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
		})
	}
}

// Tests that config translation will fail when the OpenID Connect Discovery protocol is not followed.
func TestInvalidOpenIDConnectDiscovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc        string
		providerId  string
		configArgs  []string
		expectedErr string
	}{
		{
			desc:        "Fail with provider with invalid response.",
			providerId:  testdata.OpenIdInvalidProvider,
			configArgs:  utils.CommonArgs(),
			expectedErr: "health check response was not healthy",
		},
		{
			desc:        "Fail with provider that does not exist.",
			providerId:  testdata.OpenIdNonexistentProvider,
			configArgs:  utils.CommonArgs(),
			expectedErr: "health check response was not healthy",
		},
		{
			desc:       "Fail when OpenID Connect Discovery is disabled.",
			providerId: testdata.OpenIdInvalidProvider,
			configArgs: append([]string{
				"--disable_oidc_discovery",
			}, utils.CommonArgs()...),
			expectedErr: "health check response was not healthy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestInvalidOpenIDConnectDiscovery, platform.GrpcBookstoreSidecar)
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

			err := s.Setup(tc.configArgs)

			// LIFO ordering. Disable health checks before teardown, we expect a failure.
			defer s.TearDown(t)
			defer s.SkipHealthChecks()

			if err == nil {
				t.Errorf("failed, expected error, got no err")
			} else if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("failed, expected err: %v, got err: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestFrontendAndBackendAuthHeaders(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc                             string
		method                           string
		path                             string
		enableJwtPadForwardPayloadHeader bool
		headers                          map[string]string
		wantHeaders                      map[string]string
	}{
		{
			desc: "Frontend auth preserves `Authorization` and overrides `X-Endpoint-API-UserInfo`." +
				"Backend auth is disabled, so no further header modifications.",
			method: "GET",
			path:   "/disableauthsettotrue/constant/disableauthsettotrue",
			headers: map[string]string{
				"Authorization":           "Bearer " + testdata.Es256Token,
				"X-Endpoint-API-UserInfo": "untrusted-payload",
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
		{
			desc:                             "Not pad jwt authn X-Endpoint-API-UserInfo by default",
			method:                           "GET",
			path:                             "/disableauthsettotrue/constant/disableauthsettotrue",
			enableJwtPadForwardPayloadHeader: false,
			headers: map[string]string{
				"Authorization": "Bearer " + testdata.FakeCloudTokenSingleAudience3,
			},
			wantHeaders: map[string]string{
				"X-Endpoint-API-UserInfo": testdata.FakeCloudTokenSingleAudience3Payload,
			},
		},
		{
			desc:                             "Pad jwt authn X-Endpoint-API-UserInfo",
			method:                           "GET",
			path:                             "/disableauthsettotrue/constant/disableauthsettotrue",
			enableJwtPadForwardPayloadHeader: true,
			headers: map[string]string{
				"Authorization": "Bearer " + testdata.FakeCloudTokenSingleAudience3,
			},
			wantHeaders: map[string]string{
				"X-Endpoint-API-UserInfo": testdata.FakeCloudTokenSingleAudience3Payload + "==",
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestFrontendAndBackendAuthHeaders, platform.EchoRemote)
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
							{
								ProviderId: testdata.GoogleServiceAccountProvider,
								Audiences:  "need-pad",
							},
						},
					},
				},
			})
			s.OverrideMockMetadata(
				map[string]string{
					fmt.Sprintf("%v?format=standard&audience=https://%v/bearertoken/constant", util.IdentityTokenPath, platform.GetLoopbackAddress()): "ya29.BackendAuthToken",
				}, 0)

			defer s.TearDown(t)
			flags := utils.CommonArgs()
			if tc.enableJwtPadForwardPayloadHeader {
				flags = append(flags, "--jwt_pad_forward_payload_header")
			}

			if err := s.Setup(flags); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, tc.path)
			resp, err := echo_client.DoWithHeaders(url, tc.method, "", tc.headers)

			if err != nil {
				t.Fatalf("%v", err)
			}

			var sec map[string]interface{}
			if err = json.Unmarshal(resp, &sec); err != nil {
				t.Fatalf("fail to parse response into json")
			}
			for wantKey, wantValue := range tc.wantHeaders {
				wantHeader := fmt.Sprintf(`"%v": "%v"`, wantKey, wantValue)
				gotHeaderVal, ok := sec[wantKey]
				gotHeaderValStr := fmt.Sprintf("%v", gotHeaderVal)
				if !ok || wantValue != gotHeaderValStr {
					t.Fatalf("failed on header %s,\n  got: %s\n want: %v", wantKey, gotHeaderValStr, wantHeader)
				}
			}
		})
	}
}
