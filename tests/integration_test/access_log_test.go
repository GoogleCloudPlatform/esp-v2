// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func lineCounter(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return uint32(lineCount), nil
}

func tryRemoveFile(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func TestAccessLog(t *testing.T) {
	t.Parallel()

	accessLog := platform.GetFilePath(platform.AccessLog)
	if err := tryRemoveFile(accessLog); err != nil {
		t.Fatalf("fail to remove accessLog file, %v", err)
	}

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed", "--access_log=" + accessLog}

	s := env.NewTestEnv(comp.TestAccessLog, platform.EchoSidecar)

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	reqCnt := uint32(10)
	for i := uint32(0); i < reqCnt; i++ {
		wantResp := `{"message":"hello"}`
		url := fmt.Sprintf("http://localhost:%v/echo?key=api-key", s.Ports().ListenerPort)
		resp, err := client.DoPost(url, "hello")

		if err != nil {
			t.Errorf("got unexpected error: %s", err)
			continue
		}
		if !strings.Contains(string(resp), wantResp) {
			t.Errorf("expected: %s, got: %s", wantResp, string(resp))
		}
	}

	s.TearDown()

	gotReqCnt, err := lineCounter(accessLog)
	if err != nil {
		t.Fatalf("fail to get line count in access file: %v", err)
	}
	if reqCnt != gotReqCnt {
		t.Errorf("expected request count: %v, got: %v", reqCnt, gotReqCnt)
	}

	if err := tryRemoveFile(accessLog); err != nil {
		t.Fatalf("fail to remove accessLog file, %v", err)
	}
}
