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

package managed_service_config_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore_grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestManagedServiceConfig(t *testing.T) {

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=managed", "--check_rollout_interval=1s"}
	s := env.NewTestEnv(comp.TestManagedServiceConfig, "bookstore")
	s.SetEnvoyDrainTimeInSec(1)

	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				Requirements: []*confpb.AuthRequirement{
					{
						ProviderId: testdata.TestAuthProvider,
						Audiences:  "ok_audience",
					},
				},
			},
		},
	})
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		queryInToken       bool
		token              string
		headers            map[string][]string
		wantResp           string
		wantError          string
		wantGRPCWebError   string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}{
		{
			desc:           "Fail, since the request doesn't have token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantError:      "401 Unauthorized, Jwt is missing",
		},
		{
			desc:           "Success, the new service config doesn't require authz for this API",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves?key=api-key",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
	}

	for idx, tc := range tests {
		// Remove the authentication in service config and wait envoy to update.
		if idx == 1 {
			s.OverrideAuthentication(&confpb.Authentication{})
			time.Sleep(time.Second * 5)
		}

		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		var resp string
		var err error
		if tc.queryInToken {
			resp, err = client.MakeTokenInQueryCall(addr, tc.httpMethod, tc.method, tc.token)
		} else {
			resp, err = client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, tc.headers)
		}

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			}
		}
	}
}
