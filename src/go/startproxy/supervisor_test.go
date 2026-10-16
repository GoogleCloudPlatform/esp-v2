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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return true // process does not exist
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// harness runs a launcher against mock bootstrap, configmanager and envoy
// scripts in a temporary directory. Signal registration is intercepted so
// tests inject SIGTERM into run's signal channel instead of signalling the
// shared test process, and the metadata server is never queried.
type harness struct {
	t          *testing.T
	dir        string
	l          *launcher
	registered chan chan<- os.Signal
	exitCh     chan int
}

// newHarness creates a harness. scripts receives the harness directory and
// returns the mock configmanager and envoy script bodies.
func newHarness(t *testing.T, scripts func(dir string) (cm, envoy string)) *harness {
	t.Helper()
	h := &harness{
		t:          t,
		dir:        t.TempDir(),
		l:          newLauncher(),
		registered: make(chan chan<- os.Signal, 1),
		exitCh:     make(chan int, 1),
	}
	cm, envoy := scripts(h.dir)
	h.l.bootstrapBin = h.path("bootstrap")
	h.l.configManagerBin = h.path("configmanager")
	h.l.envoyBin = h.path("envoy")
	writeMockScript(t, h.l.bootstrapBin, "#!/bin/sh\nexit 0\n")
	writeMockScript(t, h.l.configManagerBin, cm)
	writeMockScript(t, h.l.envoyBin, envoy)
	h.l.notifySignals = func(c chan<- os.Signal) { h.registered <- c }
	h.l.stopSignals = func(chan<- os.Signal) {}
	h.l.metadataProjectID = func() string { return "" }
	unsetEnvForTest(t, otelResourceAttributesEnv)
	return h
}

func (h *harness) path(name string) string {
	return filepath.Join(h.dir, name)
}

// start runs the launcher with argv in the background.
func (h *harness) start(argv ...string) {
	go func() {
		h.exitCh <- h.l.run(argv)
	}()
}

// signals waits for run to register for shutdown signals and returns the
// channel signals are delivered on.
func (h *harness) signals() chan<- os.Signal {
	h.t.Helper()
	select {
	case c := <-h.registered:
		return c
	case <-time.After(5 * time.Second):
		h.t.Fatalf("run did not register for signals")
		return nil
	}
}

// waitForPid waits for a mock child to write its pid file and registers a
// cleanup that SIGKILLs it, so a failing test never leaks child processes.
func (h *harness) waitForPid(name string) int {
	h.t.Helper()
	b, err := waitForFile(h.path(name), 5*time.Second)
	if err != nil {
		h.t.Fatalf("failed waiting for %s: %v", name, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		h.t.Fatalf("bad pid in %s: %q", name, b)
	}
	h.t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	return pid
}

func (h *harness) waitForRunExit(timeout time.Duration) int {
	h.t.Helper()
	select {
	case code := <-h.exitCh:
		return code
	case <-time.After(timeout):
		h.t.Fatalf("run did not exit within %v", timeout)
		return -1
	}
}

var defaultArgv = []string{"--service=test.com", "--version=v1"}

// trappingChild returns a mock child script that installs a TERM/INT trap
// *before* publishing its pid (so a signal can never beat the trap), runs
// onTerm when signalled, and otherwise runs forever.
func trappingChild(pidFile, onTerm string) string {
	return fmt.Sprintf("#!/bin/sh\ntrap '%s' TERM INT\necho $$ > %s\nwhile true; do sleep 1 & wait $!; done\n", onTerm, pidFile)
}

// exitingChild returns a mock child script that publishes its pid, waits until
// the file waitFor exists (so its sibling is up), then exits with code 42.
func exitingChild(pidFile, waitFor string) string {
	return fmt.Sprintf("#!/bin/sh\necho $$ > %s\nwhile [ ! -f %s ]; do sleep 0.01; done\nexit 42\n", pidFile, waitFor)
}

// TestSupervisorSIGTERMForwarding verifies that run forwards SIGTERM to both
// Config Manager and Envoy and exits once both have shut down.
func TestSupervisorSIGTERMForwarding(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		cm := trappingChild(filepath.Join(dir, "cm.pid"), "echo term > "+filepath.Join(dir, "cm.term")+"; exit 0")
		envoy := trappingChild(filepath.Join(dir, "envoy.pid"), "echo term > "+filepath.Join(dir, "envoy.term")+"; exit 0")
		return cm, envoy
	})
	h.start(defaultArgv...)
	sigCh := h.signals()
	cmPid := h.waitForPid("cm.pid")
	envoyPid := h.waitForPid("envoy.pid")

	sigCh <- syscall.SIGTERM

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if _, err := os.Stat(h.path("cm.term")); err != nil {
		t.Errorf("Config Manager did not receive SIGTERM: %v", err)
	}
	if _, err := os.Stat(h.path("envoy.term")); err != nil {
		t.Errorf("Envoy did not receive SIGTERM: %v", err)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) is still running", cmPid)
	}
	if !waitForProcessExit(envoyPid, 3*time.Second) {
		t.Errorf("Envoy (PID %d) is still running", envoyPid)
	}
}

