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

package components

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

// Echo stores data for Echo HTTP/1 backend process.
type EchoHTTPServer struct {
	*Cmd
}

func NewEchoHTTPServer(port uint16, enableHttps bool, enableRootPathHandler, useWrongCert bool, mtlsCertFile string) (*EchoHTTPServer, error) {
	serverArgs := []string{
		fmt.Sprint("--alsologtostderr"),
		fmt.Sprintf("--port=%v", port),
		fmt.Sprintf("--enable_https=%v", enableHttps),
		fmt.Sprintf("--enable_root_path_handler=%v", enableRootPathHandler),
	}

	// If Backend server uses different cert as Proxy, the HTTPS call fails.
	if useWrongCert {
		serverArgs = append(serverArgs,
			fmt.Sprintf("--https_cert_path=%v", platform.GetFilePath(platform.ServerCert)),
			fmt.Sprintf("--https_key_path=%v", platform.GetFilePath(platform.ServerKey)))
	} else {
		serverArgs = append(serverArgs,
			fmt.Sprintf("--https_cert_path=%v", platform.GetFilePath(platform.ProxyCert)),
			fmt.Sprintf("--https_key_path=%v", platform.GetFilePath(platform.ProxyKey)))
	}
	if mtlsCertFile != "" {
		serverArgs = append(serverArgs,
			fmt.Sprintf("--mtls_cert_file=%v", mtlsCertFile))
	}

	cmd := exec.Command(platform.GetFilePath(platform.Echo), serverArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &EchoHTTPServer{
		Cmd: &Cmd{
			name: "EchoHttpServer",
			Cmd:  cmd,
		},
	}, nil
}
