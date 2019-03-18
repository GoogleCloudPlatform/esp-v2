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

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

const (
	echo = "hello"
)

func TestHttp1Basic(t *testing.T) {
	serviceName := "test-echo"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestHttp1Basic, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo")
		resp, err := client.DoPost(url, echo)
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
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestHttp1JWT, "echo", []string{"google_jwt"})
	if err := s.Setup(args); err != nil {
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
			token:      testdata.FakeCloudToken,
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
			token:      testdata.FakeCloudTokenSingleAudience2,
			wantResp:   `{"id": "anonymous"}`,
		},
		{
			desc:        "Fail, with valid JWT token, without allowed audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudToken,
			wantedError: "401 Unauthorized",
		},
		{
			desc:        "Fail, with valid JWT token, with incorrect audience",
			httpMethod:  "GET",
			httpPath:    "/auth/info/auth0",
			token:       testdata.FakeCloudTokenSingleAudience1,
			wantedError: "401 Unauthorized",
		},
	}
	for _, tc := range testData {
		host := fmt.Sprintf("http://localhost:%v", s.Ports().ListenerPort)
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

func TestHttp1BackendAuth(t *testing.T) {
	serviceName := "test-echo"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=http1", "--rollout_strategy=fixed",
		"--enable_backend_routing"}

	s := env.NewTestEnv(comp.TestHttp1BackendAuth, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := []struct {
		desc       string
		httpMethod string
		httpPath   string
		message    string
		wantResp   string
	}{
		{
			desc:       "Add Bearer token for backend that requires JWT token",
			httpMethod: "GET",
			httpPath:   "/bearertoken",
			wantResp:   `{"Authorization": "Bearer ya29.new"}`,
		},
		{
			desc:       "Do not reject backend that doesn't require JWT token",
			httpMethod: "POST",
			httpPath:   "/echo",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.httpPath)
		var resp []byte
		var err error
		switch tc.httpMethod {
		case "GET":
			resp, err = client.DoGet(url)
		case "POST":
			resp, err = client.DoPost(url, tc.message)
		default:
			t.Fatalf("Test Desc(%s): unsupported HTTP Method %s", tc.desc, tc.httpPath)
		}

		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test Desc(%s): expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}
	}
}
