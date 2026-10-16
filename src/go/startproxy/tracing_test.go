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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestFetchProjectIDFromMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		flavor := make(chan string, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			flavor <- r.Header.Get("Metadata-Flavor")
			_, _ = w.Write([]byte("my-cloud-project\n"))
		}))
		defer srv.Close()

		if got := fetchProjectIDFromMetadata(srv.URL, time.Second); got != "my-cloud-project" {
			t.Errorf("got %q, want %q", got, "my-cloud-project")
		}
		select {
		case got := <-flavor:
			if got != "Google" {
				t.Errorf("Metadata-Flavor header = %q, want %q", got, "Google")
			}
		default:
			t.Errorf("metadata server was not called")
		}
	})

	t.Run("error status", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		defer srv.Close()

		if got := fetchProjectIDFromMetadata(srv.URL, time.Second); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("network unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()

		if got := fetchProjectIDFromMetadata(srv.URL, time.Second); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestFetchProjectIDFromServiceAccountKey(t *testing.T) {
	testcases := []struct {
		desc string
		path string
		want string
	}{
		{
			desc: "valid key",
			path: writeServiceAccountKey(t, `{"project_id": "test-sa-proj-123", "client_email": "sa@example.com"}`),
			want: "test-sa-proj-123",
		},
		{
			desc: "missing file",
			path: "/nonexistent/path/sa.json",
		},
		{
			desc: "corrupted JSON",
			path: writeServiceAccountKey(t, "not valid json {[["),
		},
		{
			desc: "missing project_id",
			path: writeServiceAccountKey(t, `{"client_email": "sa@example.com"}`),
		},
		{
			desc: "non-string project_id",
			path: writeServiceAccountKey(t, `{"project_id": 123}`),
		},
		{
			desc: "directory instead of file",
			path: t.TempDir(),
		},
		{
			desc: "empty path",
			path: "",
		},
	}
	for _, tc := range testcases {
		if got := projectIDFromServiceAccountKey(tc.path); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.desc, got, tc.want)
		}
	}
}

func TestSetupOtelResourceAttributes(t *testing.T) {
	saKeyPath := writeServiceAccountKey(t, `{"project_id": "sa-project-789"}`)
	strPtr := func(s string) *string { return &s }

	testcases := []struct {
		desc           string
		flags          []string
		initEnv        *string // nil: OTEL_RESOURCE_ATTRIBUTES is unset.
		metadataReturn string  // "": the metadata server lookup fails.
		wantEnv        *string // nil: OTEL_RESOURCE_ATTRIBUTES stays unset.
	}{
		{
			desc:    "Injects project ID from --tracing_project_id when env is empty",
			flags:   []string{"--tracing_project_id=flag-project-123"},
			wantEnv: strPtr("gcp.project_id=flag-project-123"),
		},
		{
			desc:    "Appends project ID to existing OTEL_RESOURCE_ATTRIBUTES",
			flags:   []string{"--tracing_project_id=flag-project-123"},
			initEnv: strPtr("service.name=my-svc,service.version=1.0"),
			wantEnv: strPtr("service.name=my-svc,service.version=1.0,gcp.project_id=flag-project-123"),
		},
		{
			desc:    "Does not duplicate if gcp.project_id already present in env",
			flags:   []string{"--tracing_project_id=flag-project-123"},
			initEnv: strPtr("gcp.project_id=existing-project"),
			wantEnv: strPtr("gcp.project_id=existing-project"),
		},
		{
			desc:    "Does not duplicate if legacy gcp.project.id already present in env",
			flags:   []string{"--tracing_project_id=flag-project-123"},
			initEnv: strPtr("gcp.project.id=existing-project,env=prod"),
			wantEnv: strPtr("gcp.project.id=existing-project,env=prod"),
		},
		{
			desc:           "Auto-detects from metadata server when flag is omitted on GCP",
			metadataReturn: "metadata-project-456",
			wantEnv:        strPtr("gcp.project_id=metadata-project-456"),
		},
		{
			desc:           "Skips metadata fetch when --non_gcp is set",
			flags:          []string{"--non_gcp"},
			metadataReturn: "metadata-project-456",
		},
		{
			desc:           "Skips when --disable_tracing is set",
			flags:          []string{"--disable_tracing", "--tracing_project_id=flag-project-123"},
			metadataReturn: "metadata-project-456",
		},
		{
			desc: "Leaves env unset when metadata fetch fails and no flag provided",
		},
		{
			desc:    "Resolves project ID from --service_account_key when flag is omitted",
			flags:   []string{"--service_account_key=" + saKeyPath},
			wantEnv: strPtr("gcp.project_id=sa-project-789"),
		},
		{
			desc:    "--tracing_project_id flag takes precedence over --service_account_key",
			flags:   []string{"--tracing_project_id=flag-project-123", "--service_account_key=" + saKeyPath},
			wantEnv: strPtr("gcp.project_id=flag-project-123"),
		},
		{
			desc:    "OTEL_RESOURCE_ATTRIBUTES takes precedence over --service_account_key",
			flags:   []string{"--service_account_key=" + saKeyPath},
			initEnv: strPtr("gcp.project_id=existing-env-project"),
			wantEnv: strPtr("gcp.project_id=existing-env-project"),
		},
		{
			desc:           "Falls back to metadata server when --service_account_key is invalid on GCP",
			flags:          []string{"--service_account_key=/invalid/path/key.json"},
			metadataReturn: "metadata-project-456",
			wantEnv:        strPtr("gcp.project_id=metadata-project-456"),
		},
		{
			desc:    "Resolves project ID from --service_account_key even on --non_gcp",
			flags:   []string{"--non_gcp", "--service_account_key=" + saKeyPath},
			wantEnv: strPtr("gcp.project_id=sa-project-789"),
		},
		{
			desc:           "Leaves env unset when --service_account_key is invalid on --non_gcp",
			flags:          []string{"--non_gcp", "--service_account_key=/invalid/path/key.json"},
			metadataReturn: "metadata-should-not-be-called",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			if tc.initEnv != nil {
				t.Setenv(otelResourceAttributesEnv, *tc.initEnv)
			} else {
				unsetEnvForTest(t, otelResourceAttributesEnv)
			}
			l := newLauncher()
			l.metadataProjectID = func() string { return tc.metadataReturn }

			flags := append([]string{"--service=echo", "--version=v1", "--backend=127.0.0.1:8080"}, tc.flags...)
			args, err := parseArgs(flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			l.setupOtelResourceAttributes(args)

			got, ok := os.LookupEnv(otelResourceAttributesEnv)
			if tc.wantEnv == nil {
				if ok {
					t.Errorf("%s = %q, want unset", otelResourceAttributesEnv, got)
				}
			} else if !ok || got != *tc.wantEnv {
				t.Errorf("%s = %q (set=%v), want %q", otelResourceAttributesEnv, got, ok, *tc.wantEnv)
			}
		})
	}
}
