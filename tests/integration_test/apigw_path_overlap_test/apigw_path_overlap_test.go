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

// Package apigw_path_overlap_test verifies prefix wildcard and overlapping
// route precedence under standard and strict path normalization in ESPv2.
package apigw_path_overlap_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

var (
	sharedPathOverlapEnv       *env.TestEnv
	sharedPathOverlapStrictEnv *env.TestEnv
)

// TestMain runs the package tests and tears down shared test environments.
func TestMain(m *testing.M) {
	code := m.Run()
	tearDownPathOverlapEnvs(nil)
	os.Exit(code)
}

// TestPrefixWildcardOpenSingleMethod verifies that an open single-method
// prefix wildcard route reports its operation on GET and rejects POST with 405.
func TestPrefixWildcardOpenSingleMethod(t *testing.T) {
	s := setupPathOverlapEnv(t)

	itemURL := s.ListenerURL("/v1/prefix/single/item1")
	_, gotStatus, body, err := utils.DoHTTP(t.Context(), itemURL, http.MethodGet, "", nil)
	if err != nil {
		t.Errorf("utils.DoHTTP(GET %q) failed: %v", itemURL, err)
	} else if gotStatus != http.StatusOK {
		t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", itemURL, gotStatus, body, http.StatusOK)
	} else if scReqs, err := s.ServiceControlServer.GetRequests(1); err != nil {
		t.Errorf("ServiceControlServer.GetRequests(1) after GET %q failed: %v", itemURL, err)
	} else {
		want := utils.ServiceControlExpectation{OperationName: apiName + ".PrefixSingleGet"}
		if err := utils.VerifyReportRequest(scReqs[0], want); err != nil {
			t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", itemURL, err)
		}
	}

	_, gotStatus, body, err = utils.DoHTTP(t.Context(), itemURL, http.MethodPost, `{"message":"test"}`, nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(POST %q) failed: %v", itemURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTP(POST %q) status = %d (body %q), want %d", itemURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
	scReqs, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Errorf("ServiceControlServer.GetRequests(1) after POST %q failed: %v", itemURL, err)
	} else {
		want := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodPost,
			HTTPStatus: http.StatusMethodNotAllowed,
		}
		if err := utils.VerifyReportRequest(scReqs[0], want); err != nil {
			t.Errorf("utils.VerifyReportRequest(POST %q) failed: %v", itemURL, err)
		}
	}
}

// TestPrefixWildcardOpenMultipleMethods verifies that an open multi-method
// prefix wildcard route reports distinct operations per configured method and
// rejects DELETE with 405.
func TestPrefixWildcardOpenMultipleMethods(t *testing.T) {
	s := setupPathOverlapEnv(t)

	itemURL := s.ListenerURL("/v1/prefix/multi/item1")
	tests := []struct {
		name   string
		method string
		body   string
		wantOp string
	}{
		{name: "get", method: http.MethodGet, wantOp: apiName + ".PrefixMultiGet"},
		{name: "post", method: http.MethodPost, body: `{"message":"test"}`, wantOp: apiName + ".PrefixMultiPost"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), itemURL, tc.method, tc.body, nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(%s %q) failed: %v", tc.method, itemURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTP(%s %q) status = %d (body %q), want %d", tc.method, itemURL, gotStatus, body, http.StatusOK)
			}
			scReqs, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(1) after %s %q failed: %v", tc.method, itemURL, err)
			}
			if err := utils.VerifyReportRequest(scReqs[0], utils.ServiceControlExpectation{OperationName: tc.wantOp}); err != nil {
				t.Errorf("utils.VerifyReportRequest(%s %q) failed: %v", tc.method, itemURL, err)
			}
		})
	}

	_, gotStatus, body, err := utils.DoHTTP(t.Context(), itemURL, http.MethodDelete, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(DELETE %q) failed: %v", itemURL, err)
	}
	if gotStatus != http.StatusMethodNotAllowed {
		t.Errorf("utils.DoHTTP(DELETE %q) status = %d (body %q), want %d", itemURL, gotStatus, body, http.StatusMethodNotAllowed)
	}
	scReqs, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Errorf("ServiceControlServer.GetRequests(1) after DELETE %q failed: %v", itemURL, err)
	} else {
		want := utils.ServiceControlExpectation{
			HTTPMethod: http.MethodDelete,
			HTTPStatus: http.StatusMethodNotAllowed,
		}
		if err := utils.VerifyReportRequest(scReqs[0], want); err != nil {
			t.Errorf("utils.VerifyReportRequest(DELETE %q) failed: %v", itemURL, err)
		}
	}
}

