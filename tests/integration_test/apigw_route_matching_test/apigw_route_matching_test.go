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

// Package apigw_route_matching_test verifies HTTP route matching, method
// dispatch, path normalization, and Service Control operation reporting for
// ESPv2 API Gateway configurations.
package apigw_route_matching_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const (
	apiName            = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"
	reportWaitAttempts = 4
)

var (
	sharedRouteMatchingEnv       *env.TestEnv
	sharedRouteMatchingStrictEnv *env.TestEnv
)

// TestMain runs the package tests and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownRouteMatchingEnvs(nil)
	os.Exit(code)
}

// TestExactRouteSingleMethod verifies that a single-method exact route reports
// its configured operation and rejects unconfigured HTTP methods with 405.
func TestExactRouteSingleMethod(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	exactURL := s.ListenerURL("/v1/exact")
	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), exactURL, http.MethodGet, "", nil)
	if err != nil {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) failed: %v", exactURL, err)
	} else if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", exactURL, gotStatus, body, http.StatusOK)
	} else {
		wantOp := apiName + ".ExactGet"
		if _, err := s.ServiceControlServer.WaitForReportedOperation(wantOp, reportWaitAttempts); err != nil {
			t.Errorf("WaitForReportedOperation(%q) after GET %q failed: %v", wantOp, exactURL, err)
		}
	}

	// An unconfigured HTTP method on a matched path template returns 405.
	_, gotStatus, body, err = utils.DoHTTPNoRedirect(t.Context(), exactURL, http.MethodPost, "{}", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(POST %q) failed: %v", exactURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTPNoRedirect(POST %q) status = %d (body %q), want %d", exactURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
}

// TestExactRouteMultipleMethods verifies that multiple explicit HTTP methods
// on an exact route dispatch to distinct operations while unconfigured methods
// return 405.
func TestExactRouteMultipleMethods(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	multiURL := s.ListenerURL("/v1/multi")
	tests := []struct {
		name   string
		method string
		wantOp string
	}{
		{name: "get", method: http.MethodGet, wantOp: apiName + ".MultiGet"},
		{name: "post", method: http.MethodPost, wantOp: apiName + ".MultiPost"},
		{name: "delete", method: http.MethodDelete, wantOp: apiName + ".MultiDelete"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), multiURL, tc.method, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTPNoRedirect(%s %q) failed: %v", tc.method, multiURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTPNoRedirect(%s %q) status = %d (body %q), want %d", tc.method, multiURL, gotStatus, body, http.StatusOK)
			}
			if _, err := s.ServiceControlServer.WaitForReportedOperation(tc.wantOp, reportWaitAttempts); err != nil {
				t.Errorf("WaitForReportedOperation(%q) after %s %q failed: %v", tc.wantOp, tc.method, multiURL, err)
			}
		})
	}

	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), multiURL, http.MethodPut, "{}", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(PUT %q) failed: %v", multiURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTPNoRedirect(PUT %q) status = %d (body %q), want %d", multiURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
}

// TestPrefixWildcardSingleMethod verifies that a single-segment wildcard route
// matches one path segment and returns 404 for deeper paths.
func TestPrefixWildcardSingleMethod(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	shelfURL := s.ListenerURL("/v1/shelves/123")
	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), shelfURL, http.MethodGet, "", nil)
	if err != nil {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) failed: %v", shelfURL, err)
	} else if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", shelfURL, gotStatus, body, http.StatusOK)
	} else {
		wantOp := apiName + ".PrefixGetShelf"
		if _, err := s.ServiceControlServer.WaitForReportedOperation(wantOp, reportWaitAttempts); err != nil {
			t.Errorf("WaitForReportedOperation(%q) after GET %q failed: %v", wantOp, shelfURL, err)
		}
	}

	// A multi-segment path does not match single-segment wildcard * and
	// returns 404 Not Found.
	booksURL := s.ListenerURL("/v1/shelves/123/books")
	_, gotStatus, body, err = utils.DoHTTPNoRedirect(t.Context(), booksURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(GET %q) failed: %v", booksURL, err)
	}
	if gotStatus != http.StatusNotFound {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", booksURL, gotStatus, body, http.StatusNotFound)
	}
}

// TestPrefixWildcardMultipleMethods verifies that multiple explicit methods on
// a prefix wildcard route report their respective operations and reject other
// methods with 405.
func TestPrefixWildcardMultipleMethods(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	itemURL := s.ListenerURL("/v1/items/abc")
	tests := []struct {
		name   string
		method string
		body   string
		wantOp string
	}{
		{name: "get", method: http.MethodGet, wantOp: apiName + ".GetItem"},
		{name: "put", method: http.MethodPut, body: "{}", wantOp: apiName + ".PutItem"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), itemURL, tc.method, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoHTTPNoRedirect(%s %q) failed: %v", tc.method, itemURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTPNoRedirect(%s %q) status = %d (body %q), want %d", tc.method, itemURL, gotStatus, body, http.StatusOK)
			}
			if _, err := s.ServiceControlServer.WaitForReportedOperation(tc.wantOp, reportWaitAttempts); err != nil {
				t.Errorf("WaitForReportedOperation(%q) after %s %q failed: %v", tc.wantOp, tc.method, itemURL, err)
			}
		})
	}

	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), itemURL, http.MethodPost, "{}", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(POST %q) failed: %v", itemURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTPNoRedirect(POST %q) status = %d (body %q), want %d", itemURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
}

