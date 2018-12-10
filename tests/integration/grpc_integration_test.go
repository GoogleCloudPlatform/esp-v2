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

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
)

const (
	addr = "127.0.0.1:8080"
)

func TestGrpc(t *testing.T) {
	serviceName := "bookstore-service"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=grpc"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc           string
		clientProtocol string
		method         string
		wantResp       string
	}{
		{
			desc:           "gRPC client calling gRPC backend",
			clientProtocol: "grpc",
			method:         "GetShelf",
			wantResp:       `{"theme":"Unknown Book"}`,
		},
		{
			desc:           "Http client calling gRPC backend",
			clientProtocol: "http",
			method:         "/v1/shelves/125",
			wantResp:       `{"id":"125","theme":"Unknown Book"}`,
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.method, "")
		if err != nil {
			t.Errorf("failed to run test: %s", err)
		}

		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}
	}
}

func TestGrpcJwt(t *testing.T) {
	serviceName := "bookstore-service"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=grpc"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc           string
		clientProtocol string
		method         string
		token          string
		wantResp       string
		wantedError    string
	}{
		{
			desc:           "gPRC client calling gPRC backend, with valid JWT token",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeGoodToken,
			wantResp:       `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
		},
		{
			desc:           "Http client calling gPRC backend, with valid JWT token",
			clientProtocol: "http",
			method:         "/v1/shelves",
			token:          testdata.FakeGoodToken,
			wantResp:       `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
		},
		{
			desc:           "gPRC client calling gPRC backend, without valid JWT token",
			clientProtocol: "grpc",
			method:         "ListShelves",
			wantedError:    "code = Unauthenticated desc = Jwt is missing",
		},
		{
			desc:           "Http client calling gPRC backend, without invalid JWT token",
			clientProtocol: "http",
			method:         "/v1/shelves",
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "gPRC client calling gPRC backend, with bad JWT token",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeBadToken,
			wantedError:    "code = Unauthenticated desc = Jwt issuer is not configured",
		},
		{
			desc:           "Http client calling gPRC backend, with bad JWT token",
			clientProtocol: "http",
			method:         "/v1/shelves",
			token:          testdata.FakeBadToken,
			wantedError:    "401 Unauthorized",
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.method, tc.token)

		if tc.wantedError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantedError)) {
			t.Errorf("Test (%s): failed, expected err: %s, got: %s", tc.desc, tc.wantedError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
