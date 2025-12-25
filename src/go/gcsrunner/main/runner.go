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

// The runner for gcsrunner fetches an Envoy Config from GCS and writes it to a local file.
// It then starts Envoy using that new config. If this process receives
// a signal, it will be forward to the Envoy process. It is expected that this
// signal is intended to stop the process.
//
// Two environment variables are required: `BUCKET` and `CONFIG_FILE_NAME`.
// Fetches from the bucket at `BUCKET` at the path `CONFIG_FILE_NAME`,
// Behavior is similar to running:
//
// `gcloud storage cp "gs://${BUCKET}/${CONFIG_FILE_NAME}" envoy.json`
//
// without needing `gsutil` in the image.
//
// If RUN_AS_SERVICE_ACCOUNT is provided, gcsrunner will attempt to impersonate
// the given service account in the call to read from GCS.
package main

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/gcsrunner"
	"github.com/golang/glog"
)

const (
	envoyConfigPath               = "envoy.json"
	fetchGCSObjectInitialInterval = 100 * time.Millisecond
	fetchGCSObjectTimeout         = 3 * time.Minute
	terminateEnvoyTimeout         = time.Minute
)

var (
	envoyBinaryPath = flag.String("envoy_bin_path", "bin/envoy", "Location of the Envoy binary.")
	envoyLogLevel   = flag.String("envoy_log_level", "info",
		"Envoy logging level. Default is `info`. Options are: [trace][debug][info][warning][error][critical][off]")
	envoyLogPath = flag.String("envoy_log_path", "",
		"Envoy application logging path. Default is to write to stderr.")
	envoyComponentLogLevel = flag.String("envoy_component_log_level", "", "Mapping for Envoy log level by component.")
	sa                     = flag.String("run_as_service_account", "", "If provided, use this account when fetching the config from GCS. If not provided, the container's default credentials are used.")
)

func main() {
	flag.Set("alsologtostderr", "true")
	flag.Parse()

	bucketName := os.Getenv("BUCKET")
	if bucketName == "" {
		glog.Fatal("Must specify the BUCKET environment variable.")
	}

	configFileName := os.Getenv("CONFIG_FILE_NAME")
	if configFileName == "" {
		glog.Fatal("Must specify the CONFIG_FILE_NAME environment variable.")
	}

	logLevel := os.Getenv("ENVOY_LOG_LEVEL")
	if logLevel == "" {
		logLevel = *envoyLogLevel
	}

	componentLogLevel := os.Getenv("ENVOY_COMPONENT_LOG_LEVEL")
	if componentLogLevel == "" {
		componentLogLevel = *envoyComponentLogLevel
	}

	logPath := os.Getenv("ENVOY_LOG_PATH")
	if logPath == "" {
		logPath = *envoyLogPath
	}

	envoyBin := os.Getenv("ENVOY_BIN_PATH")
	if envoyBin == "" {
		envoyBin = *envoyBinaryPath
	}

	runAsSA := os.Getenv("RUN_AS_SERVICE_ACCOUNT")
	if runAsSA == "" {
		runAsSA = *sa
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	start := time.Now()
	if err := gcsrunner.FetchConfigFromGCS(gcsrunner.FetchConfigOptions{
		BucketName:                    bucketName,
		ConfigFileName:                configFileName,
		FetchGCSObjectInitialInterval: fetchGCSObjectInitialInterval,
		FetchGCSObjectTimeout:         fetchGCSObjectTimeout,
		WriteFilePath:                 envoyConfigPath,
		ServiceAccount:                runAsSA,
	}); err != nil {
		glog.Fatalf("Failed to fetch config: %v", err)
	}
	glog.Infof("fetched config from GCS in %s", time.Since(start))

	if err := gcsrunner.StartEnvoyAndWait(signalChan, gcsrunner.StartEnvoyOptions{
		BinaryPath:        envoyBin,
		ComponentLogLevel: componentLogLevel,
		ConfigPath:        envoyConfigPath,
		LogLevel:          logLevel,
		LogPath:           logPath,
		TerminateTimeout:  terminateEnvoyTimeout,
	}); err != nil {
		glog.Fatalf("Envoy erred: %v", err)
	}
}

func envNum(v string, defaultVal uint32) (uint32, error) {
	p := os.Getenv(v)
	if p == "" {
		return defaultVal, nil
	}
	num, err := strconv.ParseUint(p, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(num), nil
}
