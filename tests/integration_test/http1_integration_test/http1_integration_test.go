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

package http1_integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

const (
	echo = "hello"
)

func TestHttp1Basic(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"

	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed", "--healthz=/healthz"}

	s := env.NewTestEnv(platform.TestHttp1Basic, platform.EchoSidecar)
	defer s.TearDown(t)

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc               string
		path               string
		method             string
		wantResp           string
		wantedError        string
		wantScRequestCount int
	}{
		{
			desc:               "succeed, no Jwt required",
			path:               "/echo",
			method:             "POST",
			wantResp:           `{"message":"hello"}`,
			wantScRequestCount: 2,
		},
		{
			desc:               "health check succeed",
			path:               "/healthz",
			method:             "GET",
			wantScRequestCount: 0,
		},
		{
			desc:               "health check fail",
			path:               "/healthcheck",
			method:             "GET",
			wantedError:        "404 Not Found",
			wantScRequestCount: 1,
		},
	}
	for _, tc := range testData {
		s.ServiceControlServer.ResetRequestCount()
		url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, tc.path)
		var resp []byte
		var err error
		if tc.method == "GET" {
			resp, err = client.DoGet(url)
		} else if tc.method == "POST" {
			resp, err = client.DoPost(fmt.Sprintf("%s?key=api-key", url), echo)
		} else {
			t.Fatal(fmt.Errorf("unexpected method"))
		}
		if tc.wantedError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantedError)) {
			t.Errorf("Test (%s): failed, expected err: %s, got: %s", tc.desc, tc.wantedError, err)
		} else if tc.wantedError == "" && err != nil {
			t.Errorf("Test (%s): got unexpected error: %s", tc.desc, resp)
		} else {
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}
		// Health Check should not check/report to ServiceControl.
		if err = s.ServiceControlServer.VerifyRequestCount(tc.wantScRequestCount); err != nil {
			t.Errorf("Test (%s): verify request count failed, got: %v", tc.desc, err)
		}
	}
}

func TestHttp1JWT(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--skip_service_control_filter=true", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestHttp1JWT, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc        string
		httpMethod  string
		httpPath    string
		token       string
		wantResp    string
		wantedError string
	}{
		{
			desc:       "Succeed, with valid JWT token",
			httpMethod: "GET",
			httpPath:   "/auth/info/auth0",
			token:      testdata.FakeCloudTokenMultiAudiences,
			wantResp:   `{"aud":["admin.cloud.goog","bookstore_test_client.cloud.goog"],"exp":4698318999,"iat":1544718999,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:        "Fail, with valid JWT token",
			httpMethod:  "GET",
			httpPath:    "/auth/info/googlejwt",
			token:       testdata.FakeBadToken,
			wantedError: "401 Unauthorized",
		},
		{
			desc:        "Fail, without valid JWT token",
			httpMethod:  "GET",
			httpPath:    "/auth/info/googlejwt",
			wantedError: "401 Unauthorized",
		},
		{
			desc:       "Succeed, with valid JWT token, with allowed audience",
			httpMethod: "GET",
			httpPath:   "/auth/info/auth0",
			token:      testdata.FakeCloudTokenSingleAudience2,
			wantResp:   `{"aud":"admin.cloud.goog","exp":4698318995,"iat":1544718995,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:        "Fail, with valid JWT token, without allowed audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudToken,
			wantedError: "403 Forbidden",
		},
		{
			desc:        "Fail, with valid JWT token, with incorrect audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudTokenSingleAudience1,
			wantedError: "403 Forbidden",
		},
	}
	for _, tc := range testData {
		host := fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
		resp, err := client.DoJWT(host, tc.httpMethod, tc.httpPath, "", "", tc.token)

		if tc.wantedError == "" && err != nil || tc.wantedError != "" && err == nil || err != nil && !strings.Contains(err.Error(), tc.wantedError) {
			t.Errorf("Test (%s): failed, expected err: %s, got: %s", tc.desc, tc.wantedError, err)
		} else {
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}
	}
}

