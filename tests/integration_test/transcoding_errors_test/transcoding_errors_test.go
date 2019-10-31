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

package transcoding_errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/api-proxy/tests/env"
	"github.com/GoogleCloudPlatform/api-proxy/tests/env/testdata"

	comp "github.com/GoogleCloudPlatform/api-proxy/tests/env/components"
)

type TranscodingTestType struct {
	desc               string
	clientProtocol     string
	httpMethod         string
	method             string
	noBackend          bool
	token              string
	headers            map[string][]string
	bodyBytes          []byte
	wantResp           string
	wantErr            string
	wantGRPCWebTrailer client.GRPCWebTrailer
}

func TestTranscodingServiceUnavailableError(t *testing.T) {

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestTranscodingServiceUnavailableError, "bookstore")

	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	if err := s.StopBackendServer(); err != nil {
		t.Fatalf("fail to shut down backend, %v", err)
	}
	tc := TranscodingTestType{
		desc:           "failed with 503, no backend",
		clientProtocol: "http",
		httpMethod:     "GET",
		method:         "/v1/shelves/200/books/2001?key=api-key",
		noBackend:      true,
		wantErr:        "503 Service Unavailable",
	}

	addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
	resp, err := client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)

	if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
		t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
	} else {
		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}
	}
}

func TestTranscodingErrors(t *testing.T) {

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestTranscodingErrors, "bookstore")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []TranscodingTestType{
		{
			desc:           "failed with 404, no this book",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/200/books/2002?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			noBackend:      true,
			wantErr:        "404 Not Found",
		},
		{
			desc:           "failed with 400, no braces json",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`NO_BRACES_JSON`),
			noBackend:      true,
			wantErr:        "400 Bad Request, Unexpected token",
		},
		{
			desc:           "failed with 400, not closed json",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme" : "Children"`),
			noBackend:      true,
			wantErr:        "400 Bad Request, Unexpected end of string. Expected , or } after key:value pair",
		},
		{
			desc:           "failed with 400, no colon",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme"  "Children"}`),
			noBackend:      true,
			wantErr:        "400 Bad Request, Expected : between key:value pair",
		},
		{
			desc:           "failed with 400, extra chars",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme" : "Children"}EXTRA`),
			noBackend:      true,
			wantErr:        "400 Bad Request, Parsing terminated before end of input",
		},
	}
	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, tc.token, tc.bodyBytes)

		if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
