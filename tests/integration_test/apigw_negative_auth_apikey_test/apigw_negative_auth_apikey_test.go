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

// Package apigw_negative_auth_apikey_test verifies negative API key
// authentication behavior for missing and malformed keys in ESPv2.
package apigw_negative_auth_apikey_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	scpb "cloud.google.com/go/servicecontrol/apiv1/servicecontrolpb"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const apiName = "1.echo_api_endpoints_cloudesf_testing_cloud_goog"

// TestMissingAPIKey verifies that a request without an API key to a protected
// route is rejected with 401 Unauthorized.
func TestMissingAPIKey(t *testing.T) {
	s := setupNegativeAuthAPIKeyEnv(t)

	targetURL := s.ListenerURL("/v1/auth/apikey")
	_, gotStatus, gotBody, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
	}

	// ESPv2 rejects requests missing a required API key with
	// 401 Unauthorized.
	if gotStatus != http.StatusUnauthorized {
		t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, gotBody, http.StatusUnauthorized)
	}
	if wantSubstr := "Method doesn't allow unregistered callers"; !strings.Contains(gotBody, wantSubstr) {
		t.Errorf("utils.DoHTTP(GET %q) body = %q, want substring %q", targetURL, gotBody, wantSubstr)
	}
}

// TestMalformedAPIKey verifies that a request with a malformed API key
// rejected by Service Control returns 400 Bad Request.
func TestMalformedAPIKey(t *testing.T) {
	s := setupNegativeAuthAPIKeyEnv(t)

	s.ServiceControlServer.SetCheckResponse(&scpb.CheckResponse{
		CheckErrors: []*scpb.CheckError{
			{
				Code:   scpb.CheckError_API_KEY_INVALID,
				Detail: "API key not valid. Please pass a valid API key.",
			},
		},
	})

	targetURL := s.ListenerURL("/v1/auth/apikey?key=corrupted!!!@@@key")
	_, gotStatus, gotBody, err := utils.DoHTTP(t.Context(), targetURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("utils.DoHTTP(GET %q) failed: %v", targetURL, err)
	}

	// ESPv2 maps CheckError_API_KEY_INVALID to INVALID_ARGUMENT
	// (HTTP 400 Bad Request).
	if gotStatus != http.StatusBadRequest {
		t.Errorf("utils.DoHTTP(GET %q) status = %d (body %q), want %d", targetURL, gotStatus, gotBody, http.StatusBadRequest)
	}
	if wantSubstr := "API key not valid"; !strings.Contains(gotBody, wantSubstr) {
		t.Errorf("utils.DoHTTP(GET %q) body = %q, want substring %q", targetURL, gotBody, wantSubstr)
	}
}

func setupNegativeAuthAPIKeyEnv(t *testing.T) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWNegativeAuthAPIKey, platform.EchoSidecar)
	t.Cleanup(func() { s.TearDown(t) })
	s.EnableEchoServerRootPathHandler()

	s.AppendAPIMethods([]*apipb.Method{
		{Name: "ProtectedApiKeyGet"},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{
			Selector:               apiName + ".ProtectedApiKeyGet",
			AllowUnregisteredCalls: false,
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: apiName + ".ProtectedApiKeyGet",
			Pattern:  &annotationspb.HttpRule_Get{Get: "/v1/auth/apikey"},
		},
	})

	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}
	return s
}
