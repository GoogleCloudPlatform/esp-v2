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

func writeMockScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write mock script %s: %v", path, err)
	}
}

func waitForFile(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for %s", path)
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if err != nil {
			return true // process does not exist
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// supervisorHarness runs Run() against mock child scripts.
type supervisorHarness struct {
	t      *testing.T
	dir    string
	sigCh  chan<- os.Signal
	exitCh chan int
}

// newSupervisorHarness installs mock bootstrap/configmanager/envoy scripts,
// starts Run in the background, and intercepts signal registration so tests
// inject SIGTERM directly into Run's signal channel instead of signalling the
// shared test process. scripts receives the harness directory and returns the
// mock configmanager and envoy script bodies.
func newSupervisorHarness(t *testing.T, scripts func(dir string) (cm, envoy string)) *supervisorHarness {
	t.Helper()
	h := &supervisorHarness{t: t, dir: t.TempDir(), exitCh: make(chan int, 1)}
	cmScript, envoyScript := scripts(h.dir)

	mockBootstrap := filepath.Join(h.dir, "bootstrap")
	mockCM := filepath.Join(h.dir, "configmanager")
	mockEnvoy := filepath.Join(h.dir, "envoy")
	writeMockScript(t, mockBootstrap, "#!/bin/sh\nexit 0\n")
	writeMockScript(t, mockCM, cmScript)
	writeMockScript(t, mockEnvoy, envoyScript)

	origBootstrap, origCM, origEnvoy := BootstrapCmd, ConfigManagerBin, EnvoyBin
	origNotify, origStop := notifySignals, stopSignals
	t.Cleanup(func() {
		BootstrapCmd, ConfigManagerBin, EnvoyBin = origBootstrap, origCM, origEnvoy
		notifySignals, stopSignals = origNotify, origStop
	})
	BootstrapCmd, ConfigManagerBin, EnvoyBin = mockBootstrap, mockCM, mockEnvoy

	registered := make(chan chan<- os.Signal, 1)
	notifySignals = func(c chan<- os.Signal) { registered <- c }
	stopSignals = func(chan<- os.Signal) {}

	go func() {
		h.exitCh <- Run([]string{"--service=test.com", "--version=v1"})
	}()

	select {
	case h.sigCh = <-registered:
	case <-time.After(5 * time.Second):
		t.Fatalf("Run did not register for signals")
	}
	return h
}

func (h *supervisorHarness) path(name string) string {
	return filepath.Join(h.dir, name)
}

// waitForPid waits for a mock child to write its pid file and registers a
// cleanup that SIGKILLs it, so a failing test never leaks child processes.
func (h *supervisorHarness) waitForPid(name string) int {
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

func (h *supervisorHarness) waitForRunExit(timeout time.Duration) int {
	h.t.Helper()
	select {
	case code := <-h.exitCh:
		return code
	case <-time.After(timeout):
		h.t.Fatalf("Run did not exit within %v", timeout)
		return -1
	}
}

// trappingChild returns a mock child script that installs a TERM/INT trap
// *before* publishing its pid (so a signal can never beat the trap), runs
// onTerm when signalled, and otherwise runs forever.
func trappingChild(pidFile, onTerm string) string {
	return fmt.Sprintf("#!/bin/sh\ntrap '%s' TERM INT\necho $$ > %s\nwhile true; do sleep 1 & wait $!; done\n", onTerm, pidFile)
}

// TestSupervisorSIGTERMForwarding verifies that Run forwards SIGTERM to both
// Config Manager and Envoy and exits once both have shut down.
func TestSupervisorSIGTERMForwarding(t *testing.T) {
	h := newSupervisorHarness(t, func(dir string) (string, string) {
		cm := trappingChild(filepath.Join(dir, "cm.pid"), "echo term > "+filepath.Join(dir, "cm.term")+"; exit 0")
		envoy := trappingChild(filepath.Join(dir, "envoy.pid"), "echo term > "+filepath.Join(dir, "envoy.term")+"; exit 0")
		return cm, envoy
	})
	cmPid := h.waitForPid("cm.pid")
	envoyPid := h.waitForPid("envoy.pid")

	h.sigCh <- syscall.SIGTERM

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("Run exited with code %d, want 1", code)
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
// shutdown rather than being SIGKILLed. This is the race that made
// TestSupervisorSIGTERMForwarding flaky.
func TestSupervisorSIGTERMWaitsForSlowChild(t *testing.T) {
	h := newSupervisorHarness(t, func(dir string) (string, string) {
		cm := trappingChild(filepath.Join(dir, "cm.pid"), "exit 0")
		// Envoy takes 500ms to drain after SIGTERM.
		envoy := trappingChild(filepath.Join(dir, "envoy.pid"), "sleep 0.5; echo done > "+filepath.Join(dir, "envoy.done")+"; exit 0")
		return cm, envoy
	})
	h.waitForPid("cm.pid")
	h.waitForPid("envoy.pid")

	h.sigCh <- syscall.SIGTERM

	h.waitForRunExit(5 * time.Second)
	if _, err := os.Stat(h.path("envoy.done")); err != nil {
		t.Errorf("Envoy was killed before finishing graceful shutdown: %v", err)
	}
}

// TestSupervisorSIGTERMGracePeriodExpires verifies that children which ignore
// SIGTERM are SIGKILLed once ShutdownGracePeriod elapses.
func TestSupervisorSIGTERMGracePeriodExpires(t *testing.T) {
	origGrace := ShutdownGracePeriod
	ShutdownGracePeriod = 300 * time.Millisecond
	t.Cleanup(func() { ShutdownGracePeriod = origGrace })

	h := newSupervisorHarness(t, func(dir string) (string, string) {
		// ":" is a no-op, so both children ignore SIGTERM.
		return trappingChild(filepath.Join(dir, "cm.pid"), ":"),
			trappingChild(filepath.Join(dir, "envoy.pid"), ":")
	})
	cmPid := h.waitForPid("cm.pid")
	envoyPid := h.waitForPid("envoy.pid")

	h.sigCh <- syscall.SIGTERM

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("Run exited with code %d, want 1", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) was not killed after grace period", cmPid)
	}
	if !waitForProcessExit(envoyPid, 3*time.Second) {
		t.Errorf("Envoy (PID %d) was not killed after grace period", envoyPid)
	}
}

// TestSupervisorKillsEnvoyWhenConfigManagerDies verifies that when Config Manager
// terminates prematurely, Envoy is killed and Run exits with 1.
func TestSupervisorKillsEnvoyWhenConfigManagerDies(t *testing.T) {
	h := newSupervisorHarness(t, func(dir string) (string, string) {
		// ConfigManager exits after 150ms with error code 42.
		cm := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nsleep 0.15\nexit 42\n", filepath.Join(dir, "cm.pid"))
		// Envoy would run indefinitely unless killed.
		envoy := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nwhile true; do sleep 0.05; done\n", filepath.Join(dir, "envoy.pid"))
		return cm, envoy
	})
	envoyPid := h.waitForPid("envoy.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("expected Run to return 1 when CM dies, got %d", code)
	}
	if !waitForProcessExit(envoyPid, 3*time.Second) {
		t.Errorf("Envoy (PID %d) was not killed after CM died", envoyPid)
	}
}

// TestSupervisorKillsConfigManagerWhenEnvoyDies verifies that when Envoy terminates
// prematurely, Config Manager is killed and Run exits with 1.
func TestSupervisorKillsConfigManagerWhenEnvoyDies(t *testing.T) {
	h := newSupervisorHarness(t, func(dir string) (string, string) {
		// ConfigManager would run indefinitely unless killed.
		cm := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nwhile true; do sleep 0.05; done\n", filepath.Join(dir, "cm.pid"))
		// Envoy exits after 150ms with error code 42.
		envoy := fmt.Sprintf("#!/bin/sh\necho $$ > %s\nsleep 0.15\nexit 42\n", filepath.Join(dir, "envoy.pid"))
		return cm, envoy
	})
	cmPid := h.waitForPid("cm.pid")

	if code := h.waitForRunExit(5 * time.Second); code != 1 {
		t.Errorf("expected Run to return 1 when Envoy dies, got %d", code)
	}
	if !waitForProcessExit(cmPid, 3*time.Second) {
		t.Errorf("Config Manager (PID %d) was not killed after Envoy died", cmPid)
	}
}
