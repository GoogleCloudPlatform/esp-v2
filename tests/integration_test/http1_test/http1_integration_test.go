// Copyright 2019 Google Cloud Platform Proxy Authors
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

package http1_test

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

	configID := "test-config-id"

	args := []string{"--service_config_id=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestHttp1Basic, "echo")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc     string
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

	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestHttp1JWT, "echo")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

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
			wantResp:   `{"exp":4698318356,"iat":1544718356,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
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
