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

package grpc_interop_test

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func runAndWait(cmd *exec.Cmd, t *testing.T) {
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				t.Errorf("Exit Status: %d", status.ExitStatus())
			}
		} else {
			t.Fatalf("cmd.Wait: %v", err)
		}
	}
}

func TestGRPCInterops(t *testing.T) {
	t.Parallel()

	serviceName := "grpc-interop-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestGRPCInterops, platform.GrpcInteropSidecar)
	clientPath := platform.GetFilePath(platform.GrpcInteropClient)
	_, err := os.Stat(clientPath)
	if os.IsNotExist(err) {
		t.Fatalf("TestGRPCInteropMiniStress: grpc-interop test binaris are not built. Please run make build-grpc-interop.")
	}

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	serverPortFlag := fmt.Sprintf("--server_port=%v", s.Ports().ListenerPort)
	testcases := []string{
		"cancel_after_begin",
		"cancel_after_first_response",
		"empty_unary",
		"large_unary",
		"client_streaming",
		"empty_stream",
		"ping_pong",
		"server_streaming",
		"timeout_on_sleeping_server",
		"status_code_and_message",
		"custom_metadata",
	}

	for _, tc := range testcases {
		testcaseFlag := fmt.Sprintf("--test_case=%v", tc)
		cmd := exec.Command(clientPath, serverPortFlag, testcaseFlag, "--additional_metadata=x-api-key:api-key")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		runAndWait(cmd, t)

	}
}

func TestGRPCInteropMiniStress(t *testing.T) {
	t.Parallel()

	serviceName := "grpc-interop-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestGRPCInteropMiniStress, platform.GrpcInteropSidecar)
	clientPath := platform.GetFilePath(platform.GrpcInteropStressClient)
	_, err := os.Stat(clientPath)
	if os.IsNotExist(err) {
		t.Fatalf("TestGRPCInteropMiniStress: grpc-interop test binaris are not built. Please run make build-grpc-interop.")
	}

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	serverAddrFlag := fmt.Sprintf("--server_addresses=%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
	testcasesFlag := "--test_cases=empty_unary:10,large_unary:10,empty_stream:10,client_streaming:10,ping_pong:20,server_streaming:10,status_code_and_message:10,custom_metadata:10"
	cmd := exec.Command(clientPath, serverAddrFlag, testcasesFlag, "--test_duration_secs=10", "--num_channels_per_server=200", "--num_stubs_per_channel=1")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	runAndWait(cmd, t)

}
