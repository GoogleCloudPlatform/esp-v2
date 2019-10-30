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

package backend_auth_test

import (
	"fmt"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

var testBackendAuthArgs = []string{
	"--service_config_id=test-config-id",
	"--backend_protocol=http1",
	"--rollout_strategy=fixed",
	"--enable_backend_routing",
	"--backend_dns_lookup_family=v4only",
	"--suppress_envoy_headers",
}

func NewBackendAuthTestEnv(port uint16) *env.TestEnv {
	s := env.NewTestEnv(port, "echoForDynamicRouting")
	s.EnableDynamicRoutingBackend()
	return s
}

func TestBackendAuthWithImdsIdToken(t *testing.T) {
	s := NewBackendAuthTestEnv(comp.TestBackendAuthWithImdsIdToken)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.constant",
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/append":   "ya29.append",
		})

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

func TestBackendAuthWithIamIdToken(t *testing.T) {
	s := NewBackendAuthTestEnv(comp.TestBackendAuthWithIamIdToken)
	serviceAccount := "fakeServiceAccount@google.com"

	s.SetIamServiceAccount(serviceAccount)
	s.SetIamResps(
		map[string]string{
			fmt.Sprintf("%s?audience=https://localhost/bearertoken/constant", util.IamIdentityTokenSuffix(serviceAccount)): `{"token":  "id-token-for-constant"}`,
			fmt.Sprintf("%s?audience=https://localhost/bearertoken/append", util.IamIdentityTokenSuffix(serviceAccount)):   `{"token":  "id-token-for-append"}`,
		})

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
			wantResp: `{"Authorization": "Bearer id-token-for-constant", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:     "Add Bearer token for APPEND_PATH_TO_ADDRESS backend that requires JWT token",
			method:   "GET",
			path:     "/bearertoken/append?key=api-key",
			wantResp: `{"Authorization": "Bearer id-token-for-append", "RequestURI": "/bearertoken/append?key=api-key"}`,
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