// TestSupervisorSIGTERMWaitsForSlowChild verifies that when one child exits
// promptly on SIGTERM, the other is still allowed to finish its own graceful
// shutdown rather than being SIGKILLed.
func TestSupervisorSIGTERMWaitsForSlowChild(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		cm := trappingChild(filepath.Join(dir, "cm.pid"), "exit 0")
		// Envoy takes 500ms to drain after SIGTERM.
		envoy := trappingChild(filepath.Join(dir, "envoy.pid"), "sleep 0.5; echo done > "+filepath.Join(dir, "envoy.done")+"; exit 0")
		return cm, envoy
	})
	h.start(defaultArgv...)
	sigCh := h.signals()
	h.waitForPid("cm.pid")
	h.waitForPid("envoy.pid")

	sigCh <- syscall.SIGTERM

	h.waitForRunExit(5 * time.Second)
	if _, err := os.Stat(h.path("envoy.done")); err != nil {
		t.Errorf("Envoy was killed before finishing graceful shutdown: %v", err)
	}
}

// TestSupervisorSIGTERMGracePeriodExpires verifies that children which ignore
// SIGTERM are SIGKILLed once the shutdown grace period elapses.
func TestSupervisorSIGTERMGracePeriodExpires(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		// ":" is a no-op, so both children ignore SIGTERM.
		return trappingChild(filepath.Join(dir, "cm.pid"), ":"),
			trappingChild(filepath.Join(dir, "envoy.pid"), ":")
	})
	h.l.shutdownGrace = 300 * time.Millisecond
	h.start(defaultArgv...)
	sigCh := h.signals()
	cmPid := h.waitForPid("cm.pid")
	envoyPid := h.waitForPid("envoy.pid")

	sigCh <- syscall.SIGTERM

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) was not killed after grace period", cmPid)
	}
	if !waitForProcessExit(envoyPid, 3*time.Second) {
		t.Errorf("Envoy (PID %d) was not killed after grace period", envoyPid)
	}
}

// TestSupervisorKillsEnvoyWhenConfigManagerDies verifies that when Config
// Manager terminates prematurely, Envoy is killed and run exits with 1.
func TestSupervisorKillsEnvoyWhenConfigManagerDies(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		// Config Manager exits once Envoy is up; Envoy runs until killed.
		cm := exitingChild(filepath.Join(dir, "cm.pid"), filepath.Join(dir, "envoy.pid"))
		envoy := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nwhile true; do sleep 0.05; done\n", filepath.Join(dir, "envoy.pid"))
		return cm, envoy
	})
	h.start(defaultArgv...)
	envoyPid := h.waitForPid("envoy.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(envoyPid, 3*time.Second) {
		t.Errorf("Envoy (PID %d) was not killed after Config Manager died", envoyPid)
	}
}

// TestSupervisorKillsConfigManagerWhenEnvoyDies verifies that when Envoy
// terminates prematurely, Config Manager is killed and run exits with 1.
func TestSupervisorKillsConfigManagerWhenEnvoyDies(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		// Config Manager runs until killed; Envoy exits once it is up.
		cm := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nwhile true; do sleep 0.05; done\n", filepath.Join(dir, "cm.pid"))
		envoy := exitingChild(filepath.Join(dir, "envoy.pid"), filepath.Join(dir, "cm.pid"))
		return cm, envoy
	})
	h.start(defaultArgv...)
	cmPid := h.waitForPid("cm.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) was not killed after Envoy died", cmPid)
	}
}

// TestRunBootstrapFailure verifies that a failing bootstrap step stops
// startup: Config Manager is killed, Envoy is never started, and run exits 1.
func TestRunBootstrapFailure(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		cm := trappingChild(filepath.Join(dir, "cm.pid"), "exit 0")
		envoy := "#!/bin/sh\necho started > " + filepath.Join(dir, "envoy.started") + "\n"
		return cm, envoy
	})
	// Fail only once Config Manager is up, so its pid is known.
	writeMockScript(t, h.l.bootstrapBin, fmt.Sprintf("#!/bin/sh\nwhile [ ! -f %s ]; do sleep 0.01; done\nexit 3\n", h.path("cm.pid")))
	h.start(defaultArgv...)
	cmPid := h.waitForPid("cm.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) is still running", cmPid)
	}
	if _, err := os.Stat(h.path("envoy.started")); err == nil {
		t.Errorf("Envoy was started although the bootstrap config was not generated")
	}
}

// TestRunEnvoyStartFailure verifies that Config Manager is killed and run
// exits 1 if Envoy cannot be started.
func TestRunEnvoyStartFailure(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		return trappingChild(filepath.Join(dir, "cm.pid"), "exit 0"), ""
	})
	h.l.envoyBin = h.path("no-such-envoy")
	writeMockScript(t, h.l.bootstrapBin, fmt.Sprintf("#!/bin/sh\nwhile [ ! -f %s ]; do sleep 0.01; done\n", h.path("cm.pid")))
	h.start(defaultArgv...)
	cmPid := h.waitForPid("cm.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) is still running", cmPid)
	}
}

// TestRunConflictingFlags verifies that a flag conflict stops startup before
// any child process is started.
func TestRunConflictingFlags(t *testing.T) {
	h := newHarness(t, func(dir string) (string, string) {
		started := "#!/bin/sh\necho started > " + filepath.Join(dir, "started") + "\n"
		return started, started
	})
	h.start("--rollout_strategy=fixed")
	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("run exited with code %d, want 1", code)
	}
	if _, err := os.Stat(h.path("started")); err == nil {
		t.Errorf("a child process was started despite the flag conflict")
	}
}
