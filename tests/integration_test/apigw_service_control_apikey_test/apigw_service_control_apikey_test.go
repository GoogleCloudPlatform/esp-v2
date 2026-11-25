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

// Package apigw_service_control_apikey_test tests ESPv2 API Gateway combined
// header and query API key extraction with Service Control Check and Report
// enforcement.
package apigw_service_control_apikey_test

import (
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	apipb "google.golang.org/genproto/protobuf/api"
)

const echoV2Service = "2.echo_api_endpoints_cloudesf_testing_cloud_goog"

// TestExactCombinedAPIKey verifies combined header and query parameter API key
// extraction and Service Control Check and Report enforcement on an exact GET
// route (/echoMethod/exactkey).
func TestExactCombinedAPIKey(t *testing.T) {
	const methodName = "EchoMethodExactKey"
	const wantOp = echoV2Service + "." + methodName
	s := setupAPIKeyEnv(t, methodName, &annotationspb.HttpRule{
		Selector: wantOp,
		Pattern: &annotationspb.HttpRule_Get{
			Get: "/echoMethod/exactkey",
		},
	})

	reqURL := s.ListenerURL("/echoMethod/exactkey?key=query-api-key-65")
	headers := map[string]string{
		"x-api-key":      "header-api-key-65",
		"x-goog-api-key": "header-api-key-65",
	}
	resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", headers)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
	}
	const wantResp = `{"RequestMethod": "GET"}`
	if err := util.JsonEqual(wantResp, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(2)
	if err != nil {
		t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: wantOp,
		ConsumerID:    "api_key:query-api-key-65",
		HTTPMethod:    http.MethodGet,
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
		t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", reqURL, err)
	}
}

// TestSingleWildcardCombinedAPIKey verifies combined header and query API key
// extraction and Service Control Check and Report enforcement on a
// single-segment wildcard GET route (/echoMethod/*).
func TestSingleWildcardCombinedAPIKey(t *testing.T) {
	const methodName = "EchoMethodSingleWildcardKey"
	const wantOp = echoV2Service + "." + methodName
	s := setupAPIKeyEnv(t, methodName, &annotationspb.HttpRule{
		Selector: wantOp,
		Pattern: &annotationspb.HttpRule_Get{
			Get: "/echoMethod/*",
		},
	})

	reqURL := s.ListenerURL("/echoMethod/wildcard-item-66?key=query-api-key-66")
	headers := map[string]string{
		"x-api-key":      "header-api-key-66",
		"x-goog-api-key": "header-api-key-66",
	}
	resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", headers)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
	}
	const wantResp = `{"RequestMethod": "GET"}`
	if err := util.JsonEqual(wantResp, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(2)
	if err != nil {
		t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: wantOp,
		ConsumerID:    "api_key:query-api-key-66",
		HTTPMethod:    http.MethodGet,
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
		t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", reqURL, err)
	}
}

// TestMultiSegmentWildcardCombinedAPIKey verifies combined header and query
// API key extraction and Service Control Check and Report enforcement on a
// multi-segment wildcard GET route (/dynamicrouting/**).
func TestMultiSegmentWildcardCombinedAPIKey(t *testing.T) {
	const methodName = "DynamicRoutingMultiWildcardKey"
	const wantOp = echoV2Service + "." + methodName
	s := setupAPIKeyEnv(t, methodName, &annotationspb.HttpRule{
		Selector: wantOp,
		Pattern: &annotationspb.HttpRule_Get{
			Get: "/dynamicrouting/**",
		},
	})

	reqURL := s.ListenerURL("/dynamicrouting/deep/nested/sub/resource?key=query-api-key-67")
	headers := map[string]string{
		"x-api-key":      "header-api-key-67",
		"x-goog-api-key": "header-api-key-67",
	}
	resp, err := client.DoWithHeaders(reqURL, http.MethodGet, "", headers)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, GET) failed: %v", reqURL, err)
	}
	const wantResp = `{"RequestURI": "/dynamicrouting/deep/nested/sub/resource?key=query-api-key-67"}`
	if err := util.JsonEqual(wantResp, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, GET) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(2)
	if err != nil {
		t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: wantOp,
		ConsumerID:    "api_key:query-api-key-67",
		HTTPMethod:    http.MethodGet,
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
		t.Errorf("utils.VerifyCheckAndReportRequests(GET %q) failed: %v", reqURL, err)
	}
}

// TestMultiSegmentWildcardNonStandardCombinedAPIKey verifies combined header
// and query API key extraction and Service Control enforcement on a
// multi-segment wildcard OPTIONS route (/bearertoken/**).
func TestMultiSegmentWildcardNonStandardCombinedAPIKey(t *testing.T) {
	const methodName = "BearerTokenMultiWildcardOptionsKey"
	const wantOp = echoV2Service + "." + methodName
	s := setupAPIKeyEnv(t, methodName, &annotationspb.HttpRule{
		Selector: wantOp,
		Pattern: &annotationspb.HttpRule_Custom{
			Custom: &annotationspb.CustomHttpPattern{
				Kind: http.MethodOptions,
				Path: "/bearertoken/**",
			},
		},
	})

	reqURL := s.ListenerURL("/bearertoken/deep/nested/item-71?key=query-api-key-71")
	headers := map[string]string{
		"x-api-key":      "header-api-key-71",
		"x-goog-api-key": "header-api-key-71",
	}
	resp, err := client.DoWithHeaders(reqURL, http.MethodOptions, "", headers)
	if err != nil {
		t.Fatalf("client.DoWithHeaders(%q, OPTIONS) failed: %v", reqURL, err)
	}
	const wantResp = `{"Authorization": "", "RequestURI": "/bearertoken/deep/nested/item-71?key=query-api-key-71"}`
	if err := util.JsonEqual(wantResp, string(resp)); err != nil {
		t.Errorf("client.DoWithHeaders(%q, OPTIONS) = %q, want JSON %q: %v", reqURL, string(resp), wantResp, err)
	}

	scRequests, err := s.ServiceControlServer.GetRequests(2)
	if err != nil {
		t.Fatalf("s.ServiceControlServer.GetRequests(2) failed: %v", err)
	}
	wantSC := utils.ServiceControlExpectation{
		OperationName: wantOp,
		ConsumerID:    "api_key:query-api-key-71",
		HTTPMethod:    http.MethodOptions,
		HTTPStatus:    http.StatusOK,
	}
	if err := utils.VerifyCheckAndReportRequests(scRequests, wantSC); err != nil {
		t.Errorf("utils.VerifyCheckAndReportRequests(OPTIONS %q) failed: %v", reqURL, err)
	}
}

func setupAPIKeyEnv(t *testing.T, methodName string, rule *annotationspb.HttpRule) *env.TestEnv {
	t.Helper()
	s := env.NewTestEnv(platform.TestAPIGWServiceControlAPIKey, platform.EchoSidecar)
	t.Cleanup(func() { s.TearDown(t) })

	s.AppendAPIs([]*apipb.Api{
		{
			Name: echoV2Service,
			Methods: []*apipb.Method{
				{Name: methodName},
			},
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{rule})
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("s.Setup() failed: %v", err)
	}
	return s
}
