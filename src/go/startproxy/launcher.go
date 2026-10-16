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
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
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

// run executes the ESPv2 startup and supervision loop.
func (l *launcher) run(argv []string) int {
	// Check if invoked in serverless mode via --serverless flag or binary name.
	if len(argv) > 0 && argv[0] == "--serverless" {
		flags, port, serveErrPage, err := serverlessFlags()
		if err != nil {
			if serveErrPage && port != "" {
				log.Println(err.Error())
				if serveErr := serveErrorPage(port, err.Error()); serveErr != nil {
					log.Printf("Failed to serve error message: %v", serveErr)
				}
				return 1
			}
			log.Printf("Serverless startup error: %v", err)
			return 1
		}
		argv = append(flags, argv[1:]...)
	}

	args, err := parseArgs(argv)
	if err != nil {
		if errors.Is(err, errHelpRequested) {
			printUsage(os.Stdout)
			return 0
		}
		fmt.Fprintf(os.Stderr, "startproxy: error: %v\n", err)
		return 1
	}

	if args.ServerlessEntrypoint {
		flags, port, serveErrPage, err := serverlessFlags()
		if err != nil {
			if serveErrPage && port != "" {
				log.Println(err.Error())
				if serveErr := serveErrorPage(port, err.Error()); serveErr != nil {
					log.Printf("Failed to serve error message: %v", serveErr)
				}
				return 1
			}
			log.Printf("Serverless startup error: %v", err)
			return 1
		}
		args, err = parseArgs(flags)
		if err != nil {
			if errors.Is(err, errHelpRequested) {
				printUsage(os.Stdout)
				return 0
			}
			fmt.Fprintf(os.Stderr, "startproxy: error: %v\n", err)
			return 1
		}
	}

	// Resolve the tracing project ID and export it via OTEL_RESOURCE_ATTRIBUTES
	// so Config Manager and Envoy inherit it.
	l.setupOtelResourceAttributes(args)

	proxyConf, err := l.genProxyConfig(args)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return 1
	}

	// Register for shutdown signals before starting any child so a signal that
	// arrives during startup is forwarded instead of killing startproxy with
	// the default action and orphaning the children.
	sigCh := make(chan os.Signal, 2)
	l.notifySignals(sigCh)
	defer l.stopSignals(sigCh)

	fmt.Printf("Starting Config Manager with args: %v\n", proxyConf)
	cmCmd, cmWaitCh, err := startSubprocess(proxyConf)
	if err != nil {
		log.Printf("Failed to start Config Manager: %v", err)
		return 1
	}

	bootstrapCmdArgs := l.genBootstrapConf(args)
	fmt.Printf("%v\n", bootstrapCmdArgs)
	bootstrapCmd := exec.Command(bootstrapCmdArgs[0], bootstrapCmdArgs[1:]...)
	bootstrapCmd.Stdout = os.Stdout
	bootstrapCmd.Stderr = os.Stderr
	_ = bootstrapCmd.Run()

	envoyArgs := l.genEnvoyArgs(args)
	fmt.Printf("Starting Envoy with args: %v\n", envoyArgs)
	envoyCmd, envoyWaitCh, err := startSubprocess(envoyArgs)
	if err != nil {
		log.Printf("Failed to start Envoy: %v", err)
		if cmCmd.Process != nil {
			_ = cmCmd.Process.Kill()
		}
		return 1
	}

	return l.supervise(
		&child{name: "Config Manager", cmd: cmCmd, waitCh: cmWaitCh},
		&child{name: "Envoy", cmd: envoyCmd, waitCh: envoyWaitCh},
		sigCh,
	)
}