func TestDisableJwtServiceNameCheckFlag(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configID := "test-config-id"

	// Add flag "--disable_jwt_audience_service_name_check"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--skip_service_control_filter=true", "--rollout_strategy=fixed",
		"--disable_jwt_audience_service_name_check"}

	s := env.NewTestEnv(platform.TestJWTDisabledAudCheck, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc        string
		httpMethod  string
		httpPath    string
		token       string
		wantResp    string
		wantedError string
	}{
		// Path "/auth/info/googlejwt" uses AuthRequirement that only has "provider_id",
		// Supposely, Jwt audience should check with "audiences" specified in AuthProvider.audiences
		// which is empty, the service_name "bookstore_test_client.cloud.goog" should be checked.
		// But since JwtAudienceServiceNameCheck is disabled, JWT audience is not checked.
		{
			desc:       `JWT audience is "admin.cloud.goog" and check against empty Provider.audiences.`,
			httpMethod: "GET",
			httpPath:   "/auth/info/googlejwt",
			token:      testdata.FakeCloudTokenSingleAudience2,
			wantResp:   `{"aud":"admin.cloud.goog","exp":4698318995,"iat":1544718995,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:       `JWT without audience and check against empty Provider.audiences.`,
			httpMethod: "GET",
			httpPath:   "/auth/info/googlejwt",
			token:      testdata.FakeCloudToken,
			wantResp:   `{"exp":4698318356,"iat":1544718356,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:       `JWT audience is "bookstore_test_client.cloud.goog" and check against empty Provider.audiences`,
			httpMethod: "GET",
			httpPath:   "/auth/info/googlejwt",
			token:      testdata.FakeCloudTokenSingleAudience1,
			wantResp:   `{"aud":"bookstore_test_client.cloud.goog","exp":4698318811,"iat":1544718811,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		// Path "/auth/info/auth0" uses AuthRequirement that has both "provider_id" and "audiences" is "admin.cloud.goog"
		// Jwt audience should be still checked against "admin.cloud.goog" even JwtAudienceServiceNameCheck is disabled.
		{
			desc:       `JWT audience is "admin.cloud.goog" and check against AuthRequirement.audiences "admin.cloud.goog"`,
			httpMethod: "GET",
			httpPath:   "/auth/info/auth0",
			token:      testdata.FakeCloudTokenSingleAudience2,
			wantResp:   `{"aud":"admin.cloud.goog","exp":4698318995,"iat":1544718995,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:        `JWT without audience and check against AuthRequirement.audiences "admin.cloud.goog"`,
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudToken,
			wantedError: "403 Forbidden",
		},
		{
			desc:        `JWT audience is "bookstore_test_client.cloud.goog" and check against AuthRequirement.audiences "admin.cloud.goog"`,
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudTokenSingleAudience1,
			wantedError: "403 Forbidden",
		},
	}
	for _, tc := range testData {
		host := fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
		resp, err := client.DoJWT(host, tc.httpMethod, tc.httpPath, "", "", tc.token)

		if tc.wantedError == "" && err != nil || tc.wantedError != "" && err == nil || err != nil && !strings.Contains(err.Error(), tc.wantedError) {
			t.Errorf("Test (%s): failed, expected err: %s, got: %s", tc.desc, tc.wantedError, err)
		} else {
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}
	}
}

func TestHttpHeaders(t *testing.T) {
	t.Parallel()
	testData := []struct {
		desc                 string
		method               string
		requestHeader        map[string]string
		underscoresInHeaders bool
		wantResp             string
		wantError            string
	}{
		{
			desc:                 "Allow HTTP headers with underscore when configured",
			underscoresInHeaders: true,
			wantResp:             `{"id":"100","theme":"Kids"}`,
			requestHeader: map[string]string{
				"X-API-KEY":  "key-3",
				"X_INTERNAL": "underscore-header",
			},
		},
		{
			desc: "Doesn't allow HTTP headers with underscore by default",
			requestHeader: map[string]string{
				"X-API-KEY":  "key-3",
				"X_INTERNAL": "underscore-header",
			},
			wantError: `400 Bad Request`,
		},
	}
	for _, tc := range testData {
		func() {
			s := env.NewTestEnv(platform.TestHttpHeaders, platform.GrpcBookstoreSidecar)
			defer s.TearDown(t)
			args := utils.CommonArgs()
			if tc.underscoresInHeaders {
				args = append(args, "--underscores_in_headers")
			}
			err := s.Setup(args)
			url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/v1/shelves/100")
			resp, err := client.DoWithHeaders(url, "GET", "", tc.requestHeader)
			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(resp), tc.wantResp) {
					t.Errorf("Test desc (%v) expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
				}
			} else if err == nil {
				t.Errorf("Test (%s): failed\nexpected: %v\ngot nil", tc.desc, tc.wantError)
			} else if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("Test (%s): failed\nexpected: %v\ngot: %v", tc.desc, tc.wantError, err)
			}
		}()
	}
}
