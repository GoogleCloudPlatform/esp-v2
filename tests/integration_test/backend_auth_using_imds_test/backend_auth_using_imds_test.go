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

package backend_auth_using_imds_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestBackendAuthWithImdsIdToken(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestBackendAuthWithImdsIdToken, platform.EchoRemote)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenPath + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
			util.IdentityTokenPath + "?format=standard&audience=https://localhost/bearertoken/append":   "ya29.append",
		}, 0)

	defer s.TearDown(t)
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
			wantResp: `{"Authorization": "Bearer ya29.constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:     "Add Bearer token for APPEND_PATH_TO_ADDRESS backend that requires JWT token",
			method:   "GET",
			path:     "/bearertoken/append?key=api-key",
			wantResp: `{"Authorization": "Bearer ya29.append", "RequestURI": "/bearertoken/append?key=api-key"}`,
		},
		{
			desc:     "Do not reject backend that doesn't require JWT token",
			method:   "POST",
			path:     "/echo?key=api-key",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
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

func TestBackendAuthWithImdsIdTokenRetries(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc           string
		method         string
		path           string
		confArgs       []string
		wantNumFails   int
		wantInitialErr string
		wantFinalResp  string
	}{
		{
			desc:           "By default, envoy does not start until token is successfully fetched.",
			method:         "GET",
			path:           "/bearertoken/constant/42",
			confArgs:       utils.CommonArgs(),
			wantNumFails:   5,
			wantInitialErr: `connect: connection refused`,
			wantFinalResp:  `{"Authorization": "Bearer ya29.constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:   "With modified error behavior, envoy starts but returns errors before token is successfully fetched.",
			method: "GET",
			path:   "/bearertoken/constant/42",
			confArgs: append([]string{
				"--dependency_error_behavior=ALWAYS_INIT",
			}, utils.CommonArgs()...),
			wantNumFails:   5,
			wantInitialErr: `{"code":500,"message":"Token not found for audience: https://localhost/bearertoken/constant"}`,
			wantFinalResp:  `{"Authorization": "Bearer ya29.constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestBackendAuthWithImdsIdTokenRetries, platform.EchoRemote)
			// Health checks prevent envoy from starting up due to bad responses from IMDS for tokens.
			s.SkipHealthChecks()
			s.OverrideMockMetadata(
				map[string]string{
					util.IdentityTokenPath + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
				}, tc.wantNumFails)

			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			// Sleep some time to allow Envoy to startup.
			time.Sleep(2 * time.Second)

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)

			// The first call should fail since IMDS is responding with failures.
			_, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err == nil {
				t.Fatalf("Test Desc(%s): expected failure while IAM is unhealthy", tc.desc)
			}
			if !strings.Contains(err.Error(), tc.wantInitialErr) {
				t.Fatalf("Test Desc(%s): expected failure (%v), got failure (%v)", tc.desc, tc.wantInitialErr, err)
			}

			// Sleep enough time for IMDS to become healthy. This depends on the retry timer in TokenSubscriber (with some slack).
			time.Sleep(time.Duration(tc.wantNumFails) * time.Second * 4)

			// The second request should work.
			resp, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err != nil {
				t.Fatalf("Test Desc(%s): %v", tc.desc, err)
			}

			gotResp := string(resp)
			if err := util.JsonEqual(tc.wantFinalResp, gotResp); err != nil {
				t.Errorf("Test Desc(%s) fails: \n %s", tc.desc, err)
			}
		})
	}
}

func TestBackendAuthWithImdsIdTokenWhileAllowCors(t *testing.T) {
	t.Parallel()

	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsOrigin := "http://cloud.google.com"

	s := env.NewTestEnv(platform.TestBackendAuthWithImdsIdTokenWhileAllowCors, platform.EchoRemote)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenPath + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
			util.IdentityTokenPath + "?format=standard&audience=https://localhost/bearertoken/append":   "ya29.append",
		}, 0)
	s.SetAllowCors()

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc       string
		path       string
		message    string
		wantHeader string
	}{
		{
			desc:       "Add Bearer token for CONSTANT_ADDRESS backend that requires JWT token",
			path:       "/bearertoken/constant/42",
			wantHeader: `X-Token: Bearer ya29.constant`,
		},
		{
			desc:       "Add Bearer token for APPEND_PATH_TO_ADDRESS backend that requires JWT token",
			path:       "/bearertoken/append?key=api-key",
			wantHeader: `X-Token: Bearer ya29.append`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		respHeader, err := client.DoCorsPreflightRequest(url, corsOrigin, corsRequestMethod, corsRequestHeader, "")
		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}
		if gotHeader := respHeader.Get("Access-Control-Expose-Headers"); gotHeader != tc.wantHeader {
			t.Errorf("Test Desc(%s) expected: %s, got: %s", tc.desc, tc.wantHeader, gotHeader)
		}
	}
}