// TestMultiSegmentWildcardRoute verifies that a multi-segment wildcard (**)
// route matches nested paths and reports its operation.
func TestMultiSegmentWildcardRoute(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	deepURL := s.ListenerURL("/v1/deep/a/b/c")
	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), deepURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(GET %q) failed: %v", deepURL, err)
	}
	if gotStatus != http.StatusOK {
		t.Fatalf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", deepURL, gotStatus, body, http.StatusOK)
	}
	wantOp := apiName + ".DeepWildcard"
	if _, err := s.ServiceControlServer.WaitForReportedOperation(wantOp, reportWaitAttempts); err != nil {
		t.Errorf("WaitForReportedOperation(%q) after GET %q failed: %v", wantOp, deepURL, err)
	}
}

// TestCommonNonStandardHTTPMethods verifies that a configured custom OPTIONS
// route succeeds and reports its operation while unconfigured TRACE returns
// 405.
func TestCommonNonStandardHTTPMethods(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	nonStdURL := s.ListenerURL("/v1/nonstd")
	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), nonStdURL, http.MethodOptions, "", nil)
	if err != nil {
		t.Errorf("utils.DoHTTPNoRedirect(OPTIONS %q) failed: %v", nonStdURL, err)
	} else if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTPNoRedirect(OPTIONS %q) status = %d (body %q), want %d", nonStdURL, gotStatus, body, http.StatusOK)
	} else {
		wantOp := apiName + ".NonStdOptions"
		if _, err := s.ServiceControlServer.WaitForReportedOperation(wantOp, reportWaitAttempts); err != nil {
			t.Errorf("WaitForReportedOperation(%q) after OPTIONS %q failed: %v", wantOp, nonStdURL, err)
		}
	}

	_, gotStatus, body, err = utils.DoHTTPNoRedirect(t.Context(), nonStdURL, http.MethodTrace, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(TRACE %q) failed: %v", nonStdURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTPNoRedirect(TRACE %q) status = %d (body %q), want %d", nonStdURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
}

// TestArbitraryHTTPMethodRejected verifies that an arbitrary unrecognized HTTP
// verb is rejected with 400 Bad Request.
func TestArbitraryHTTPMethodRejected(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	addr := s.ListenerAddr()
	rawReq := "FOOBAR /v1/exact HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	gotStatusLine, err := utils.SendRawTCPStatusLine(t.Context(), addr, rawReq, 5*time.Second)
	if err != nil {
		t.Fatalf("utils.SendRawTCPStatusLine(%q, FOOBAR) failed: %v", addr, err)
	}
	if want := "400 Bad Request"; !strings.Contains(gotStatusLine, want) {
		t.Errorf("utils.SendRawTCPStatusLine(%q, FOOBAR) = %q, want substring %q", addr, gotStatusLine, want)
	}
}

// TestSingleSegmentWildcardTraversalNormalized verifies that Envoy normalizes
// percent-encoded dot-dot traversal segments before matching single-segment
// wildcards and rejects out-of-bounds traversal with 404.
func TestSingleSegmentWildcardTraversalNormalized(t *testing.T) {
	s := setupRouteMatchingEnv(t)

	// Envoy normalizes /v1/shelves/subdir/%2E%2E/100 -> /v1/shelves/100
	// (matching single-segment wildcard /v1/shelves/*).
	inBoundsURL := s.ListenerURL("/v1/shelves/subdir/%2E%2E/100")
	_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), inBoundsURL, http.MethodGet, "", nil)
	if err != nil {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) failed: %v", inBoundsURL, err)
	} else if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", inBoundsURL, gotStatus, body, http.StatusOK)
	} else {
		wantOp := apiName + ".PrefixGetShelf"
		if _, err := s.ServiceControlServer.WaitForReportedOperation(wantOp, reportWaitAttempts); err != nil {
			t.Errorf("WaitForReportedOperation(%q) after GET %q failed: %v", wantOp, inBoundsURL, err)
		}
	}

	// Out-of-bounds traversal /v1/shelves/%2E%2E/%2E%2E/etc/passwd
	// normalizes to /etc/passwd and returns 404 Not Found.
	outOfBoundsURL := s.ListenerURL("/v1/shelves/%2E%2E/%2E%2E/etc/passwd")
	_, gotStatus, body, err = utils.DoHTTPNoRedirect(t.Context(), outOfBoundsURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTPNoRedirect(GET %q) failed: %v", outOfBoundsURL, err)
	}
	if gotStatus != http.StatusNotFound {
		t.Errorf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", outOfBoundsURL, gotStatus, body, http.StatusNotFound)
	}
}

