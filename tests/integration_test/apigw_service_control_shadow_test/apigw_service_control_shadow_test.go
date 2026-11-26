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

// Package apigw_service_control_shadow_test tests ESPv2 API Gateway shadowed
// wildcard route precedence, authentication enforcement, and Service Control
// reporting.
package apigw_service_control_shadow_test

import (
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const echoV2Service = "2.echo_api_endpoints_cloudesf_testing_cloud_goog"

// TestExactShadowedWildcardUnauthenticatedExactRouteSucceeds verifies that an
// unauthenticated request to a public exact route succeeds and reports the
// exact operation when a broader wildcard route requires authentication.
func TestExactShadowedWildcardUnauthenticatedExactRouteSucceeds(t *testing.T) {
	s := setupShadowEnv(t, exactShadowedWildcardConfig)

	reqURL := s.ListenerURL("/echoMethod/public")
	resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
	}
	const wantResp = `{"RequestMethod": "GET"}`
	if err := util.JsonEqual(wantResp, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: echoV2Service + ".EchoMethodPublicExact",
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyReportRequest(scRequests[0], wantSC); err != nil {
		t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", reqURL, err)
	}
}

// TestShadowedWildcardSuccessfulRequests verifies route precedence and Service
// Control Check and Report telemetry for public and authenticated requests
// across exact, single-segment, and multi-segment shadowed wildcard routes.
func TestShadowedWildcardSuccessfulRequests(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        shadowRouteConfig
		pathSuffix string
		headers    map[string]string
		wantCheck  bool
		wantSC     utils.ServiceControlExpectation
	}{
		{
			name:       "exact_shadowed_authenticated_wildcard",
			cfg:        exactShadowedWildcardConfig,
			pathSuffix: "/echoMethod/other-secure-item?key=valid-api-key-68",
			wantCheck:  true,
			wantSC: utils.ServiceControlExpectation{
				OperationName: echoV2Service + ".EchoMethodWildcardAuth",
				ConsumerID:    "api_key:valid-api-key-68",
				HTTPStatus:    http.StatusOK,
			},
		},
		{
			name:       "single_wildcard_unauthenticated_precedence",
			cfg:        singleWildcardShadowedConfig,
			pathSuffix: "/echoMethod/public/open-item-123",
			wantSC: utils.ServiceControlExpectation{
				OperationName: echoV2Service + ".EchoMethodPublicWildcard",
				HTTPStatus:    http.StatusOK,
			},
		},
		{
			name:       "single_wildcard_authenticated_catch_all",
			cfg:        singleWildcardShadowedConfig,
			pathSuffix: "/echoMethod/secure/sub/nested?key=valid-api-key-69",
			wantCheck:  true,
			wantSC: utils.ServiceControlExpectation{
				OperationName: echoV2Service + ".EchoMethodCatchAllAuth",
				ConsumerID:    "api_key:valid-api-key-69",
				HTTPStatus:    http.StatusOK,
			},
		},
		{
			name:       "multi_segment_unauthenticated_deep_open_path",
			cfg:        multiSegmentShadowedConfig,
			pathSuffix: "/echoMethod/open/deep/sub/path/resource",
			wantSC: utils.ServiceControlExpectation{
				OperationName: echoV2Service + ".EchoMethodOpenMultiWildcard",
				HTTPStatus:    http.StatusOK,
			},
		},
		{
			name:       "multi_segment_authenticated_private_path",
			cfg:        multiSegmentShadowedConfig,
			pathSuffix: "/echoMethod/private/deep/sub/path/resource?key=valid-api-key-70",
			headers: map[string]string{
				"Authorization": "Bearer " + testdata.FakeCloudTokenMultiAudiences,
			},
			wantCheck: true,
			wantSC: utils.ServiceControlExpectation{
				OperationName: echoV2Service + ".EchoMethodParentWildcardAuth",
				ConsumerID:    "api_key:valid-api-key-70",
				HTTPStatus:    http.StatusOK,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := setupShadowEnv(t, tc.cfg)

			reqURL := s.ListenerURL(tc.pathSuffix)
			resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", tc.headers)
			if err != nil {
				t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
			}
			const wantResp = `{"RequestMethod": "GET"}`
			if err := util.JsonEqual(wantResp, string(resp)); err != nil {
				t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
			}

			if tc.wantCheck {
				scRequests, err := s.ServiceControlServer.GetRequests(2)
				if err != nil {
					t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
				}
				if err := utils.VerifyCheckAndReportRequests(scRequests, tc.wantSC); err != nil {
					t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", reqURL, err)
				}
				return
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("s.ServiceControlServer.GetRequests(1) failed: %v", err)
			}
			if err := utils.VerifyReportRequest(scRequests[0], tc.wantSC); err != nil {
				t.Errorf("utils.VerifyReportRequest(GET %q) failed: %v", reqURL, err)
			}
		})
	}
}

