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

package grpc_deadline_test_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func elapsed(what string) func() {
	start := time.Now()
	return func() {
		glog.Infof("%s took %v\n", what, time.Since(start))
	}
}

// Tests the deadlines configured in backend rules for a gRPC remote backends.
func TestDeadlinesForGrpcDynamicRouting(t *testing.T) {
	args := []string{
		"--service_config_id=test-config-id",

		"--rollout_strategy=fixed",
		"--backend_dns_lookup_family=v4only",
		"--suppress_envoy_headers",
	}

	s := env.NewTestEnv(comp.TestDeadlinesForGrpcDynamicRouting, platform.GrpcEchoRemote)

	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc     string
		wantErr  string
		testPlan string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc: "Success after 5s due to user-configured response deadline being 10s",
			testPlan: `
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 5
    }
  }
}`,
		},
		{
			desc:    "Fail before 15s due to user-configured response deadline being 10s",
			wantErr: "upstream request timeout",
			testPlan: `
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 15
    }
  }
}`,
		},
		{
			desc: "Success after 20s because ESPv2 automatically disables response timeouts for streaming RPCs",
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 20
    }
    count: 1
  }
}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow efficient measuring of elapsed time.
		// Elapsed time is not checked in the test, it's just for debugging.
		func() {
			defer elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

			// For this client, `err` will always be "exit status 1" on failures.
			// Check for actual error in `resp` instead.
			resp, err := client.RunGRPCEchoTest(tc.testPlan, s.Ports().ListenerPort)

			if tc.wantErr == "" && err != nil {
				t.Errorf("Test (%v): Error during running test: want no err, got err (%v)", tc.desc, resp)
			}

			if tc.wantErr != "" && err == nil {
				t.Errorf("Test (%v): Error during running test: got no err, want err (%v)", tc.desc, tc.wantErr)
			}

			if err != nil && !strings.Contains(resp, tc.wantErr) {
				t.Errorf("Test (%s): failed, got err (%v), expected err (%v)", tc.desc, resp, tc.wantErr)
			}

		}()
	}
}

// Tests the deadlines configured in backend rules for a gRPC sidecar backends.
func TestDeadlinesForGrpcCatchAllBackend(t *testing.T) {
	args := []string{
		"--service_config_id=test-config-id",

		"--rollout_strategy=fixed",
		"--backend_dns_lookup_family=v4only",
		"--suppress_envoy_headers",
	}

	s := env.NewTestEnv(comp.TestDeadlinesForGrpcCatchAllBackend, platform.GrpcEchoSidecar)

	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc     string
		wantErr  string
		testPlan string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc: "Success after 10s due to ESPv2 default response timeout being 15s",
			testPlan: `
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 10
    }
  }
}`,
		},
		{
			desc:    "Fail before 20s due to ESPv2 default response timeout being 15s",
			wantErr: "upstream request timeout",
			testPlan: `
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 20
    }
  }
}`,
		},
		{
			desc: "Success after 20s because ESPv2 automatically disables response timeouts for streaming RPCs",
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 20
    }
    count: 1
  }
}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow efficient measuring of elapsed time.
		// Elapsed time is not checked in the test, it's just for debugging.
		func() {
			defer elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

			// For this client, `err` will always be "exit status 1" on failures.
			// Check for actual error in `resp` instead.
			resp, err := client.RunGRPCEchoTest(tc.testPlan, s.Ports().ListenerPort)

			if tc.wantErr == "" && err != nil {
				t.Errorf("Test (%v): Error during running test: want no err, got err (%v)", tc.desc, resp)
			}

			if tc.wantErr != "" && err == nil {
				t.Errorf("Test (%v): Error during running test: got no err, want err (%v)", tc.desc, tc.wantErr)
			}

			if err != nil && !strings.Contains(resp, tc.wantErr) {
				t.Errorf("Test (%s): failed, got err (%v), expected err (%v)", tc.desc, resp, tc.wantErr)
			}

		}()
	}
}
