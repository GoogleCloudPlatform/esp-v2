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
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
)

const (
	echo = "hello"
	host = "http://localhost:8080"
)

func TestHttp1Basic(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=http1"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := []struct {
		desc     string
		method   string
		wantResp string
	}{
		{
			desc:     "succeed, no Jwt required",
			wantResp: `{"message":"hello"}`,
		},
	}
	for _, tc := range testData {
		resp, err := client.DoEcho(host, echo, "")
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
		}
	}
}

func TestHttp1JWT(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=http1"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	time.Sleep(time.Duration(3 * time.Second))

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
			httpPath:   "/auth/info/googlejwt",
			token:      testdata.FakeGoodToken,
			wantResp:   `{"id": "anonymous"}`,
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
			token:      testdata.FakeGoodAdminToken,
			wantResp:   `{"id": "anonymous"}`,
		},
		{
			desc:        "Fail, with valid JWT token, without allowed audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeGoodToken,
			wantedError: "401 Unauthorized",
		},
		{
			desc:        "Fail, with valid JWT token, with incorrect audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeGoodTokenSingleAud,
			wantedError: "401 Unauthorized",
		},
	}
	for _, tc := range testData {
		resp, err := client.DoJWT(host, tc.httpMethod, tc.httpPath, "", "", tc.token)

		if tc.wantedError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantedError)) {
			t.Errorf("Test (%s): failed, expected err: %s, got: %s", tc.desc, tc.wantedError, err)
		} else {
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}
	}
}