// TestShadowedWildcardUnauthenticatedRequestsRejected verifies that
// unauthenticated requests matching protected single-segment and multi-segment
// wildcard routes are rejected with HTTP 401 Unauthorized.
func TestShadowedWildcardUnauthenticatedRequestsRejected(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        shadowRouteConfig
		pathSuffix string
	}{
		{
			name:       "exact_shadowed_unauthenticated_wildcard",
			cfg:        exactShadowedWildcardConfig,
			pathSuffix: "/echoMethod/other-secure-item",
		},
		{
			name:       "single_wildcard_unauthenticated_catch_all",
			cfg:        singleWildcardShadowedConfig,
			pathSuffix: "/echoMethod/secure/sub/nested",
		},
		{
			name:       "multi_segment_unauthenticated_private_path",
			cfg:        multiSegmentShadowedConfig,
			pathSuffix: "/echoMethod/private/deep/sub/path/resource",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := setupShadowEnv(t, tc.cfg)

			reqURL := s.ListenerURL(tc.pathSuffix)
			_, gotStatus, gotBody, err := utils.DoHTTPNoRedirect(t.Context(), reqURL, http.MethodGet, "", nil)
			if err != nil {
				t.Fatalf("utils.DoHTTPNoRedirect(GET, %q) failed: %v", reqURL, err)
			}
			if gotStatus != http.StatusUnauthorized {
				t.Errorf("utils.DoHTTPNoRedirect(GET, %q) status = %d, want %d (body: %q)", reqURL, gotStatus, http.StatusUnauthorized, gotBody)
			}
		})
	}
}

type shadowRouteConfig struct {
	publicMethod      string
	publicPattern     string
	protectedMethod   string
	protectedPattern  string
	protectedAudience string
}

var (
	exactShadowedWildcardConfig = shadowRouteConfig{
		publicMethod:     "EchoMethodPublicExact",
		publicPattern:    "/echoMethod/public",
		protectedMethod:  "EchoMethodWildcardAuth",
		protectedPattern: "/echoMethod/*",
	}
	singleWildcardShadowedConfig = shadowRouteConfig{
		publicMethod:     "EchoMethodPublicWildcard",
		publicPattern:    "/echoMethod/public/*",
		protectedMethod:  "EchoMethodCatchAllAuth",
		protectedPattern: "/echoMethod/**",
	}
	multiSegmentShadowedConfig = shadowRouteConfig{
		publicMethod:      "EchoMethodOpenMultiWildcard",
		publicPattern:     "/echoMethod/open/**",
		protectedMethod:   "EchoMethodParentWildcardAuth",
		protectedPattern:  "/echoMethod/**",
		protectedAudience: "admin.cloud.goog",
	}
)

func setupShadowEnv(t *testing.T, cfg shadowRouteConfig) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWServiceControlShadow, platform.EchoSidecar)
	t.Cleanup(func() { s.TearDown(t) })

	publicSelector := echoV2Service + "." + cfg.publicMethod
	protectedSelector := echoV2Service + "." + cfg.protectedMethod

	s.AppendAPIs([]*apipb.Api{
		{
			Name: echoV2Service,
			Methods: []*apipb.Method{
				{Name: cfg.publicMethod},
				{Name: cfg.protectedMethod},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: publicSelector,
			Pattern:  &annotationspb.HttpRule_Get{Get: cfg.publicPattern},
		},
		{
			Selector: protectedSelector,
			Pattern:  &annotationspb.HttpRule_Get{Get: cfg.protectedPattern},
		},
	})
	if cfg.protectedAudience != "" {
		s.OverrideAuthentication(&confpb.Authentication{
			Rules: []*confpb.AuthenticationRule{
				{
					Selector: protectedSelector,
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: testdata.GoogleJwtProvider,
							Audiences:  cfg.protectedAudience,
						},
					},
				},
			},
		})
	}
	s.OverrideUsage(&confpb.Usage{
		Rules: []*confpb.UsageRule{
			{
				Selector:               publicSelector,
				AllowUnregisteredCalls: true,
			},
			{
				Selector:               protectedSelector,
				AllowUnregisteredCalls: false,
			},
		},
	})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}
	return s
}
