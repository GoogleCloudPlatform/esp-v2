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
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startSubprocess(cmdArgs []string) (*exec.Cmd, <-chan error, error) {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	return cmd, waitCh, nil
}

// child is a supervised subprocess.
type child struct {
	name   string
	cmd    *exec.Cmd
	waitCh <-chan error
	exited bool
}

func (c *child) signal(sig syscall.Signal) {
	if c.exited || c.cmd.Process == nil {
		return
	}
	if sig == syscall.SIGKILL {
		log.Printf("INFO: Killing process: pid=%d", c.cmd.Process.Pid)
		_ = c.cmd.Process.Kill()
		return
	}
	log.Printf("INFO: sending TERM to PID=%d", c.cmd.Process.Pid)
	if err := c.cmd.Process.Signal(sig); err != nil {
		log.Printf("ERROR: error sending TERM to PID=%d continuing", c.cmd.Process.Pid)
	}
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
			log.Printf("INFO: %s exited", c.name)
			return cm.exited && envoy.exited
		}
		other := envoy
		if c == envoy {
			other = cm
		}
		log.Printf("FATAL: %s is down, killing %s process.", c.name, other.name)
		other.signal(syscall.SIGKILL)
		return true
	}

	for {
		select {
		case sig := <-sigCh:
			log.Printf("WARNING: got signal: %v", sig)
			for _, c := range children {
				c.signal(syscall.SIGTERM)
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
			log.Printf("WARNING: child processes did not exit within %v of signal", l.shutdownGrace)
			for _, c := range children {
				c.signal(syscall.SIGKILL)
			}
			return 1
		}
	}
}
