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
	"errors"
	"strings"
	"testing"
)

func TestHelpFlag(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		_, err := parseArgs([]string{flag})
		if !errors.Is(err, errHelpRequested) {
			t.Errorf("expected errHelpRequested for flag %q, got: %v", flag, err)
		}

		if exitCode := Run([]string{flag}); exitCode != 0 {
			t.Errorf("expected Run(%q) to return 0, got %d", flag, exitCode)
		}
	}
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "usage: startproxy [-h]") {
		t.Errorf("expected usage line, got:\n%s", out)
	}
	if !strings.Contains(out, "--service") {
		t.Errorf("expected --service in usage, got:\n%s", out)
	}
	if !strings.Contains(out, "--help") {
		t.Errorf("expected --help in usage, got:\n%s", out)
	}
}
