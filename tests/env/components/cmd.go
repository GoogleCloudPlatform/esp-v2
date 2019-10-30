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
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

const (
	startWaitTime = 500 * time.Millisecond
	stopWaitTime  = 500 * time.Millisecond
)

type Cmd struct {
	name string
	*exec.Cmd
}

// StartAndWait starts the command and waits for startWaitTime.
func (c *Cmd) StartAndWait() error {
	glog.Infof("Starting %v ...", c.name)
	if err := c.Start(); err != nil {
		return err
	}
	time.Sleep(startWaitTime)
	return nil
}

// StopAndWait stops the command and waits for stopWaitTime, and kills the command if it doesn't
// stop.
func (c *Cmd) StopAndWait() error {
	glog.Infof("Stopping %v ...", c.name)

	done := make(chan error)
	go func() {
		done <- c.Wait()
	}()

	// Send termination
	err := c.Process.Signal(os.Interrupt)
	if err != nil {
		glog.Infof("Error interrupting process %v: %v", c.name, err)
	}

	select {
	case <-time.After(stopWaitTime):
		glog.Infof("%s killed as timeout reached", c.name)
		if err := c.Process.Kill(); err != nil {
			return err
		}
	case err := <-done:
		glog.Infof("Stopped %v\n", c.name)
		return err
	}
	return nil
}
