// Copyright 2026 Google LLC
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

// Package apigw_dataplane_query_test verifies API Gateway data-plane query
// string forwarding, including duplicate keys, array syntax, injection
// payloads, null bytes, and overlong query URIs.
package apigw_dataplane_query_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

var sharedEnv *env.TestEnv

// TestMain runs the test suite and tears down the shared test environment.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.TearDown(nil)
	}
	os.Exit(code)
}

// TestDuplicateQueryKeys verifies that repeated query keys preserve all values
// in order.
func TestDuplicateQueryKeys(t *testing.T) {
	s := setupDataplaneQueryEnv(t)
	const reqPath = "/v1/query?filter=first&filter=second&filter=third&sort=asc&sort=desc"

	_, status, body, err := utils.DoHTTP(t.Context(), s.ListenerURL(reqPath), http.MethodGet, "", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET %s status = %d, err = %v, want %d (body: %q)", reqPath, status, err, http.StatusOK, body)
	}

	u, err := parseEchoRequestURI(body)
	if err != nil {
		t.Fatalf("GET %s parseEchoRequestURI failed: %v", reqPath, err)
	}

	gotFilters := u.Query()["filter"]
	wantFilters := []string{"first", "second", "third"}
	if !slices.Equal(gotFilters, wantFilters) {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "filter", gotFilters, wantFilters)
	}

	gotSort := u.Query()["sort"]
	wantSort := []string{"asc", "desc"}
	if !slices.Equal(gotSort, wantSort) {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "sort", gotSort, wantSort)
	}
}

// TestArraySyntaxQueryKeys verifies that bracketed and comma-separated query
// keys are forwarded verbatim.
func TestArraySyntaxQueryKeys(t *testing.T) {
	s := setupDataplaneQueryEnv(t)
	const reqPath = "/v1/query?ids[]=1&ids[]=2&items[0]=alpha&items[1]=beta&tags=first,second"

	_, status, body, err := utils.DoHTTP(t.Context(), s.ListenerURL(reqPath), http.MethodGet, "", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET %s status = %d, err = %v, want %d (body: %q)", reqPath, status, err, http.StatusOK, body)
	}

	u, err := parseEchoRequestURI(body)
	if err != nil {
		t.Fatalf("GET %s parseEchoRequestURI failed: %v", reqPath, err)
	}

	gotIDs := u.Query()["ids[]"]
	wantIDs := []string{"1", "2"}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "ids[]", gotIDs, wantIDs)
	}
	if got := u.Query().Get("items[0]"); got != "alpha" {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "items[0]", got, "alpha")
	}
	if got := u.Query().Get("items[1]"); got != "beta" {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "items[1]", got, "beta")
	}
	if got := u.Query().Get("tags"); got != "first,second" {
		t.Errorf("GET %s query[%q] = %q, want %q", reqPath, "tags", got, "first,second")
	}
}

// TestSQLInjectionAndXSSQuery verifies that SQL injection and XSS strings in
// query values are forwarded without proxy-side mutation or blocking.
func TestSQLInjectionAndXSSQuery(t *testing.T) {
	s := setupDataplaneQueryEnv(t)

	testCases := []struct {
		name        string
		query       string
		checkSubstr string
	}{
		{
			name:        "sql_injection_boolean",
			query:       "q=" + url.QueryEscape("' OR 1=1--"),
			checkSubstr: "1=1",
		},
		{
			name:        "sql_injection_union_select",
			query:       "filter=" + url.QueryEscape("admin' UNION SELECT * FROM users--"),
			checkSubstr: "UNION",
		},
		{
			name:        "xss_script_tag",
			query:       "search=" + url.QueryEscape("<script>alert('xss')</script>"),
			checkSubstr: "<script>",
		},
		{
			name:        "xss_img_onerror",
			query:       "img=" + url.QueryEscape("<img src=x onerror=alert(1)>"),
			checkSubstr: "onerror",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqPath := "/v1/query?" + tc.query
			_, status, body, err := utils.DoHTTP(t.Context(), s.ListenerURL(reqPath), http.MethodGet, "", nil)
			if err != nil || status != http.StatusOK {
				t.Fatalf("GET %s (%s) status = %d, err = %v, want %d (body: %q)", reqPath, tc.name, status, err, http.StatusOK, body)
			}

			u, err := parseEchoRequestURI(body)
			if err != nil {
				t.Fatalf("GET %s (%s) parseEchoRequestURI failed: %v", reqPath, tc.name, err)
			}
			found := false
		search:
			for _, vals := range u.Query() {
				for _, val := range vals {
					if strings.Contains(val, tc.checkSubstr) {
						found = true
						break search
					}
				}
			}
			if !found {
				t.Errorf("GET %s (%s) RequestURI = %q, want query value containing %q", reqPath, tc.name, u.String(), tc.checkSubstr)
			}
		})
	}
}

// TestNullByteInQueryString verifies that percent-encoded null bytes in query
// strings are not truncated.
func TestNullByteInQueryString(t *testing.T) {
	s := setupDataplaneQueryEnv(t)
	const reqPath = "/v1/query?param=value%00hidden"

	_, status, body, err := utils.DoHTTP(t.Context(), s.ListenerURL(reqPath), http.MethodGet, "", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET %s status = %d, err = %v, want %d (body: %q)", reqPath, status, err, http.StatusOK, body)
	}
	if !strings.Contains(body, "value%00hidden") && !strings.Contains(body, `value\u0000hidden`) {
		t.Errorf("GET %s body = %q, want substring %q or %q", reqPath, body, "value%00hidden", `value\u0000hidden`)
	}
}

// TestOverlongQueryURI verifies that 16KB query strings succeed while 62KB
// query URIs exceeding the buffer limit are rejected with HTTP 414 or 431.
func TestOverlongQueryURI(t *testing.T) {
	s := setupDataplaneQueryEnv(t)

	tests := []struct {
		name         string
		valueSize    int
		wantStatuses []int
	}{
		{
			name:         "within_limit_16kb",
			valueSize:    16 * 1024,
			wantStatuses: []int{http.StatusOK},
		},
		{
			name:         "exceeds_limit_62kb",
			valueSize:    62 * 1024,
			wantStatuses: []int{http.StatusRequestURITooLong, http.StatusRequestHeaderFieldsTooLarge},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL("/v1/query?data=" + strings.Repeat("x", tc.valueSize))
			_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
			if err != nil || !slices.Contains(tc.wantStatuses, status) {
				t.Errorf("GET /v1/query (%s, %d bytes) status = %d, err = %v, want one of %v (body: %q)", tc.name, tc.valueSize, status, err, tc.wantStatuses, body)
			}
		})
	}
}

func setupDataplaneQueryEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWDataplaneQuery, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	const svc = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	s.AppendAPIMethods([]*apipb.Method{
		{Name: "QueryEcho"},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".QueryEcho",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/query"},
		},
	})
	s.OverrideAuthentication(&confpb.Authentication{})
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               svc + ".QueryEcho",
				AllowUnregisteredCalls: true,
			},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(dataplane query env) failed: %v", err)
	}
	sharedEnv = s
	return sharedEnv
}

func parseEchoRequestURI(body string) (*url.URL, error) {
	var parsed struct {
		RequestURI string `json:"RequestURI"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, fmt.Errorf("unmarshaling echo response JSON %q: %w", body, err)
	}
	u, err := url.Parse(parsed.RequestURI)
	if err != nil {
		return nil, fmt.Errorf("parsing RequestURI %q: %w", parsed.RequestURI, err)
	}
	return u, nil
}
