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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"github.com/golang/glog"
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
			method:   "http://localhost:8080/echo",
			wantResp: `{"message":"hello"}`,
		},
		{
			desc:     "failed, missing Jwt",
			method:   "http://localhost:8080/auth/info/googlejwt",
			wantResp: `Jwt is missing`,
		},
	}
	for _, tc := range testData {
		var resp *http.Response
		var err error
		resp, err = doEcho(tc.method)
		if err != nil {
			glog.Fatal(err)
		}
		out, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Fatal(err)
		}

		if !strings.Contains(string(out), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(out))
		}
	}
}

func doEcho(method string) (*http.Response, error) {
	msg := map[string]string{
		"message": "hello",
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(msg); err != nil {
		return nil, err
	}
	return http.Post(method, "application/json", &buf)
}

func TestHttp1Jwt(t *testing.T) {
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

	var resp *http.Response
	var err error
	resp, err = doJWT()
	if err != nil {
		glog.Fatal(err)
	}
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatal(err)
	}
	want := `{"id": "anonymous"}`

	if !strings.Contains(string(out), want) {
		t.Errorf("expected: %s, got: %s", want, string(out))
	}
}

func doJWT() (*http.Response, error) {
	req, _ := http.NewRequest("GET", "http://localhost:8080/auth/info/googlejwt", nil)
	req.Header.Add("Authorization", "Bearer "+testdata.FakeGoodToken)
	return http.DefaultClient.Do(req)
}
