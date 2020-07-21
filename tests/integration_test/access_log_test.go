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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func tryRemoveFile(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func makeOneRequest(t *testing.T, s *env.TestEnv) {
	wantResp := `{"message":"hello"}`
	url := fmt.Sprintf("http://localhost:%v/echo?key=test-api-key", s.Ports().ListenerPort)
	resp, err := client.DoPost(url, "hello")

	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
		return
	}
	if !strings.Contains(string(resp), wantResp) {
		t.Errorf("expected: %s, got: %s", wantResp, string(resp))
	}
}

func TestAccessLog(t *testing.T) {
	t.Parallel()

	accessLog := platform.GetFilePath(platform.AccessLog)
	if err := tryRemoveFile(accessLog); err != nil {
		t.Fatalf("fail to remove accessLog file, %v", err)
	}

	// For the detailed format grammar, refer to
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#command-operators
	accessLogFormat := "\"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\"" +
		"%RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT%" +
		"\"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\"" +
		"\"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\" " +
		"%FILTER_STATE(com.google.espv2.filters.http.path_matcher.operation):60% " +
		"%FILTER_STATE(com.google.espv2.filters.http.service_control.api_key):30%\n"

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed", "--access_log=" + accessLog, "--access_log_format=" + accessLogFormat}

	s := env.NewTestEnv(comp.TestAccessLog, platform.EchoSidecar)

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	makeOneRequest(t, s)
	s.TearDown(t)
	expectAccessLog := fmt.Sprintf("\"POST /echo?key=test-api-key HTTP/1.1\"200"+
		" - 20 19\"-\" \"Go-http-client/1.1\"\"localhost:%v\" \"127.0.0.1:%v\" "+
		"\"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo\" \"test-api-key\"\n",
		s.Ports().ListenerPort, s.Ports().BackendServerPort)

	bytes, err := ioutil.ReadFile(accessLog)
	if err != nil {
		t.Fatalf("fail to read access log file: %v", err)
	}

	if gotAccessLog := string(bytes); expectAccessLog != gotAccessLog {
		t.Errorf("expect access log: %s, get acccess log: %v", expectAccessLog, gotAccessLog)
	}

	if err := tryRemoveFile(accessLog); err != nil {
		t.Fatalf("fail to remove accessLog file, %v", err)
	}
}
