// Copyright 2019 Google LLC
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

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
)

type ConfigManagerServer struct {
	*Cmd
	adsNamedPipe string
}

func NewConfigManagerServer(debugMode bool, ports *platform.Ports, args []string) (*ConfigManagerServer, error) {

	adsNamedPipe := fmt.Sprintf("@espv2-ads-cluster-integ-test-%v", ports.TestId)

	// Set config manager flags.
	args = append(args, "--listener_address", platform.GetAnyAddress())
	args = append(args, "--ssl_backend_client_root_certs_path", platform.GetFilePath(platform.ProxyCert))
	args = append(args, fmt.Sprintf("--ads_named_pipe=%v", adsNamedPipe))

	if debugMode {
		args = append(args, "--logtostderr", "--v=1")
	}

	glog.Infof("config manager args: %v", args)
	cmd := exec.Command(platform.GetFilePath(platform.ConfigManager), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &ConfigManagerServer{
		Cmd: &Cmd{
			name: "ConfigManager",
			Cmd:  cmd,
		},
		adsNamedPipe: adsNamedPipe,
	}, nil
}

func (s ConfigManagerServer) String() string {
	return "Config Manager gRPC Server"
}

func (s ConfigManagerServer) CheckHealth() error {
	opts := NewHealthCheckOptions()
	return UdsConnectionCheck(s.adsNamedPipe, opts)
}
