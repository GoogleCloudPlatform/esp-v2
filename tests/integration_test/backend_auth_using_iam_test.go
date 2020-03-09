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
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestBackendAuthWithIamIdToken(t *testing.T) {
	t.Parallel()

	s := env.NewBackendAuthTestEnv(comp.TestBackendAuthWithIamIdToken)
	serviceAccount := "fakeServiceAccount@google.com"

	s.SetBackendAuthIamServiceAccount(serviceAccount)
	s.SetIamResps(
		map[string]string{
			fmt.Sprintf("%s?audience=https://localhost/bearertoken/constant", util.IamIdentityTokenSuffix(serviceAccount)): `{"token":  "id-token-for-constant"}`,
			fmt.Sprintf("%s?audience=https://localhost/bearertoken/append", util.IamIdentityTokenSuffix(serviceAccount)):   `{"token":  "id-token-for-append"}`,
		}, 0)

	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc     string
		method   string
		path     string
		message  string
		wantResp string
	}{
		{
			desc:     "Add Bearer token for CONSTANT_ADDRESS backend that requires JWT token",
			method:   "GET",
			path:     "/bearertoken/constant/42",
			wantResp: `{"Authorization": "Bearer id-token-for-constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:     "Add Bearer token for APPEND_PATH_TO_ADDRESS backend that requires JWT token",
			method:   "GET",
			path:     "/bearertoken/append?key=api-key",
			wantResp: `{"Authorization": "Bearer id-token-for-append", "RequestURI": "/bearertoken/append?key=api-key"}`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		resp, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		gotResp := string(resp)
		if err := util.JsonEqual(tc.wantResp, gotResp); err != nil {
			t.Errorf("Test Desc(%s) fails: \n %s", tc.desc, err)
		}
	}
}

func TestBackendAuthWithIamIdTokenRetries(t *testing.T) {
	t.Parallel()
	s := env.NewBackendAuthTestEnv(comp.TestBackendAuthWithIamIdTokenRetries)
	serviceAccount := "fakeServiceAccount@google.com"
	s.SetBackendAuthIamServiceAccount(serviceAccount)

	// Health checks prevent envoy from starting up due to bad responses from IAM for tokens.
	s.SkipHealthChecks()

	testData := []struct {
		desc           string
		method         string
		path           string
		wantNumFails   int
		wantInitialErr string
		wantFinalResp  string
	}{
		{
			desc:           "Add Bearer token for CONSTANT_ADDRESS backend that requires JWT token",
			method:         "GET",
			path:           "/bearertoken/constant/42",
			wantNumFails:   5,
			wantInitialErr: `connect: connection refused`,
			wantFinalResp:  `{"Authorization": "Bearer id-token-for-constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow deferring in loop.
		func() {
			s.SetIamResps(
				map[string]string{
					fmt.Sprintf("%s?audience=https://localhost/bearertoken/constant", util.IamIdentityTokenSuffix(serviceAccount)): `{"token":  "id-token-for-constant"}`,
				}, tc.wantNumFails)

			defer s.TearDown()
			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)

			// The first call should fail since IAM is responding with failures.
			_, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err == nil {
				t.Fatalf("Test Desc(%s): expected failure while IAM is unhealthy", tc.desc)
			}
			if !strings.Contains(err.Error(), tc.wantInitialErr) {
				t.Fatalf("Test Desc(%s): expected failure (%v), got failure (%v)", tc.desc, tc.wantInitialErr, err)
			}

			// Sleep enough time for IAM to become healthy. This depends on the retry timer in TokenSubscriber.
			time.Sleep(time.Duration(3*tc.wantNumFails) * time.Second)

			// The second request should work.
			resp, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err != nil {
				t.Fatalf("Test Desc(%s): %v", tc.desc, err)
			}

			gotResp := string(resp)
			if err := util.JsonEqual(tc.wantFinalResp, gotResp); err != nil {
				t.Errorf("Test Desc(%s) fails: \n %s", tc.desc, err)
			}
		}()
	}
}

func TestBackendAuthUsingIamIdTokenWithDelegates(t *testing.T) {
	t.Parallel()

	s := env.NewBackendAuthTestEnv(comp.TestBackendAuthUsingIamIdTokenWithDelegates)
	serviceAccount := "fakeServiceAccount@google.com"

	s.SetBackendAuthIamServiceAccount(serviceAccount)
	s.SetBackendAuthIamDelegates("delegate_foo,delegate_bar,delegate_baz")

	s.SetIamResps(
		map[string]string{
			fmt.Sprintf("/v1/projects/-/serviceAccounts/%s:generateIdToken?audience=https://localhost/bearertoken/constant", serviceAccount): `{"token":  "id-token-for-constant"}`,
		}, 0)

	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc            string
		method          string
		path            string
		message         string
		wantReqBody     string
		wantIamReqToken string
		wantIamReqBody  string
		wantResp        string
	}{
		{
			desc:            "Use delegates when fetching identity token from IAM server",
			method:          "GET",
			path:            "/bearertoken/constant/42",
			wantIamReqToken: "Bearer ya29.new",
			wantIamReqBody:  `{"includeEmail":true,"delegates":["projects/-/serviceAccounts/delegate_foo","projects/-/serviceAccounts/delegate_bar","projects/-/serviceAccounts/delegate_baz"]}`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		_, err := client.DoWithHeaders(url, tc.method, tc.message, nil)
		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		if iamReqToken, err := s.MockIamServer.GetRequestToken(); err != nil {
			t.Errorf("Test Desc(%s): failed to get request header", tc.desc)
		} else if tc.wantIamReqToken != iamReqToken {
			t.Errorf("Test Desc(%s), different iam request token, wanted: %s, got: %s", tc.desc, tc.wantIamReqToken, iamReqToken)
		}

		if iamReqBody, err := s.MockIamServer.GetRequestBody(); err != nil {
			t.Errorf("Test Desc(%s): failed to get request body", tc.desc)
		} else if tc.wantIamReqBody != "" {
			if err := util.JsonEqual(tc.wantIamReqBody, iamReqBody); err != nil {
				t.Errorf("Test Desc(%s), different iam request body, \n %v", tc.desc, err)
			}
		}
	}
}
