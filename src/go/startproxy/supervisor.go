// Copyright 2026 Google LLC
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

package startproxy

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/golang/glog"
)

// child is a supervised subprocess.
type child struct {
	// name is used in "<name> is down" messages.
	name string
	// killName is used in "killing <killName> process" messages. Both match
	// start_proxy.py's log text.
	killName string
	cmd      *exec.Cmd
	waitCh   <-chan error
	exited   bool
}

// startChild starts argv with its stdout and stderr merged into ours, like
// start_proxy.py relayed child output.
func startChild(name, killName string, argv []string) (*child, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	return &child{name: name, killName: killName, cmd: cmd, waitCh: waitCh}, nil
}

func (c *child) terminate() {
	if c.exited {
		return
	}
	glog.Infof("sending TERM to PID=%d", c.cmd.Process.Pid)
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		glog.Errorf("error sending TERM to PID=%d continuing", c.cmd.Process.Pid)
	}
}

func (c *child) kill() {
	if c.exited {
		return
	}
	glog.Infof("Killing process: pid=%d", c.cmd.Process.Pid)
	if err := c.cmd.Process.Kill(); err != nil {
		glog.Errorf("The child process: pid=%d may not exist.", c.cmd.Process.Pid)
	}
}

// signalName returns the signal's name as start_proxy.py logged it.
func signalName(sig os.Signal) string {
	switch sig {
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGINT:
		return "SIGINT"
	}
	return sig.String()
}

// supervise monitors Config Manager and Envoy until both have exited.
//
//   - If either child exits unexpectedly, the other is killed immediately.
//   - On SIGTERM/SIGINT, SIGTERM is forwarded to both children and they are
//     given l.shutdownGrace to exit on their own; one child finishing its
//     shutdown first must not cause the other to be SIGKILLed mid-shutdown.
//     Survivors are SIGKILLed when the grace period expires.
//
// It always returns 1, matching the behavior of the original start_proxy.py.
func (l *launcher) supervise(cm, envoy *child, sigCh <-chan os.Signal) int {
	children := []*child{cm, envoy}
	terminating := false
	var graceTimeout <-chan time.Time

	onExit := func(c *child) (done bool) {
		c.exited = true
		c.waitCh = nil // never selected again
		if terminating {
			glog.Infof("%s exited", c.name)
			return cm.exited && envoy.exited
		}
		other := envoy
		if c == envoy {
			other = cm
		}
		glog.Errorf("%s is down, killing %s process.", c.name, other.killName)
		other.kill()
		return true
	}

	for {
		select {
		case sig := <-sigCh:
			glog.Warningf("got signal: %s", signalName(sig))
			for _, c := range children {
				c.terminate()
			}
			if !terminating {
				terminating = true
				graceTimeout = time.After(l.shutdownGrace)
			}
		case <-cm.waitCh:
			if onExit(cm) {
				return 1
			}
		case <-envoy.waitCh:
			if onExit(envoy) {
				return 1
			}
		case <-graceTimeout:
			glog.Warningf("child processes did not exit within %v of signal", l.shutdownGrace)
			for _, c := range children {
				c.kill()
			}
			return 1
		}
	}
}
