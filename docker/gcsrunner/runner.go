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

// This package fetches an Envoy Config from GCS and writes it to a local file.
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
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"google.golang.org/api/option"

	google_oauth "golang.org/x/oauth2/google"
)

const (
	envoyConfigPath               = "envoy.json"
	fetchGCSObjectInitialInterval = 10 * time.Second
	fetchGCSObjectTimeout         = 5 * time.Minute
)

var (
	envoyBinaryPath = flag.String("envoy_bin_path", "apiproxy/envoy", "Location of the Envoy binary.")
	envoyLogLevel   = flag.String("envoy_log_level", "info",
		"Envoy logging level. Default is `info`. Options are: [trace][debug][info][warning][error][critical][off]")

	// Watches signals to the main thread to pass on to Envoy
	signalChan chan os.Signal
)

func main() {
	flag.Parse()

	bucketName := os.Getenv("BUCKET")
	if bucketName == "" {
		glog.Fatal("Must specify the BUCKET environment variable.")
	}

	configFileName := os.Getenv("CONFIG_FILE_NAME")
	if configFileName == "" {
		glog.Fatal("Must specify the CONFIG_FILE_NAME environment variable.")
	}

	signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan)

	if err := fetchConfigFromGCS(bucketName, configFileName); err != nil {
		glog.Fatalf("Failed to fetch config: %v", err)
	}

	startEnvoyAndWait()
}

func fetchConfigFromGCS(bucketName, configFileName string) error {
	r, err := getGCSReader(bucketName, configFileName)
	if err != nil {
		return fmt.Errorf("failed to get reader for object: %v", err)
	}

	if err := writeFile(r); err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}

	return nil
}

func getGCSReader(bucketName, configFileName string) (io.Reader, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchGCSObjectTimeout)
	defer cancel()

	creds, err := google_oauth.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %v", err)
	}

	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage client: %v", err)
	}

	object := client.Bucket(bucketName).Object(configFileName)

	ebo := backoff.NewExponentialBackOff()
	ebo.InitialInterval = fetchGCSObjectInitialInterval

	var reader io.Reader
	var retryErr error
	op := func() error {
		r, err := object.NewReader(ctx)
		if err == context.DeadlineExceeded {
			retryErr = err
			// return nil to end the backoff
			return nil
		}
		if err != nil {
			glog.Errorf("error getting reader for object (retrying): %v", err)
			return err
		}
		reader = r
		return nil
	}

	if err := backoff.Retry(op, ebo); err != nil {
		return nil, err
	}

	if retryErr != nil {
		return nil, retryErr
	}

	return reader, nil
}

func writeFile(rc io.Reader) error {
	file, err := os.Create(envoyConfigPath)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}

	if _, err := io.Copy(file, rc); err != nil {
		return fmt.Errorf("failed to write bytes to file: %v", err)
	}

	return nil
}

func startEnvoyAndWait() {
	cmd := exec.Command(
		*envoyBinaryPath,
		"--config-path", envoyConfigPath,
		"--log-level", *envoyLogLevel)
	cmd.Env = []string{
		"TMPDIR=/tmp",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start Envoy: %v", err)
	}

	envoyExitChan := make(chan error)
	go func() {
		err := cmd.Wait()
		if err == nil {
			err = errors.New("unexpectedly exited OK from Envoy, which should never happen")
		}

		envoyExitChan <- err
	}()

	select {
	case err := <-envoyExitChan:
		log.Fatalf("Envoy exited unexpectedly: %v", err)
	case sig := <-signalChan:
		if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
			glog.Errorf("Stopping Envoy due to signal: %v", sig)

			// This will always be a signal to stop the process.
			if err := cmd.Process.Signal(sig); err != nil {
				glog.Fatalf("Failed to signal Envoy: %v", err)
			}

			// The cluster will shut off the container. No need to impose another deadline in the code.
			cmd.Wait()
		}
	}
}
