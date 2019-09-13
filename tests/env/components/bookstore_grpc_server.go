// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"cloudesf.googlesource.com/gcpproxy/tests/env/platform"
)

type BookstoreGrpcServer struct {
	*Cmd
	grpcPort uint16
}

func NewBookstoreGrpcServer(port uint16) (*BookstoreGrpcServer, error) {
	cmd := exec.Command("node", platform.GetFilePath(platform.GrpcBookstore), strconv.Itoa(int(port)))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &BookstoreGrpcServer{
		Cmd: &Cmd{
			name: "BookstoreGrpcServer",
			Cmd:  cmd,
		},
		grpcPort: port,
	}, nil
}

func (s BookstoreGrpcServer) String() string {
	return "Nodejs Bookstore gRPC Server"
}

func (s BookstoreGrpcServer) CheckHealth() error {
	opts := NewHealthCheckOptions()
	addr := fmt.Sprintf("localhost:%v", s.grpcPort)
	return GrpcHealthCheck(addr, opts)
}
