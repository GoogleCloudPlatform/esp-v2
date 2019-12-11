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
// `gsutil cp "gs://${BUCKET}/${CONFIG_FILE_NAME}" envoy.json`
//
// without needing `gsutil` in the image.
//
// Additionally, an optional `PORT` variable may be provided to override
// where Envoy listens to traffic. This will be used only if the original config
// specifies the port `8080`.
package main

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/gcsrunner"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/glog"
)

const (
	envoyConfigPath               = "envoy.json"
	fetchGCSObjectInitialInterval = 10 * time.Second
	fetchGCSObjectTimeout         = 5 * time.Minute
	terminateEnvoyTimeout         = time.Minute
	replaceListenerPort           = 8080
)

var (
	envoyBinaryPath = flag.String("envoy_bin_path", "bin/envoy", "Location of the Envoy binary.")
	envoyLogLevel   = flag.String("envoy_log_level", "info",
		"Envoy logging level. Default is `info`. Options are: [trace][debug][info][warning][error][critical][off]")
)

func main() {
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portNum, err := strconv.ParseUint(port, 10, 32)
	if err != nil {
		glog.Fatal("PORT variable must be a valid port number.")
	}

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

	envoyBin := os.Getenv("ENVOY_BIN_PATH")
	if envoyBin == "" {
		envoyBin = *envoyBinaryPath
	}

	metadataURL := os.Getenv("METADATA_URL")
	if metadataURL == "" {
		metadataURL = options.DefaultCommonOptions().MetadataURL
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)

	if err := gcsrunner.FetchConfigFromGCS(gcsrunner.FetchConfigOptions{
		BucketName:                    bucketName,
		ConfigFileName:                configFileName,
		WantPort:                      uint32(portNum),
		ReplacePort:                   replaceListenerPort,
		FetchGCSObjectInitialInterval: fetchGCSObjectInitialInterval,
		FetchGCSObjectTimeout:         fetchGCSObjectTimeout,
		WriteFilePath:                 envoyConfigPath,
		MetadataURL:                   metadataURL,
	}); err != nil {
		glog.Fatalf("Failed to fetch config: %v", err)
	}

	if err := gcsrunner.StartEnvoyAndWait(signalChan, gcsrunner.StartEnvoyOptions{
		BinaryPath:       envoyBin,
		ConfigPath:       envoyConfigPath,
		LogLevel:         logLevel,
		TerminateTimeout: terminateEnvoyTimeout,
	}); err != nil {
		glog.Fatalf("Envoy erred: %v", err)
	}
}
