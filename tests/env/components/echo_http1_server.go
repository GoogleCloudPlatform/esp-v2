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
	echoPath = "../../bin/echo/server"
)

// Echo stores data for Echo HTTP/1 backend process.
type EchoHTTPServer struct {
	cmd *exec.Cmd
}

func NewEchoHTTPServer() (*EchoHTTPServer, error) {
	cmd := exec.Command(echoPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &EchoHTTPServer{
		cmd: cmd,
	}, nil
}

// Start starts the Echo process.
func (e *EchoHTTPServer) Start() <-chan error {
	glog.Infof("Starting Echo HTTP/1 Server...")
	errCh := make(chan error)
	go func() {
		err := e.cmd.Start()
		if err != nil {
			errCh <- err
		}
	}()

	// wait for server up.
	time.AfterFunc(1*time.Second, func() { close(errCh) })
	return errCh
}

// Stop stops the Echo process.
func (e *EchoHTTPServer) Stop() error {
	glog.Infof("Stop Echo server...")
	done := make(chan error, 1)
	go func() {
		done <- e.cmd.Wait()
	}()

	select {
	case <-time.After(testEnvTTL):
		if err := e.cmd.Process.Kill(); err != nil {
			return err
		}
	case err := <-done:
		glog.Infof("stop Echo ... done\n")
		return err
	}
	return nil
}
