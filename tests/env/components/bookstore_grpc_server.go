// Copyright 2018 Google Cloud Platform Proxy Authors
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

package components

import (
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

const (
	bookstorePath = "../endpoints/bookstore-grpc/grpc_server.js"
)

type BookstoreGrpcServer struct {
	cmd *exec.Cmd
}

func NewBookstoreGrpcServer() (*BookstoreGrpcServer, error) {
	cmd := exec.Command("node", bookstorePath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &BookstoreGrpcServer{
		cmd: cmd,
	}, nil
}

// Start starts the Bookstore process.
func (b *BookstoreGrpcServer) Start() <-chan error {
	errCh := make(chan error)
	go func() {
		err := b.cmd.Start()
		if err != nil {
			errCh <- err
		}
	}()

	// wait for grpc server up.
	time.AfterFunc(1*time.Second, func() { close(errCh) })
	return errCh
}

// Stop stops the Bookstore process.
func (b *BookstoreGrpcServer) Stop() error {
	glog.Infof("Stop xDS server...")

	done := make(chan error, 1)
	go func() {
		done <- b.cmd.Wait()
	}()

	select {
	case <-time.After(testEnvTTL):
		glog.Infof("BookstoreGrpcServer killed as timeout reached")
		if err := b.cmd.Process.Kill(); err != nil {
			return err
		}
	case err := <-done:
		glog.Infof("stop BookstoreGrpcServer ... done\n")
		return err
	}
	return nil
}
