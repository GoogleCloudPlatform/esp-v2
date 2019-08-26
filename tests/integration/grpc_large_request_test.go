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

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/grpc-echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestGRPCLargeRequest(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCFallback, "grpc-echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testPlans := `
	plans {
	 echo {
	   call_config {
	     api_key: "this-is-an-api-key"
	   }
	   request {
	     space_payload_size: 30000000
	   }
	 }
	}`
	result, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	wantResult := ``
	if err != nil {
		t.Errorf("TestGRPCErrors: error during tests: %v", err)
	}
	if !strings.Contains(result, "echo") {
		t.Errorf("TestGRPCErrors: the results are different,\nreceived:\n%s,\nwanted:\n%s", result, wantResult)
	}
}
