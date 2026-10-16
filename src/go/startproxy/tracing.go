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
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// otelResourceAttributesEnv is the OpenTelemetry resource attributes environment variable.
	otelResourceAttributesEnv = "OTEL_RESOURCE_ATTRIBUTES"

	// metadataProjectIDURL is the GCP metadata server URL for the project ID.
	metadataProjectIDURL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"

	// metadataFetchTimeout bounds the project ID lookup on the metadata server.
	metadataFetchTimeout = 2 * time.Second
)

// fetchProjectIDFromMetadata fetches the GCP project ID from the GCP metadata
// server at url. It returns "" if the project ID cannot be determined.
func fetchProjectIDFromMetadata(url string, timeout time.Duration) string {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// projectIDFromServiceAccountKey extracts project_id from a Google service
// account JSON key file. It returns "" if the path is empty, the file is
// missing or unreadable, the JSON is invalid, or project_id is not a string.
func projectIDFromServiceAccountKey(keyPath string) string {
	if keyPath == "" {
		return ""
	}
	if info, err := os.Stat(keyPath); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return ""
	}
	var key map[string]interface{}
	if err := json.Unmarshal(data, &key); err != nil {
		return ""
	}
	projectID, _ := key["project_id"].(string)
	return strings.TrimSpace(projectID)
}

// otelAttrsHaveProjectID reports whether a comma-separated
// OTEL_RESOURCE_ATTRIBUTES value contains gcp.project_id or the legacy
// gcp.project.id attribute.
func otelAttrsHaveProjectID(attrs string) bool {
	for _, attr := range strings.Split(attrs, ",") {
		attr = strings.TrimSpace(attr)
		if strings.HasPrefix(attr, "gcp.project_id=") || strings.HasPrefix(attr, "gcp.project.id=") {
			return true
		}
	}
	return false
}

// serviceAccountKeyPath returns --service_account_key if set, otherwise the
// GOOGLE_APPLICATION_CREDENTIALS environment variable.
func serviceAccountKeyPath(args *proxyArgs) string {
	if args.ServiceAccountKey != nil && *args.ServiceAccountKey != "" {
		return *args.ServiceAccountKey
	}
	return os.Getenv(googleCredsEnv)
}

// setupOtelResourceAttributes auto-injects gcp.project_id into
// OTEL_RESOURCE_ATTRIBUTES (inherited by Config Manager and Envoy) when tracing
// is enabled and it is not already configured. The project ID is resolved with
// the following precedence:
//  1. OTEL_RESOURCE_ATTRIBUTES already contains gcp.project_id or gcp.project.id
//  2. --tracing_project_id
//  3. --service_account_key / GOOGLE_APPLICATION_CREDENTIALS JSON key file
//  4. GCP metadata server (skipped with --non_gcp)
//
// args.TracingProjectID is set to the resolved project ID if it was empty.
func (l *launcher) setupOtelResourceAttributes(args *proxyArgs) {
	if args.DisableTracing {
		return
	}

	existing := strings.TrimSpace(os.Getenv(otelResourceAttributesEnv))
	if otelAttrsHaveProjectID(existing) {
		return
	}

	projectID := args.TracingProjectID
	if projectID == "" {
		projectID = projectIDFromServiceAccountKey(serviceAccountKeyPath(args))
	}
	if projectID == "" && !args.NonGcp {
		projectID = l.metadataProjectID()
	}
	if projectID == "" {
		return
	}

	if args.TracingProjectID == "" {
		args.TracingProjectID = projectID
	}

	attrs := "gcp.project_id=" + projectID
	if existing != "" {
		attrs = existing + "," + attrs
	}
	_ = os.Setenv(otelResourceAttributesEnv, attrs)
}
