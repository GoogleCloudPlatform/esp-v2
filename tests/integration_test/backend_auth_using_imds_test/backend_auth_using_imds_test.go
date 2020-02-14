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

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

var testBackendAuthArgs = []string{
	"--service_config_id=test-config-id",
	"--backend_protocol=http",
	"--rollout_strategy=fixed",
	"--backend_dns_lookup_family=v4only",
	"--suppress_envoy_headers",
}

func NewBackendAuthTestEnv(port uint16) *env.TestEnv {
	s := env.NewTestEnv(port, platform.EchoRemote)
	return s
}

func TestBackendAuthWithImdsIdToken(t *testing.T) {
	s := NewBackendAuthTestEnv(comp.TestBackendAuthWithImdsIdToken)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/append":   "ya29.append",
		}, 0)

	defer s.TearDown()
	if err := s.Setup(testBackendAuthArgs); err != nil {
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
		if !utils.JsonEqual(gotResp, tc.wantResp) {
			t.Errorf("Test Desc(%s): want: %s, got: %s", tc.desc, tc.wantResp, gotResp)
		}
	}
}

func TestBackendAuthWithImdsIdTokenRetries(t *testing.T) {
	s := NewBackendAuthTestEnv(comp.TestBackendAuthWithImdsIdTokenRetries)

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
			wantInitialErr: `500 Internal Server Error, missing tokens`,
			wantFinalResp:  `{"Authorization": "Bearer ya29.constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow deferring in loop.
		func() {
			s.OverrideMockMetadata(
				map[string]string{
					util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
				}, tc.wantNumFails)

			defer s.TearDown()
			if err := s.Setup(testBackendAuthArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)

			// The first call should fail since IMDS is responding with failures.
			_, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err == nil {
				t.Fatalf("Test Desc(%s): expected failure while IAM is unhealthy", tc.desc)
			}
			if !strings.Contains(err.Error(), tc.wantInitialErr) {
				t.Fatalf("Test Desc(%s): expected failure (%v), got failure (%v)", tc.desc, tc.wantInitialErr, err)
			}

			// Sleep enough time for IMDS to become healthy. This depends on the retry timer in TokenSubscriber.
			time.Sleep(time.Duration(2*tc.wantNumFails) * time.Second)

			// The second request should work.
			resp, err := client.DoWithHeaders(url, tc.method, "", nil)
			if err != nil {
				t.Fatalf("Test Desc(%s): %v", tc.desc, err)
			}

			gotResp := string(resp)
			if !utils.JsonEqual(gotResp, tc.wantFinalResp) {
				t.Errorf("Test Desc(%s): want: %s, got: %s", tc.desc, tc.wantFinalResp, gotResp)
			}
		}()
	}
}

func TestBackendAuthWithImdsIdTokenWhileAllowCors(t *testing.T) {
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsOrigin := "http://cloud.google.com"

	s := NewBackendAuthTestEnv(comp.TestBackendAuthWithImdsIdTokenWhileAllowCors)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/append":   "ya29.append",
		}, 0)
	s.SetAllowCors()

	defer s.TearDown()
	if err := s.Setup(testBackendAuthArgs); err != nil {
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
