// Copyright 2022 Google LLC
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

package backend_circuit_breaker_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

const (
	// A test plan to run 2 echo tests in parallel, each with a delay of 2 seconds.
	testPlan = `
plans {
	parallel {
		test_count: 2
		parallel_limit: 2
		subtests {
			weight: 1
			echo {
				request {
					text: "Hello, world!"
					response_delay: 2
				}
				call_config {
					api_key: "this-is-an-api-key"
					auth_token: "this-is-auth-token"
				}
			}
		}
	}
}`

	// expected error message from envoy
	overflowError = "upstream connect error or disconnect/reset before headers. reset reason: overflow"
)

func TestBackendCircuitBreaker(t *testing.T) {
	testData := []struct {
		desc      string
		extraFlag string
		wantError bool
	}{
		{
			desc: "succeed with default the backend_cluster_max_requests flag.",
		},
		{
			desc:      "overflow with custom flag backend_cluster_max_requests=1",
			extraFlag: "--backend_cluster_maximum_requests=1",
			wantError: true,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			args := utils.CommonArgs()
			s := env.NewTestEnv(platform.TestBackendCircuitBreaker, platform.GrpcEchoSidecar)

			if tc.extraFlag != "" {
				args = append(args, tc.extraFlag)
			}

			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			defer s.TearDown(t)

			result, err := client.RunGRPCEchoTest(testPlan, s.Ports().ListenerPort)
			if err == nil {
				if tc.wantError {
					t.Errorf("expected error, but not error.")
				}
				return
			}

			if !tc.wantError {
				t.Errorf("fail, error during running test: %v", err)
			} else if !strings.Contains(result, overflowError) {
				t.Errorf("diff error message, got: %v, expect: %s", result, overflowError)
			}
		})
	}
}