// TestOverlappingRoutesPrecedence verifies exact-over-wildcard route
// precedence and raw percent-encoded path matching under
// --normalize_path=false.
func TestOverlappingRoutesPrecedence(t *testing.T) {
	s := setupRouteMatchingStrictEnv(t)

	// Under --normalize_path=false, percent-encoded /v1/overlap/st%61tic
	// does not match exact /v1/overlap/static and routes instead to
	// wildcard /v1/overlap/{id}.
	tests := []struct {
		name   string
		path   string
		wantOp string
	}{
		{
			name:   "exact_static",
			path:   "/v1/overlap/static",
			wantOp: apiName + ".OverlapStatic",
		},
		{
			name:   "percent_encoded_static",
			path:   "/v1/overlap/st%61tic",
			wantOp: apiName + ".OverlapWildcard",
		},
		{
			name:   "dynamic_segment",
			path:   "/v1/overlap/dynamic123",
			wantOp: apiName + ".OverlapWildcard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTPNoRedirect(t.Context(), targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTPNoRedirect(GET %q) failed: %v", targetURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTPNoRedirect(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, body, http.StatusOK)
			}
			if _, err := s.ServiceControlServer.WaitForReportedOperation(tc.wantOp, reportWaitAttempts); err != nil {
				t.Errorf("WaitForReportedOperation(%q) after GET %q failed: %v", tc.wantOp, targetURL, err)
			}
		})
	}
}

func tearDownRouteMatchingEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedRouteMatchingEnv != nil {
		sharedRouteMatchingEnv.TearDown(t)
		sharedRouteMatchingEnv = nil
	}
	if sharedRouteMatchingStrictEnv != nil {
		sharedRouteMatchingStrictEnv.TearDown(t)
		sharedRouteMatchingStrictEnv = nil
	}
}

func buildRouteMatchingEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWRouteMatching, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	methodNames := []string{
		"ExactGet",
		"MultiGet",
		"MultiPost",
		"MultiDelete",
		"PrefixGetShelf",
		"GetItem",
		"PutItem",
		"OverlapStatic",
		"OverlapWildcard",
		"DeepWildcard",
		"NonStdOptions",
	}

	var methods []*apipb.Method
	var usageRules []*confpb.UsageRule
	for _, m := range methodNames {
		methods = append(methods, &apipb.Method{Name: m})
		usageRules = append(usageRules, &confpb.UsageRule{
			Selector:               apiName + "." + m,
			AllowUnregisteredCalls: true,
		})
	}
	s.AppendAPIMethods(methods)
	s.AppendUsageRules(usageRules)

	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".ExactGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/exact"},
		},
		{
			Selector: apiName + ".MultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/multi"},
		},
		{
			Selector: apiName + ".MultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/multi"},
		},
		{
			Selector: apiName + ".MultiDelete",
			Pattern:  &annotationspb.HttpRule_Delete{Delete: "/v1/multi"},
		},
		{
			Selector: apiName + ".PrefixGetShelf",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shelves/*"},
		},
		{
			Selector: apiName + ".GetItem",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/items/*"},
		},
		{
			Selector: apiName + ".PutItem",
			Pattern:  &annotationspb.HttpRule_Put{Put: "/v1/items/*"},
		},
		{
			Selector: apiName + ".OverlapStatic",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/static"},
		},
		{
			Selector: apiName + ".OverlapWildcard",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/{id}"},
		},
		{
			Selector: apiName + ".DeepWildcard",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/deep/**"},
		},
		{
			Selector: apiName + ".NonStdOptions",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: "OPTIONS",
					Path: "/v1/nonstd",
				},
			},
		},
	})

	args := append(utils.CommonArgs(), extraArgs...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	return s
}

func setupRouteMatchingEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedRouteMatchingEnv != nil {
		sharedRouteMatchingEnv.ServiceControlServer.DrainRequests()
		return sharedRouteMatchingEnv
	}
	tearDownRouteMatchingEnvs(t)
	sharedRouteMatchingEnv = buildRouteMatchingEnv(t)
	return sharedRouteMatchingEnv
}

func setupRouteMatchingStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedRouteMatchingStrictEnv != nil {
		sharedRouteMatchingStrictEnv.ServiceControlServer.DrainRequests()
		return sharedRouteMatchingStrictEnv
	}
	tearDownRouteMatchingEnvs(t)
	sharedRouteMatchingStrictEnv = buildRouteMatchingEnv(t, "--normalize_path=false")
	return sharedRouteMatchingStrictEnv
}
