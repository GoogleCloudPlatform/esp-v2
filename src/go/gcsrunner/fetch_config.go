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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/api/option"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	google_oauth "golang.org/x/oauth2/google"
)

// FetchConfigOptions provides a set of configurations when fetching and writing config files.
type FetchConfigOptions struct {
	BucketName                    string
	ConfigFileName                string
	WantPort                      uint32
	ReplacePort                   uint32
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

var osCreate = func(p string) (file, error) { return os.Create(p) }

// FetchConfigFromGCS handles both fetching the config from GCS and writing it to file.
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
	updated, err := replaceListenerPort(b, opts)
	if err != nil {
		return fmt.Errorf("failed to replace listener port: %v", err)
	}
	if err := writeFile(updated, opts); err != nil {
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

func replaceListenerPort(config []byte, opts FetchConfigOptions) ([]byte, error) {
	if opts.WantPort == 0 || opts.WantPort == opts.ReplacePort {
		return config, nil
	}

	bootstrap := &bootstrappb.Bootstrap{}
	u := &jsonpb.Unmarshaler{
		AnyResolver: util.Resolver,
	}
	if err := u.Unmarshal(bytes.NewBuffer(config), bootstrap); err != nil {
		return nil, err
	}

	replaced := false
	listeners := bootstrap.GetStaticResources().GetListeners()
	if len(listeners) != 1 {
		return nil, fmt.Errorf("expected exactly 1 listener, got: %d", len(listeners))
	}

	if addr := listeners[0].GetAddress().GetSocketAddress(); addr != nil {
		portSpecifier := addr.GetPortSpecifier()
		if portValue, ok := portSpecifier.(*corepb.SocketAddress_PortValue); ok {
			if portValue.PortValue == opts.ReplacePort {
				portValue.PortValue = opts.WantPort
				replaced = true
			}
		}
	}
	if !replaced {
		return nil, fmt.Errorf("expected a listener with port value %d but got none: %v", opts.ReplacePort, bootstrap)
	}

	m := &jsonpb.Marshaler{
		OrigName:    true,
		AnyResolver: util.Resolver,
	}
	buf := &bytes.Buffer{}
	if err := m.Marshal(buf, bootstrap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
