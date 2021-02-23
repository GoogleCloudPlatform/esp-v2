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

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
)

type TranscodingTestType struct {
	desc               string
	clientProtocol     string
	httpMethod         string
	method             string
	token              string
	headers            map[string][]string
	bodyBytes          []byte
	wantResp           string
	wantErr            string
	wantGRPCWebTrailer client.GRPCWebTrailer
}

func TestTranscodingBackendUnavailableError(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestTranscodingBackendUnavailableError, platform.GrpcBookstoreSidecar)

	defer s.TearDown(t)
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
		wantErr:        `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: connection failure"}`,
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
	t.Parallel()

	serviceName := "bookstore-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestTranscodingErrors, platform.GrpcBookstoreSidecar)
	defer s.TearDown(t)
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
			wantErr:        "404 Not Found",
		},
		{
			desc:           "failed with 400, no braces json",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`NO_BRACES_JSON`),
			wantErr: `400 Bad Request, {"code":400,"message":"Unexpected token.
NO_BRACES_JSON
^"}`,
		},
		{
			desc:           "failed with 400, not closed json",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme" : "Children"`),
			wantErr: `400 Bad Request, {"code":400,"message":"Unexpected end of string. Expected , or } after key:value pair.

^"}`,
		},
		{
			desc:           "failed with 400, no colon",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme"  "Children"}`),
			wantErr: `400 Bad Request, {"code":400,"message":"Expected : between key:value pair.
{"theme"  "Children"}
          ^"}`,
		},
		{
			desc:           "failed with 400, extra chars",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/0/books?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			bodyBytes:      []byte(`{"theme" : "Children"}EXTRA`),
			wantErr: `{"code":400,"message":"Parsing terminated before end of input.
theme" : "Children"}EXTRA
                    ^"}`,
		},
		{
			// TODO(b/177252401): When invalid query param is passed, the error is the incorrect type.
			desc:           "Failed due to bad query parameter. Error returned is the wrong type.",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key&badQueryParam=test",
			wantErr:        `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: remote reset"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, tc.token, tc.bodyBytes)

			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
			} else {
				if !strings.Contains(resp, tc.wantResp) {
					t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
				}
			}
		})
	}
}
