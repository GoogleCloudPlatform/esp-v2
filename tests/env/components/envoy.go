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

package components

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/golang/glog"
)

const (
	envoyPath        = "../../bazel-bin/src/envoy/envoy"
	bootstrapperPath = "../../bin/bootstrap"
)

// Envoy stores data for Envoy process
type Envoy struct {
	*Cmd
}

// createEnvoyConf create envoy config.
func createEnvoyConf(configPath string, bootstrapArgs []string, ports *Ports) error {

	glog.Infof("Outputting envoy bootstrap config to: %v", configPath)

	bootstrapArgs = append(bootstrapArgs, "--enable_tracing=true")
	bootstrapArgs = append(bootstrapArgs, "--tracing_sample_rate=1.0")
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--discovery_address=http://127.0.0.1:%v", ports.DiscoveryPort))
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--admin_port=%v", ports.AdminPort))
	// This address must be in gRPC format: https://github.com/grpc/grpc/blob/master/doc/naming.md
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--tracing_stackdriver_address=ipv4:127.0.0.1:%v", ports.FakeStackdriverPort))
	bootstrapArgs = append(bootstrapArgs, configPath)

	// Call bootstrapper to create the bootstrap config
	glog.Infof("Calling bootstrapper at %v with args: %v", bootstrapperPath, bootstrapArgs)
	cmd := exec.Command(bootstrapperPath, bootstrapArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// NewEnvoy creates a new Envoy struct and starts envoy.
func NewEnvoy(args []string, bootstrapArgs []string, confPath string, ports *Ports, testId uint16) (*Envoy, error) {

	if err := createEnvoyConf(confPath, bootstrapArgs, ports); err != nil {
		return nil, err
	}

	args = append(args,
		"-c", confPath,
		// Set concurrency to 1 to have only one worker thread to test client cache.
		"--concurrency", "1",
		// Allows multiple envoys to run on a single machine. If one test fails to stop envoy, this ID
		// will allow other tests to run afterwords without conflicting.
		// See: https://www.envoyproxy.io/docs/envoy/latest/operations/cli#cmdoption-base-id
		"--base-id", strconv.Itoa(int(testId)),
	)

	glog.Infof("Calling envoy at %v with args: %v", envoyPath, args)
	cmd := exec.Command(envoyPath, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &Envoy{
		Cmd: &Cmd{
			name: "Envoy",
			Cmd:  cmd,
		},
	}, nil
}
