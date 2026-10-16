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

// Package startproxy implements the ESPv2 startup and process supervision logic,
// replacing docker/generic/start_proxy.py and docker/serverless/env_start_proxy.py.
package startproxy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
)

// launcher holds the settings Run takes from its environment. Production uses
// newLauncher(); tests override fields instead of mutating package globals.
type launcher struct {
	// bootstrapBin generates the Envoy bootstrap config.
	bootstrapBin string
	// configManagerBin is the Config Manager binary.
	configManagerBin string
	// envoyBin is the Envoy binary.
	envoyBin string
	// certDir is where --generate_self_signed_cert writes server.crt/server.key.
	certDir string
	// shutdownGrace is how long run waits for both child processes to exit
	// after forwarding SIGTERM/SIGINT before SIGKILLing any survivors.
	shutdownGrace time.Duration
	// metadataProjectID returns the GCP project ID from the metadata server,
	// or "" if it is unavailable.
	metadataProjectID func() string
	// notifySignals registers c to receive shutdown signals.
	notifySignals func(c chan<- os.Signal)
	// stopSignals undoes notifySignals.
	stopSignals func(c chan<- os.Signal)
}

func newLauncher() *launcher {
	return &launcher{
		bootstrapBin:     "bin/bootstrap",
		configManagerBin: "bin/configmanager",
		envoyBin:         "bin/envoy",
		certDir:          defaultSelfSignedCertDir,
		shutdownGrace:    10 * time.Second,
		metadataProjectID: func() string {
			return fetchProjectIDFromMetadata(metadataProjectIDURL, metadataFetchTimeout)
		},
		notifySignals: func(c chan<- os.Signal) {
			signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
		},
		stopSignals: func(c chan<- os.Signal) {
			signal.Stop(c)
		},
	}
}

// Run executes the ESPv2 startup and supervision loop with argv (excluding
// the program name) and returns the process exit code.
func Run(argv []string) int {
	return newLauncher().run(argv)
}

func (l *launcher) run(argv []string) int {
	if len(argv) > 0 && argv[0] == serverlessModeArg {
		return l.runServerless(argv[1:])
	}
	return l.start(argv)
}

// runServerless starts ESPv2 configured by environment variables, like
// env_start_proxy.py. Like env_start_proxy.py, it ignores container arguments.
func (l *launcher) runServerless(ignored []string) int {
	if len(ignored) > 0 {
		glog.Warningf("Ignoring container arguments in serverless mode, set ESPv2_ARGS instead: %q", ignored)
	}
	flags, err := serverlessFlags()
	if err != nil {
		var se *serverlessError
		if errors.As(err, &se) && se.port != "" {
			if se.warning {
				glog.Warning(se.msg)
			} else {
				glog.Error(se.msg)
			}
			return l.serveErrorPage(se.port, se.msg)
		}
		glog.Error(err)
		return 1
	}
	return l.start(flags)
}

// start parses argv, then starts and supervises Config Manager and Envoy.
func (l *launcher) start(argv []string) int {
	args, err := parseArgs(argv)
	if err != nil {
		p := newArgParser(flagDefs(defaultArgs()))
		if errors.Is(err, errHelpRequested) {
			fmt.Print(p.formatHelp(helpWidth()))
			return 0
		}
		// Like start_proxy.py's ArgumentParser.error(): the full help, then
		// the error, on stderr, and exit status 1.
		fmt.Fprint(os.Stderr, p.formatHelp(helpWidth()))
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", progName, err)
		return 1
	}

	// Resolve the tracing project ID and export it via OTEL_RESOURCE_ATTRIBUTES
	// so Config Manager and Envoy inherit it.
	l.setupOtelResourceAttributes(args)

	proxyConf, err := l.genProxyConfig(args)
	if err != nil {
		glog.Error(err)
		return 1
	}

	// Register for shutdown signals before starting any child so a signal that
	// arrives during startup is forwarded instead of killing startproxy with
	// the default action and orphaning the children.
	sigCh := make(chan os.Signal, 2)
	l.notifySignals(sigCh)
	defer l.stopSignals(sigCh)

	glog.Infof("Starting Config Manager with args: %q", proxyConf)
	cm, err := startChild("Config Manager", "Config Manager", proxyConf)
	if err != nil {
		glog.Errorf("Failed to start Config Manager: %v", err)
		return 1
	}

	bootstrapArgs := l.genBootstrapConf(args)
	glog.Infof("Generating Envoy bootstrap config with args: %q", bootstrapArgs)
	bootstrap := exec.Command(bootstrapArgs[0], bootstrapArgs[1:]...)
	bootstrap.Stdout = os.Stdout
	bootstrap.Stderr = os.Stderr
	if err := bootstrap.Run(); err != nil {
		glog.Errorf("Failed to generate the Envoy bootstrap config: %v", err)
		cm.kill()
		return 1
	}

	envoyArgs := l.genEnvoyArgs(args)
	glog.Infof("Starting Envoy with args: %q", envoyArgs)
	envoy, err := startChild("Envoy", "envoy", envoyArgs)
	if err != nil {
		glog.Errorf("Failed to start Envoy: %v", err)
		cm.kill()
		return 1
	}

	return l.supervise(cm, envoy, sigCh)
}
