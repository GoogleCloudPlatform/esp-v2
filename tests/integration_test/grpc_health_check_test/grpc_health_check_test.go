// Copyright 2021 Google LLC
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

package grpc_health_check_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func TestHealthCheckGrpcBackend(t *testing.T) {
	type HealthPeriod struct {
		backend  bool
		expected bool
	}
	tests := []struct {
		desc               string
		healthCheckBackend bool
		periods            []HealthPeriod
	}{
		{
			desc:               "not health check grpc backend",
			healthCheckBackend: false,
			periods:            []HealthPeriod{{backend: true, expected: true}, {backend: false, expected: true}, {backend: true, expected: true}},
		},
		{
			desc:               "health check grpc backend",
			healthCheckBackend: true,
			periods:            []HealthPeriod{{backend: true, expected: true}, {backend: false, expected: false}, {backend: true, expected: true}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			configID := "test-config-id"
			args := []string{"--service_config_id=" + configID, "--rollout_strategy=fixed", "--healthz=healthz"}
			if tc.healthCheckBackend {
				// When Envoy starts, it uses "no_traffic_interval" to set a timer to health check the cluster.
				// Only after that timer fired, it will use regular interval if there are some traffic.
				// But the default no_traffic_interval is 60s, it is too long for this test.
				// Here we need to change it to 1s.
				args = append(args, "--health_check_grpc_backend", "--health_check_grpc_backend_no_traffic_interval=1s")
			}

			s := env.NewTestEnv(platform.TestHealthCheckGrpcBackend, platform.GrpcBookstoreSidecar)
			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			healthUrl := fmt.Sprintf("http://%v:%v/healthz", platform.GetLoopbackAddress(), s.Ports().ListenerPort)

			for idx, period := range tc.periods {
				s.SetBookstoreServerHealthState(period.backend)

				// Wait for 5 seconds for cluster to detect the changes,
				time.Sleep(5 * time.Second)

				// http health check
				resp, err := http.Get(healthUrl)
				if err != nil {
					t.Fatalf("fail to healthz url: err: %v", err)
				}

				got := (resp.StatusCode == http.StatusOK)
				if period.expected != got {
					t.Errorf("Failed in period %v healthy state, expected: %v but got: %v", idx, period.expected, got)
				}
			}
		})
	}
}
