// Copyright 2026 Google LLC
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

package startproxy

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Log to stderr instead of leaving glog files in $TMPDIR.
	_ = flag.Set("logtostderr", "true")
	os.Exit(m.Run())
}

// unsetEnvForTest unsets key for the duration of the test and restores its
// original value afterwards.
func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	_ = os.Unsetenv(key)
}

// writeServiceAccountKey writes content to a temporary key file and returns its path.
func writeServiceAccountKey(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write service account key: %v", err)
	}
	return path
}

func writeMockScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write mock script %s: %v", path, err)
	}
}

// waitForFile waits until path holds at least one complete line. Mock scripts
// write files with "echo ... > path", which creates the file before writing.
func waitForFile(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && bytes.HasSuffix(data, []byte("\n")) {
			return data, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for %s", path)
}
