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
	"os/signal"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	waitForEnvoyToStart = 50 * time.Millisecond
	waitForSleep        = 2 * waitForEnvoyToStart
	testTimeout         = 2 * waitForSleep
	helperCmdTimeout    = time.Second
)

type fakeCmdOptions struct {
	name         string
	exitStatus   string // a number, or empty string to leave unset
	ignoreSignal bool
}

func fakeExecCommand(opts fakeCmdOptions, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	env := []string{
		"GO_WANT_HELPER_PROCESS=1",
		fmt.Sprintf("GO_HELPER_PROCESS_TEST_NAME=%s", opts.name),
		fmt.Sprintf("GO_HELPER_PROCESS_IGNORE_SIGNAL=%v", opts.ignoreSignal),
	}
	if opts.exitStatus != "" {
		env = append(env, "GO_HELPER_PROCESS_EXIT_STATUS="+opts.exitStatus)
	}
	cmd.Env = env
	return cmd
}

func TestStartEnvoyAndWait(t *testing.T) {
	opts := StartEnvoyOptions{
		BinaryPath:       "binary",
		ConfigPath:       "config",
		LogLevel:         "loglevel",
		TerminateTimeout: testTimeout,
	}

	testCases := []struct {
		name       string
		opts       fakeCmdOptions
		sendSignal os.Signal
	}{
		{
			name: "Envoy exiting with 0 status is an error",
			opts: fakeCmdOptions{
				name:       "test exiting 0",
				exitStatus: "0",
			},
		},
		{
			name: "Envoy exiting with non-0 status is an error",
			opts: fakeCmdOptions{
				name:       "test exiting 1",
				exitStatus: "1",
			},
		},
		{
			name: "Envoy not stopping when signal sent is an error",
			opts: fakeCmdOptions{
				name:         "test ignoring signal",
				ignoreSignal: true,
			},
			sendSignal: os.Interrupt,
		},
		{
			name: "Envoy stopping when signal sent is not an error",
			opts: fakeCmdOptions{
				name: "test stopping on signal",
			},
			sendSignal: os.Kill,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t2 *testing.T) {
			execCommand = func(cmd string, args ...string) *exec.Cmd {
				return fakeExecCommand(tc.opts, cmd, args...)
			}
			defer func() { execCommand = exec.Command }()

			signalChan := make(chan os.Signal, 1)
			defer close(signalChan)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := StartEnvoyAndWait(signalChan, opts)
				if err == nil {
					t2.Errorf("StartEnvoyAndWait(chan, %v) returned nil; should never return nil", opts)
				}
			}()

			time.Sleep(waitForEnvoyToStart)
			if tc.sendSignal != nil {
				signalChan <- tc.sendSignal
			}

			wg.Wait()
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)

	testName := os.Getenv("GO_HELPER_PROCESS_TEST_NAME")

	if tmpdir := os.Getenv("TMPDIR"); tmpdir != "/tmp" {
		t.Fatalf("TMPDIR=%q, want %s", tmpdir, "/tmp")
	}

	if s := os.Getenv("GO_HELPER_PROCESS_EXIT_STATUS"); s != "" {
		sc, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("bad exit status %q: %v", s, err)
		}
		os.Exit(sc)
	}

	ignoreSignal := os.Getenv("GO_HELPER_PROCESS_IGNORE_SIGNAL")
	ignore, err := strconv.ParseBool(ignoreSignal)
	if err != nil {
		t.Fatalf("%s: bad ignore signal %q: %v", testName, ignoreSignal, err)
	}
	timeout := time.After(helperCmdTimeout)
	select {
	case sig := <-signalChan:
		t.Logf("%s: got signal %v", testName, sig)
		if !ignore {
			os.Exit(1)
		}
		time.Sleep(helperCmdTimeout)
		t.Logf("%s: correctly reached test timeout, exiting", testName)
		os.Exit(1)
	case <-timeout:
		t.Fatalf("%s: timed out waiting for signal after %v", testName, testTimeout)
	}
}