// TestOverlappingRoutesOpen verifies exact-over-wildcard precedence on open
// routes under --normalize_path=false.
func TestOverlappingRoutesOpen(t *testing.T) {
	s := setupPathOverlapStrictEnv(t)

	// Under --normalize_path=false, percent-encoded
	// /v1/overlap/open/st%61tic bypasses exact /v1/overlap/open/static
	// and routes to wildcard /v1/overlap/open/{id}.
	tests := []struct {
		name   string
		path   string
		wantOp string
	}{
		{
			name:   "exact_static",
			path:   "/v1/overlap/open/static",
			wantOp: apiName + ".OverlapOpenExact",
		},
		{
			name:   "percent_encoded_static",
			path:   "/v1/overlap/open/st%61tic",
			wantOp: apiName + ".OverlapOpenWild",
		},
		{
			name:   "dynamic_segment",
			path:   "/v1/overlap/open/123",
			wantOp: apiName + ".OverlapOpenWild",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := s.ListenerURL(tc.path)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, body, http.StatusOK)
			}
			scReqs, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(1) after GET %q failed: %v", targetURL, err)
			}
			if err := utils.VerifyReportRequest(scReqs[0], utils.ServiceControlExpectation{OperationName: tc.wantOp}); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", targetURL, err)
			}
		})
	}
}

// TestOverlappingRoutesAPIKey verifies exact-over-wildcard precedence and
// Service Control Check/Report enforcement on API-key-protected routes under
// --normalize_path=false.
func TestOverlappingRoutesAPIKey(t *testing.T) {
	s := setupPathOverlapStrictEnv(t)

	// Both exact /v1/overlap/key/protected and wildcard
	// /v1/overlap/key/{id} require an API key. Under
	// --normalize_path=false, /v1/overlap/key/prot%65cted routes to
	// wildcard OverlapKeyWild.
	tests := []struct {
		name    string
		rawPath string
		apiKey  string
		wantOp  string
	}{
		{
			name:    "exact_protected",
			rawPath: "/v1/overlap/key/protected",
			apiKey:  "api-key-1",
			wantOp:  apiName + ".OverlapKeyExact",
		},
		{
			name:    "percent_encoded_protected",
			rawPath: "/v1/overlap/key/prot%65cted",
			apiKey:  "api-key-2",
			wantOp:  apiName + ".OverlapKeyWild",
		},
		{
			name:    "dynamic_segment",
			rawPath: "/v1/overlap/key/123",
			apiKey:  "api-key-3",
			wantOp:  apiName + ".OverlapKeyWild",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			noKeyURL := s.ListenerURL(tc.rawPath)
			_, gotStatus, body, err := utils.DoHTTP(t.Context(), noKeyURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", noKeyURL, err)
			}
			if gotStatus != http.StatusUnauthorized {
				t.Errorf("utils.DoHTTP(GET %q) without key status = %d (body %q), want %d", noKeyURL, gotStatus, body, http.StatusUnauthorized)
			}
			scReqs, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(1) after unauthorized GET %q failed: %v", noKeyURL, err)
			}
			wantUnauth := utils.ServiceControlExpectation{
				OperationName: tc.wantOp,
				HTTPMethod:    http.MethodGet,
				HTTPStatus:    http.StatusUnauthorized,
			}
			if err := utils.VerifyReportRequest(scReqs[0], wantUnauth); err != nil {
				t.Errorf("utils.VerifyReportRequest(unauthorized GET %q) failed: %v", noKeyURL, err)
			}

			withKeyURL := s.ListenerURL(tc.rawPath + "?key=" + tc.apiKey)
			_, gotStatus, body, err = utils.DoHTTP(t.Context(), withKeyURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTP(GET %q) failed: %v", withKeyURL, err)
			}
			if gotStatus != http.StatusOK {
				t.Fatalf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", withKeyURL, gotStatus, body, http.StatusOK)
			}
			scReqs, err = s.ServiceControlServer.GetRequests(2)
			if err != nil {
				t.Fatalf("ServiceControlServer.GetRequests(2) after GET %q failed: %v", withKeyURL, err)
			}
			if err := utils.VerifyCheckAndReportRequests(scReqs, utils.ServiceControlExpectation{OperationName: tc.wantOp}); err != nil {
				t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", withKeyURL, err)
			}
		})
	}
}

