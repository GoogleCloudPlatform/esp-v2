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
	"google.golang.org/api/option"

	google_oauth "golang.org/x/oauth2/google"
)

// FetchConfigOptions provides a set of configurations when fetching and writing config files.
type FetchConfigOptions struct {
	BucketName                    string
	ConfigFileName                string
	WriteFilePath                 string
	FetchGCSObjectInitialInterval time.Duration
	FetchGCSObjectTimeout         time.Duration
}

var findDefaultCredentials = google_oauth.FindDefaultCredentials

type getObjectRequest struct {
	Bucket, Object string
}
type storageObjectReader interface {
	readObject(ctx context.Context, req getObjectRequest) (io.Reader, error)
}

type gcsObjectReader struct {
	client *storage.Client
}

func (g *gcsObjectReader) readObject(ctx context.Context, req getObjectRequest) (io.Reader, error) {
	return g.client.Bucket(req.Bucket).Object(req.Object).NewReader(ctx)
}

var newStorageObjectReader = func(ctx context.Context, opts ...option.ClientOption) (storageObjectReader, error) {
	c, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &gcsObjectReader{c}, nil
}

type file interface {
	io.Writer
	io.Closer
}

var (
	osCreate = func(p string) (file, error) { return os.Create(p) }
)

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
		return fmt.Errorf("failed to get reader for object: %v", err)
	}
	if err := writeFile(b, opts); err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}
	return nil
}

func readBytes(opts FetchConfigOptions) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opts.FetchGCSObjectTimeout)
	defer cancel()

	creds, err := findDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %v", err)
	}
	client, err := newStorageObjectReader(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage client: %v", err)
	}
	ebo := backoff.NewExponentialBackOff()
	ebo.InitialInterval = opts.FetchGCSObjectInitialInterval
	var reader io.Reader
	var retryErr error
	op := func() error {
		r, err := client.readObject(ctx, getObjectRequest{
			Bucket: opts.BucketName,
			Object: opts.ConfigFileName,
		})
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

	return ioutil.ReadAll(reader)
}

func writeFile(b []byte, opts FetchConfigOptions) error {
	file, err := osCreate(opts.WriteFilePath)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}

	if _, err := file.Write(b); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}
