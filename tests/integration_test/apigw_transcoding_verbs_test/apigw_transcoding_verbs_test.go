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

// Package apigw_transcoding_verbs_test verifies API Gateway custom HTTP verb
// routing and transcoding, including case sensitivity and escaped slashes.
package apigw_transcoding_verbs_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
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

// TestCustomVerbCreate verifies gRPC transcoding for the :create custom verb.
func TestCustomVerbCreate(t *testing.T) {
	s := setupTranscodingVerbsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves/100:create")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"id":500,"author":"Leo Tolstoy","title":"War and Peace"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("POST /v1/shelves/100:create status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"id":"500","author":"Leo Tolstoy","title":"War and Peace"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("POST /v1/shelves/100:create JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestCustomVerbBatchGet verifies gRPC transcoding for the :batchGet custom
// verb with query parameters.
func TestCustomVerbBatchGet(t *testing.T) {
	s := setupTranscodingVerbsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves/100:batchGet?book=1001")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GET /v1/shelves/100:batchGet?book=1001 status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	}
	want := `{"id":"1001","title":"Alphabet"}`
	if err := util.JsonEqual(want, body); err != nil {
		t.Errorf("GET /v1/shelves/100:batchGet?book=1001 JSON mismatch (got %q, want %q):\n%v", body, want, err)
	}
}

// TestCustomVerbUpperCase verifies case-sensitive matching for uppercase
// custom verbs.
func TestCustomVerbUpperCase(t *testing.T) {
	s := setupTranscodingVerbsEnv(t)
	upperURL := s.ListenerURL("/v1/shelves/100:UPPER")
	_, status, body, err := utils.DoHTTP(t.Context(), upperURL, http.MethodPost, `{"id":600,"author":"Homer","title":"Odyssey"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusOK {
		t.Errorf("POST /v1/shelves/100:UPPER status = %d, err = %v, want %d (body: %q)", status, err, http.StatusOK, body)
	} else {
		want := `{"id":"600","author":"Homer","title":"Odyssey"}`
		if err := util.JsonEqual(want, body); err != nil {
			t.Errorf("POST /v1/shelves/100:UPPER JSON mismatch (got %q, want %q):\n%v", body, want, err)
		}
	}

	// Lowercase :upper does not match uppercase :UPPER.
	lowerURL := s.ListenerURL("/v1/shelves/100:upper")
	_, statusLower, bodyLower, err := utils.DoHTTP(t.Context(), lowerURL, http.MethodPost, `{"id":601,"title":"Iliad"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || statusLower != http.StatusNotFound {
		t.Errorf("POST /v1/shelves/100:upper status = %d, err = %v, want %d (body: %q)", statusLower, err, http.StatusNotFound, bodyLower)
	}
}

// TestCustomVerbEscapedSlash verifies rejection when an escaped slash appears
// in a custom verb segment.
func TestCustomVerbEscapedSlash(t *testing.T) {
	s := setupTranscodingVerbsEnv(t)
	targetURL := s.ListenerURL("/v1/shelves/100:%2Fcreate")
	_, status, body, err := utils.DoHTTP(t.Context(), targetURL, http.MethodPost, `{"id":700,"title":"Escaped"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil || status != http.StatusNotFound {
		t.Errorf("POST /v1/shelves/100:%%2Fcreate status = %d, err = %v, want %d (body: %q)", status, err, http.StatusNotFound, body)
	}
}

func setupTranscodingVerbsEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	if sharedEnv != nil {
		return sharedEnv
	}

	s := env.NewTestEnv(platform.TestAPIGWTranscodingVerbs, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{})

	const svc = "endpoints.examples.bookstore.Bookstore"
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               svc + ".CreateBookWithCustomVerb",
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               svc + ".GetBook",
				AllowUnregisteredCalls: true,
			},
		},
	})

	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: svc + ".CreateBookWithCustomVerb",
			Pattern:  &annotationspb.HttpRule_Post{Post: "/v1/shelves/{shelf}:create"},
			Body:     "book",
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Post{Post: "/v1/shelves/{shelf}:UPPER"},
					Body:    "book",
				},
			},
		},
		{
			Selector: svc + ".GetBook",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/shelves/{shelf=*}/books/{book}"},
			AdditionalBindings: []*annotationspb.HttpRule{
				{
					Pattern: &annotationspb.HttpRule_Get{Get: "/v1/shelves/{shelf}:batchGet"},
				},
			},
		},
	})

	args := append(utils.CommonArgs(), "--disallow_colon_in_wildcard_path_segment")
	if err := s.Setup(args); err != nil {
		s.TearDown(t)
		t.Fatalf("s.Setup(%v) failed: %v", args, err)
	}
	sharedEnv = s
	return sharedEnv
}
