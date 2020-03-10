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

package client

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
)

func RunGRPCEchoTest(testPlans string, serverPort uint16) (string, error) {
	testPlans = fmt.Sprintf("server_addr:\"127.0.0.1:%v\"\n%s", serverPort, testPlans)
	f, err := ioutil.TempFile("", "test_plans.*.txt")
	if err != nil {
		return "cannot create temporary file", err
	}
	_, err = f.WriteString(testPlans)
	defer os.Remove(f.Name())
	if err != nil {
		f.Close()
		return "", err
	}
	err = f.Close()
	if err != nil {
		return "", err
	}

	realCmd := fmt.Sprintf("%s < %s", platform.GetFilePath(platform.GrpcEchoClient), f.Name())
	cmd := exec.Command("bash", "-c", realCmd)
	out, err := cmd.CombinedOutput()

	glog.Infof("gRPC Echo output: %v", string(out))
	return string(out), err
}
