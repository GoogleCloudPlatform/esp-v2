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

package grpc_deadline_test

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
			desc: "Fail before 15s due to user-configured response deadline being 10s",
			// TODO(b/185919750):deflake the timeout integration tests on 408 downstream timeout  and 504 upstream timeout.
			wantErr: "timeout",
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
			desc: "Fail before 20s due to ESPv2 default response timeout being 15s",
			// TODO(b/185919750):deflake the timeout integration tests on 408 downstream timeout  and 504 upstream timeout.
			wantErr: "timeout",
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

// Tests the stream idle timeouts configured via deadline in backend rules for gRPC streaming methods.
// gRPC streaming methods do not adhere to deadlines, they use stream idle timeouts.
func TestIdleTimeoutsForGrpcStreaming(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc           string
		confArgs       []string
		methodDeadline time.Duration
		wantErr        string
		testPlan       string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			// route deadline = 15s (default, not explicitly specified), global stream idle timeout = 17s, request = 20s
			// This 408 is caused by global stream idle timeout because deadline was not explicitly specified.
			desc: "When deadline is NOT specified, stream idle timeout specified via flag kicks in and the request fails with 408.",
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=17s",
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
			// route deadline = 15s (default, not explicitly specified), global stream idle timeout = 20, request = 17s
			// Global stream idle timeout is used because route deadline is not configured. Request is under the route's stream idle timeout, so it succeeds.
			desc: "When deadline is NOT specified, the global idle timeout flag is honored. But it is large, so the request succeeds.",
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=20s",
			}, utils.CommonArgs()...),
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 17
    }
    count: 1
  }
}`,
		},
		{
			// route deadline = 15s (default, not explicitly specified), global stream idle timeout = 3s, request = 7s
			// Stream idle timeout is automatically increased to match the default route deadline. Request is under the route's stream idle timeout, so it succeeds.
			desc: "When deadline is NOT specified, ESPv2 does not honor the global idle timeout flag if the value is lower than the default deadline (15s). The request succeeds.",
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=3s",
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
		{
			// route deadline = 25s, global stream idle timeout = 15s, request = 20s
			// Stream idle timeout is automatically increased to match the specified route deadline. Request is under the route's stream idle timeout, so it succeeds.
			desc:           "When a large deadline is specified, it overrides the global stream idle timeout specified by flag. The request succeeds.",
			methodDeadline: 25 * time.Second,
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=15s",
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
			// route deadline = 5s, global stream idle timeout = 15s, request = 10s
			// This 408 is caused by the route's stream idle timeout because deadline was explicitly configured.
			desc:           "When a small deadline is specified, it overrides the larger global stream idle timeout specified by flag. The stream fails with a 408, not 504.",
			methodDeadline: 5 * time.Second,
			wantErr:        "stream timeout",
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=15s",
			}, utils.CommonArgs()...),
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 10
    }
    count: 1
  }
}`,
		},
		{
			// route deadline = 5s, global stream idle timeout = 2s, request = 8s
			// This 408 is caused by the route's stream idle timeout because deadline was explicitly configured.
			desc:           "When a small deadline is specified, it overrides the smaller global stream idle timeout specified by flag. The stream fails with a 408, not 504.",
			methodDeadline: 5 * time.Second,
			wantErr:        "stream timeout",
			confArgs: append([]string{
				"--stream_idle_timeout_test_only=2s",
			}, utils.CommonArgs()...),
			testPlan: `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
      response_delay: 8
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
			s := env.NewTestEnv(platform.TestIdleTimeoutsForGrpcStreaming, platform.GrpcEchoSidecar)

			// b/194502699: Always create the backend rule, even if deadline is 0.
			s.AppendBackendRules([]*confpb.BackendRule{
				{
					Selector: "test.grpc.Test.EchoStream",
					Deadline: tc.methodDeadline.Seconds(),
				},
			})

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
