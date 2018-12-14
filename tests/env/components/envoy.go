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
	"time"

	"github.com/golang/glog"
)

const (
	testEnvTTL = 5 * time.Second
	envoyPath  = "../../bazel-bin/src/envoy/envoy"
)

// Envoy stores data for Envoy process
type Envoy struct {
	cmd    *exec.Cmd
	baseID string
}

// NewEnvoy creates a new Envoy struct and starts envoy.
func NewEnvoy(debugMode bool, confPath string) (*Envoy, error) {
	args := []string{"-c", confPath}
	if debugMode {
		args = append(args, "--log-level", "debug", "--drain-time-s", "1")
	}

	cmd := exec.Command(envoyPath, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &Envoy{
		cmd: cmd,
	}, nil
}

// Start starts the envoy process
func (s *Envoy) Start() error {
	return s.cmd.Start()
}

// Stop stops the envoy process
func (s *Envoy) Stop() error {
	glog.Infof("stop envoy ...\n")
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-time.After(testEnvTTL):
		glog.Infof("envoy killed as timeout reached")
		if err := s.cmd.Process.Kill(); err != nil {
			return err
		}
	case err := <-done:
		glog.Infof("stop envoy ... done\n")
		return err
	}
	return nil
}
