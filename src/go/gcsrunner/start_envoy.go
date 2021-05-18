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

package gcsrunner

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

var execCommand = exec.Command

// StartEnvoyOptions provides a set of configurations when starting Envoy.
type StartEnvoyOptions struct {
	BinaryPath        string
	ComponentLogLevel string
	ConfigPath        string
	LogLevel          string
	LogPath           string
	TerminateTimeout  time.Duration
}

// StartEnvoyAndWait starts Envoy and waits.
//
// Any Envoy exit is assumed to be an error.
//
// Any signal sent to signalChan is expected to be an exit signal. A failure to
// signal Envoy results in an error.
func StartEnvoyAndWait(signalChan chan os.Signal, opts StartEnvoyOptions) error {
	startupFlags := []string{
		"--service-cluster", "front-envoy",
		"--service-node", "front-envoy",
		"--disable-hot-restart",
		"--config-path", opts.ConfigPath,
		"--log-level", opts.LogLevel,
		"--log-path", opts.LogPath,
		"--log-format", "%L%m%d %T.%e %t envoy] [%t][%n]%v",
		"--log-format-escaped",
		"--allow-unknown-static-fields",
	}
	if opts.ComponentLogLevel != "" {
		startupFlags = append(startupFlags, "--component-log-level", opts.ComponentLogLevel)
	}
	cmd := execCommand(opts.BinaryPath, startupFlags...)
	cmd.Env = append(cmd.Env, "TMPDIR=/tmp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Envoy: %v", err)
	}

	envoyExitChan := make(chan error)
	go func() {
		err := cmd.Wait()
		if err == nil {
			err = fmt.Errorf("unexpectedly exited OK from Envoy, which should never happen")
		}
		envoyExitChan <- err
	}()

	select {
	case err := <-envoyExitChan:
		return fmt.Errorf("envoy exited: %v", err)
	case sig := <-signalChan:
		if cmd.Process == nil {
			return fmt.Errorf("cmd not started, which should never happen")
		}
		glog.Errorf("Stopping Envoy due to signal: %v", sig)

		// This will always be a signal to stop the process.
		if err := cmd.Process.Signal(sig); err != nil {
			return fmt.Errorf("failed to signal Envoy: %v", err)
		}
		select {
		case err := <-envoyExitChan:
			return err
		case <-time.After(opts.TerminateTimeout):
			return fmt.Errorf("timed out waiting for Envoy to exit after %v", opts.TerminateTimeout)
		}
	}
}
