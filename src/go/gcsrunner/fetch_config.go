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

// Package gcsrunner contains helper functions to support the GCS Runner.
// See gcsrunner/main/runner.go for more complete details
package gcsrunner

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

var (
	// can be overwritten in unit tests
	newGCS             = newGCSClient
	newDefaultCredsGCS = newDefaultCredsGCSClient
	osCreate           = func(p string) (io.WriteCloser, error) { return os.Create(p) }
)

// can be mocked in unit tests
type gcsReader interface {
	Reader(ctx context.Context, bucket, object string) (io.Reader, error)
}

// Implements a minimal GCS client for reading files from GCS.
type gcsClient struct {
	client *storage.Client
}

func (c *gcsClient) Reader(ctx context.Context, bucket, object string) (io.Reader, error) {
	return c.client.Bucket(bucket).Object(object).NewReader(ctx)
}

// FetchConfigOptions provides a set of configurations when fetching and writing config files.
type FetchConfigOptions struct {
	// ServiceAccount, if provided, is impersonated in the GCS fetch call.
	// If left blank, the default credentials are used.
	ServiceAccount                string
	BucketName                    string
	ConfigFileName                string
	WriteFilePath                 string
	FetchGCSObjectInitialInterval time.Duration
	FetchGCSObjectTimeout         time.Duration
}

func newGCSClient(ctx context.Context, sa string) (gcsReader, error) {
	ts, err := TokenSource(ctx, sa)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	client, err := storage.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		glog.Errorf("error creating new GCS client with token source: %v", err)
		return nil, err
	}
	glog.Infof("created new GCS client with token source in %s", time.Since(start))
	return &gcsClient{client}, nil
}
func newDefaultCredsGCSClient(ctx context.Context) (gcsReader, error) {
	start := time.Now()
	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		glog.Errorf("error finding default credentials: %v", err)
		return nil, err
	}
	glog.Infof("found default credentials in %s", time.Since(start))

	start = time.Now()
	c, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		glog.Errorf("error creating new GCS client with default credentials: %v", err)
		return nil, err
	}
	glog.Infof("created new GCS client with default credentials in %s", time.Since(start))
	return &gcsClient{c}, nil
}

// FetchConfigFromGCS handles fetching a config from GCS, applying any transformation,
// and writing it to file.
//
// Fetching the GCS object is retried either until success or until
// `opts.fetchGCSObjectTimeout` has passed.
//
// Note that writing to file does not time out.
func FetchConfigFromGCS(opts FetchConfigOptions) error {
	b, err := readBytes(opts)
	if err != nil {
		return fmt.Errorf("failed to read object: %v", err)
	}
	if err := writeFile(b, opts); err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}
	return nil
}

func readBytes(opts FetchConfigOptions) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opts.FetchGCSObjectTimeout)
	defer cancel()

	var client gcsReader

	ebo := backoff.NewExponentialBackOff()
	ebo.InitialInterval = opts.FetchGCSObjectInitialInterval
	var out []byte
	op := func() error {
		if err := ctx.Err(); err != nil {
			return backoff.Permanent(err)
		}

		if client == nil {
			var err error
			if opts.ServiceAccount == "" {
				client, err = newDefaultCredsGCS(ctx)
				if err != nil {
					glog.Errorf("error getting default creds GCS client (retrying): %v", err)
					return err
				}
			} else {
				client, err = newGCS(ctx, opts.ServiceAccount)
				if err != nil {
					glog.Errorf("error getting GCS client using service account (retrying): %v", err)
					return err
				}
			}
		}

		start := time.Now()
		r, err := client.Reader(ctx, opts.BucketName, opts.ConfigFileName)
		if err != nil {
			glog.Errorf("error getting reader for object (retrying): %v", err)
			return err
		}
		glog.Infof("obtained reader for object in %s", time.Since(start))

		if out, err = ioutil.ReadAll(r); err != nil {
			glog.Errorf("error reading object bytes (retrying): %v", err)
			return err
		}
		return nil
	}

	if err := backoff.Retry(op, ebo); err != nil {
		return nil, err
	}
	return out, nil
}

func writeFile(b []byte, opts FetchConfigOptions) error {
	file, err := osCreate(opts.WriteFilePath)
	defer file.Close()
	if err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	return nil
}
