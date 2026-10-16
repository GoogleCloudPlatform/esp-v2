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

package access_log_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

const (
	// Envoy writes the access log asynchronously, possibly after the client has
	// received the response, and flushes it to the file at least every 10s
	// (--file-flush-interval-msec). Wait a bit longer than that.
	accessLogWaitTimeout  = 15 * time.Second
	accessLogPollInterval = 100 * time.Millisecond
)

func tryRemoveFile(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// readAccessLog polls the access log file until its content is want, and
// returns the last content read.
func readAccessLog(path, want string) (string, error) {
	deadline := time.Now().Add(accessLogWaitTimeout)
	for {
		bytes, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		got := string(bytes)
		// The file is only appended to, so stop early if it can no longer match.
		if got == want || !strings.HasPrefix(want, got) {
			return got, nil
		}
		if time.Now().After(deadline) {
			return got, err
		}
		time.Sleep(accessLogPollInterval)
	}
}

func makeOneRequest(t *testing.T, s *env.TestEnv, path, wantError string) {
	url := fmt.Sprintf("http://%v:%v%s?key=test-api-key", platform.GetLoopbackAddress(), s.Ports().ListenerPort, path)
	_, err := client.DoGet(url)

	if err != nil {
		if wantError == "" {
			t.Errorf("got unexpected error: %s", err)
		} else if !strings.Contains(err.Error(), wantError) {
			t.Errorf("expected error: %s, got: %s", wantError, err.Error())
		}

		return
	}
}

func TestAccessLog(t *testing.T) {
	t.Parallel()

	accessLogFilePath := platform.GetFilePath(platform.AccessLog)
	if err := tryRemoveFile(accessLogFilePath); err != nil {
		t.Fatalf("fail to remove accessLogFile, %v", err)
	}

	// For the detailed format grammar, refer to
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#command-operators
	accessLogFormat := "\"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\"" +
		"%RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT%" +
		"\"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" " +
		"%FILTER_STATE(com.google.espv2.filters.http.service_control.api_method):70% " +
		"%FILTER_STATE(com.google.espv2.filters.http.service_control.api_key):30%" +
		"\n"

	testCases := []struct {
		desc          string
		requestPath   string
		wantError     string
		wantAccessLog string
	}{
		{
			desc:        "successful request",
			requestPath: "/echoHeader",
			wantAccessLog: "\"GET /echoHeader?key=test-api-key HTTP/1.1\"200" +
				" - 0 0\"-\" \"Go-http-client/1.1\" " +
				"\"1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader\" \"test-api-key\"\n",
		},
		{
			desc:        "request failed in path matcher",
			requestPath: "/noexistpath",
			wantError:   `http response status is not 200 OK: 404 Not Found`,
			wantAccessLog: "\"GET /noexistpath?key=test-api-key HTTP/1.1\"404" +
				" - 0 73\"-\" \"Go-http-client/1.1\" " +
				"\"\\u003cUnknown Operation Name\\u003e\" \"test-api-key\"\n",
		},
	}

	for _, tc := range testCases {
		_t := func() {
			configID := "test-config-id"
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed", "--access_log=" + accessLogFilePath, "--access_log_format=" + accessLogFormat}

			s := env.NewTestEnv(platform.TestAccessLog, platform.EchoSidecar)

			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			defer s.TearDown(t)
			makeOneRequest(t, s, tc.requestPath, tc.wantError)

			gotAccessLog, err := readAccessLog(accessLogFilePath, tc.wantAccessLog)
			if err != nil {
				t.Fatalf("fail to read access log file: %v", err)
			}

			if tc.wantAccessLog != gotAccessLog {
				t.Errorf("expect access log: %s, get acccess log: %v", tc.wantAccessLog, gotAccessLog)
			}

			if err := tryRemoveFile(accessLogFilePath); err != nil {
				t.Fatalf("fail to remove accessLogFile, %v", err)
			}
		}

		_t()
	}
}
