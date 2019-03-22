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
	"os"
	"os/exec"
	"text/template"

	"github.com/golang/glog"
)

const (
	envoyPath = "../../bazel-bin/src/envoy/envoy"
)
const envoyConfBootstrapYaml = `
node:
  id: "api_proxy"
  cluster: "api_proxy_cluster"

dynamic_resources:
  lds_config: {ads: {}}
  cds_config: {ads: {}}
  ads_config:
    api_type: GRPC
    grpc_services:
      envoy_grpc:
        cluster_name: ads_cluster

static_resources:
  clusters:
  - name: ads_cluster
    connect_timeout: { seconds: 5 }
    type: STATIC
    hosts:
    - socket_address:
        address: 127.0.0.1
        port_value: {{.DiscoveryPort}}
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}

admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 0.0.0.0
      port_value: {{.AdminPort}}
`

// Envoy stores data for Envoy process
type Envoy struct {
	*Cmd
}

// CreateEnvoyConf create envoy config.
func CreateEnvoyConf(path string, ports *Ports) error {
	confTmpl := envoyConfBootstrapYaml
	tmpl, err := template.New("test").Parse(confTmpl)
	if err != nil {
		glog.Errorf("failed to parse config YAML template: %v", err)
		return err
	}

	yamlFile, err := os.Create(path)
	if err != nil {
		glog.Errorf("failed to create YAML file %v: %v", path, err)
		return err
	}
	defer func() {
		_ = yamlFile.Close()
	}()

	return tmpl.Execute(yamlFile, ports)
}

// NewEnvoy creates a new Envoy struct and starts envoy.
func NewEnvoy(debugMode bool, confPath string, ports *Ports) (*Envoy, error) {
	// set concurrency to 1 to have only one worker thread to test client cache.
	args := []string{"-c", confPath, "--concurrency", "1"}
	if err := CreateEnvoyConf(confPath, ports); err != nil {
		return nil, err
	}
	if debugMode {
		args = append(args, "--log-level", "debug", "--drain-time-s", "1")
	}

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
