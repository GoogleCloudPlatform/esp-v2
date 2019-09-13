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

	"cloudesf.googlesource.com/gcpproxy/tests/env/platform"
	"github.com/golang/glog"
)

// Envoy stores data for Envoy process
type Envoy struct {
	*Cmd
	adminPort uint16
}

// createEnvoyConf create envoy config.
func createEnvoyConf(configPath string, bootstrapArgs []string, ports *Ports) error {

	glog.Infof("Outputting envoy bootstrap config to: %v", configPath)

	bootstrapArgs = append(bootstrapArgs, "--enable_tracing=true")
	bootstrapArgs = append(bootstrapArgs, "--tracing_sample_rate=1.0")
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--discovery_address=http://%v:%v", platform.GetLoopbackAddress(), ports.DiscoveryPort))
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--admin_port=%v", ports.AdminPort))
	// This address must be in gRPC format: https://github.com/grpc/grpc/blob/master/doc/naming.md
	bootstrapArgs = append(bootstrapArgs, fmt.Sprintf("--tracing_stackdriver_address=%v:%v:%v", platform.GetIpProtocol(), platform.GetLoopbackAddress(), ports.FakeStackdriverPort))
	bootstrapArgs = append(bootstrapArgs, "--admin_address", platform.GetAnyAddress())
	bootstrapArgs = append(bootstrapArgs, configPath)

	// Call bootstrapper to create the bootstrap config
	glog.Infof("Calling bootstrapper at %v with args: %v", platform.GetFilePath(platform.Bootstrapper), bootstrapArgs)
	cmd := exec.Command(platform.GetFilePath(platform.Bootstrapper), bootstrapArgs...)
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

	glog.Infof("Calling envoy at %v with args: %v", platform.GetFilePath(platform.Envoy), args)
	cmd := exec.Command(platform.GetFilePath(platform.Envoy), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &Envoy{
		Cmd: &Cmd{
			name: "Envoy",
			Cmd:  cmd,
		},
		adminPort: ports.AdminPort,
	}, nil
}

func (s Envoy) String() string {
	return "Envoy Proxy Admin HTTP Endpoint"
}

func (s Envoy) CheckHealth() error {
	opts := NewHealthCheckOptions()
	addr := fmt.Sprintf("http://localhost:%v", s.adminPort)
	return HttpHealthCheck(addr, "/ready", opts)
}
