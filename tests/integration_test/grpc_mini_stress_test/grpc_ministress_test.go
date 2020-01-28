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

package grpc_mini_stress_test

import (
	"regexp"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestGRPCMinistress(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCMinistress, platform.GrpcEchoSidecar)
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testPlans := `
plans {
  parallel {
    test_count: 100
    parallel_limit: 10
    subtests {
      weight: 1
      echo {
        request {
          text: "Hello, world!"
        }
        call_config {
          api_key: "this-is-an-api-key"
        }
      }
    }
    subtests {
      weight: 1
      echo_stream {
        request {
          text: "Hello, world!"
        }
        call_config {
          api_key: "this-is-an-api-key"
        }
        count: 100
      }
    }
    subtests {
      weight: 1
      echo {
        request {
          text: "Hello, world!"
        }
        expected_status {
          code: 16
          details: "UNAUTHENTICATED:Method doesn\'t allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API."
        }
      }
    }
    subtests {
      weight: 1
      echo_stream {
        request {
          text: "Hello, world!"
        }
        count: 100
        expected_status {
          code: 16
          details: "UNAUTHENTICATED:Method doesn\'t allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API."
        }
      }
    }
  }
}`
	wantResult := regexp.MustCompile(`
Complete requests 100
Failed requests 0
Writing test outputs
results {
  parallel {
    total_time_micros: \d+
    stats {
      succeeded_count: \d+
      mean_latency_micros: \d+
      stddev_latency_micros: \d+
    }
    stats {
      succeeded_count: \d+
      mean_latency_micros: \d+
      stddev_latency_micros: \d+
    }
    stats {
      succeeded_count: \d+
      mean_latency_micros: \d+
      stddev_latency_micros: \d+
    }
    stats {
      succeeded_count: \d+
      mean_latency_micros: \d+
      stddev_latency_micros: \d+
    }
  }
}`)

	result, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	if err != nil {
		t.Errorf("Error during running test: %v", err)
	}
	if !wantResult.MatchString(result) {
		t.Errorf("The results are different,\nreceived:\n%s,\nwanted:\n%s", result, wantResult)
	}
}
