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

package integration_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// Tests the deadlines configured in backend rules for a gRPC remote backends.
func TestDeadlinesForGrpcDynamicRouting(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestDeadlinesForGrpcDynamicRouting, platform.GrpcEchoRemote)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
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
		t.Run(tc.desc, func(t *testing.T) {
			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

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
		})
	}
}

// Tests the deadlines configured in backend rules for a gRPC sidecar backends.
func TestDeadlinesForGrpcCatchAllBackend(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestDeadlinesForGrpcCatchAllBackend, platform.GrpcEchoSidecar)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
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
		t.Run(tc.desc, func(t *testing.T) {
			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

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
		})
	}
}

func TestIdleTimeoutsForGrpcStreaming(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc        string
		confArgs    []string
		addDeadline time.Duration
		wantErr     string
		testPlan    string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc: "When deadline is NOT specified for method, stream idle timeout specified via flag kicks in and the request fails.",
			confArgs: append([]string{
				"--stream_idle_timeout=15s",
			}, utils.CommonArgs()...),
			wantErr: `stream timeout`,
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
		{
			desc:        "When deadline is specified for method, it overrides the global stream idle timeout and the request succeeds.",
			addDeadline: 25 * time.Second,
			confArgs: append([]string{
				"--stream_idle_timeout=15s",
			}, utils.CommonArgs()...),
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
		{
			desc: "When deadline is NOT specified for method, a low stream idle timeout specified via flag is not honored and the request succeeds.",
			confArgs: append([]string{
				"--stream_idle_timeout=3s",
			}, utils.CommonArgs()...),
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 7
    }
    count: 1
  }
}`,
		},
	}

	for _, tc := range testData {
		// Place in closure to allow efficient measuring of elapsed time.
		// Elapsed time is not checked in the test, it's just for debugging.
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestDeadlinesForGrpcCatchAllBackend, platform.GrpcEchoSidecar)

			if tc.addDeadline != 0 {
				s.AppendBackendRules([]*confpb.BackendRule{
					{
						Selector: "test.grpc.Test.EchoStream",
						Deadline: tc.addDeadline.Seconds(),
					},
				})
			}

			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

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

		})
	}
}