func tearDownPathOverlapEnvs(t *testing.T) {
	if t != nil {
		t.Helper()
	}
	if sharedPathOverlapEnv != nil {
		sharedPathOverlapEnv.TearDown(t)
		sharedPathOverlapEnv = nil
	}
	if sharedPathOverlapStrictEnv != nil {
		sharedPathOverlapStrictEnv.TearDown(t)
		sharedPathOverlapStrictEnv = nil
	}
}

func buildPathOverlapEnv(t *testing.T, extraArgs ...string) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWPathOverlap, platform.EchoSidecar)
	s.EnableEchoServerRootPathHandler()

	s.AppendAPIMethods([]*apipb.Method{
		{Name: "PrefixSingleGet"},
		{Name: "PrefixMultiGet"},
		{Name: "PrefixMultiPost"},
		{Name: "OverlapOpenExact"},
		{Name: "OverlapOpenWild"},
		{Name: "OverlapKeyExact"},
		{Name: "OverlapKeyWild"},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{Selector: apiName + ".PrefixSingleGet", AllowUnregisteredCalls: true},
		{Selector: apiName + ".PrefixMultiGet", AllowUnregisteredCalls: true},
		{Selector: apiName + ".PrefixMultiPost", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapOpenExact", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapOpenWild", AllowUnregisteredCalls: true},
		{Selector: apiName + ".OverlapKeyExact", AllowUnregisteredCalls: false},
		{Selector: apiName + ".OverlapKeyWild", AllowUnregisteredCalls: false},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".PrefixSingleGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/prefix/single/*"},
		},
		{
			Selector: apiName + ".PrefixMultiGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/prefix/multi/*"},
		},
		{
			Selector: apiName + ".PrefixMultiPost",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/prefix/multi/*"},
		},
		{
			Selector: apiName + ".OverlapOpenExact",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/open/static"},
		},
		{
			Selector: apiName + ".OverlapOpenWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/open/{id}"},
		},
		{
			Selector: apiName + ".OverlapKeyExact",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/key/protected"},
		},
		{
			Selector: apiName + ".OverlapKeyWild",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/overlap/key/{id}"},
		},
	})

	args := append(utils.CommonArgs(), extraArgs...)
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	return s
}

func setupPathOverlapEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedPathOverlapEnv != nil {
		sharedPathOverlapEnv.ServiceControlServer.DrainRequests()
		return sharedPathOverlapEnv
	}
	tearDownPathOverlapEnvs(t)
	sharedPathOverlapEnv = buildPathOverlapEnv(t)
	return sharedPathOverlapEnv
}

func setupPathOverlapStrictEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedPathOverlapStrictEnv != nil {
		sharedPathOverlapStrictEnv.ServiceControlServer.DrainRequests()
		return sharedPathOverlapStrictEnv
	}
	tearDownPathOverlapEnvs(t)
	sharedPathOverlapStrictEnv = buildPathOverlapEnv(t, "--normalize_path=false")
	return sharedPathOverlapStrictEnv
}
