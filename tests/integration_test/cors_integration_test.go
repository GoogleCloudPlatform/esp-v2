// Copyright 2019 Google LLC
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

package integration_test

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
)

const (
	echoMsg = "hello"
)

// Simple CORS request with basic preset in config manager, response should have CORS headers
func TestSimpleCorsWithBasicPreset(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestSimpleCorsWithBasicPreset, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc              string
		path              string
		httpMethod        string
		msg               string
		corsAllowOrigin   string
		corsExposeHeaders string
	}{
		{
			desc:              "Succeed, response has CORS headers",
			path:              "/echo",
			httpMethod:        "POST",
			msg:               echoMsg,
			corsAllowOrigin:   corsAllowOriginValue,
			corsExposeHeaders: corsExposeHeadersValue,
		},
		{
			// send to an endpoint that requires JWT, response still has CORS headers though the request does not pass through jwt filter
			desc:              "Succeed, response has CORS headers",
			path:              "/auth/info/googlejwt",
			httpMethod:        "GET",
			msg:               "",
			corsAllowOrigin:   corsAllowOriginValue,
			corsExposeHeaders: corsExposeHeadersValue,
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		respHeader, err := client.DoCorsSimpleRequest(url, tc.httpMethod, corsAllowOriginValue, tc.msg)
		if err != nil {
			t.Fatal(err)
		}

		if respHeader.Get("Access-Control-Allow-Origin") != tc.corsAllowOrigin {
			t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", tc.corsAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
		}
		if respHeader.Get("Access-Control-Expose-Headers") != tc.corsExposeHeaders {
			t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", tc.corsExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
		}
	}
}

// CORS request Origin is different from cors_allow_origin setting in config manager
// since these two does not match, envoy CORS filter does not put CORS headers in response
func TestDifferentOriginSimpleCors(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsDifferentOriginValue := "http://www.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestDifferentOriginSimpleCors, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc       string
		corsOrigin string
	}{
		desc:       "Fail, response does not have CORS headers",
		corsOrigin: corsDifferentOriginValue,
	}
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo")
	respHeader, err := client.DoCorsSimpleRequest(url, "POST", testData.corsOrigin, echoMsg)
	if err != nil {
		t.Fatal(err)
	}

	if respHeader.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Access-Control-Allow-Origin expected to be empty string, got: %s", respHeader.Get("Access-Control-Allow-Origin"))
	}
	if respHeader.Get("Access-Control-Expose-Headers") != "" {
		t.Errorf("Access-Control-Expose-Headers expected to be empty string, got: %s", respHeader.Get("Access-Control-Expose-Headers"))
	}
}

// Simple CORS request with regex origin in config manager, response should have CORS headers
func TestSimpleCorsWithRegexPreset(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginRegex := "^https?://.+\\.google\\.com$"
	corsAllowOriginValue := "http://gcpproxy.cloud.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=cors_with_regex",
		"--cors_allow_origin_regex=" + corsAllowOriginRegex,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestSimpleCorsWithRegexPreset, platform.EchoSidecar)
	// UseWrongBackendCertForDR shouldn't impact simple Cors.
	s.UseWrongBackendCertForDR(true)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc              string
		corsAllowOrigin   string
		corsExposeHeaders string
	}{
		desc:              "Succeed, response has CORS headers",
		corsAllowOrigin:   corsAllowOriginValue,
		corsExposeHeaders: corsExposeHeadersValue,
	}
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo")
	respHeader, err := client.DoCorsSimpleRequest(url, "POST", corsAllowOriginValue, echoMsg)
	if err != nil {
		t.Fatal(err)
	}

	if respHeader.Get("Access-Control-Allow-Origin") != testData.corsAllowOrigin {
		t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", testData.corsAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
	}
	if respHeader.Get("Access-Control-Expose-Headers") != testData.corsExposeHeaders {
		t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", testData.corsExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
	}
}

// Preflight CORS request with basic preset in config manager, response should have CORS headers
func TestPreflightCorsWithBasicPreset(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsAllowOriginValue := "http://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "DNT,User-Agent,Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsExposeHeadersValue := "Content-Length,Content-Range"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestPreflightCorsWithBasicPreset, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc          string
		respHeaderMap map[string]string
	}{
		desc: "Succeed, response has CORS headers",
		respHeaderMap: map[string]string{
			"Access-Control-Allow-Origin":      corsAllowOriginValue,
			"Access-Control-Allow-Methods":     corsAllowMethodsValue,
			"Access-Control-Allow-Headers":     corsAllowHeadersValue,
			"Access-Control-Expose-Headers":    corsExposeHeadersValue,
			"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
		},
	}

	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo")
	respHeader, err := client.DoCorsPreflightRequest(url, corsAllowOriginValue, corsRequestMethod, corsRequestHeader, "")
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range testData.respHeaderMap {
		if respHeader.Get(key) != value {
			t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
		}
	}

}

// Preflight request Origin is different from cors_allow_origin setting in config manager
// since these two does not match, envoy CORS filter does not put CORS headers in response
func TestDifferentOriginPreflightCors(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsAllowOriginValue := "http://cloud.google.com"
	corsOrigin := "https://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestDifferentOriginPreflightCors, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc          string
		respHeaderMap map[string]string
	}{
		desc: "Fail, response does not have CORS headers",
		respHeaderMap: map[string]string{
			"Access-Control-Allow-Origin":      "",
			"Access-Control-Allow-Methods":     "",
			"Access-Control-Allow-Headers":     "",
			"Access-Control-Expose-Headers":    "",
			"Access-Control-Allow-Credentials": "",
		},
	}

	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo")
	respHeader, err := client.DoCorsPreflightRequest(url, corsOrigin, corsRequestMethod, "", "")
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range testData.respHeaderMap {
		if respHeader.Get(key) != value {
			t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
		}
	}
}

// Simple CORS request with GRPC backend and basic preset in config manager, response should have CORS headers
func TestGrpcBackendSimpleCors(t *testing.T) {
	t.Parallel()

	serviceName := "bookstore-service"
	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsExposeHeadersValue := "custom-header-1,custom-header-2"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestGrpcBackendSimpleCors, platform.GrpcBookstoreSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc              string
		corsAllowOrigin   string
		corsExposeHeaders string
	}{
		desc:              "Succeed, response has CORS headers",
		corsAllowOrigin:   corsAllowOriginValue,
		corsExposeHeaders: corsExposeHeadersValue,
	}
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/v1/shelves/200")
	respHeader, err := client.DoCorsSimpleRequest(url, "GET", corsAllowOriginValue, "")
	if err != nil {
		t.Fatal(err)
	}

	if respHeader.Get("Access-Control-Allow-Origin") != testData.corsAllowOrigin {
		t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", testData.corsAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
	}
	if respHeader.Get("Access-Control-Expose-Headers") != testData.corsExposeHeaders {
		t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", testData.corsExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
	}
}

// Preflight CORS request with GRPC backend and basic preset in config manager, response should have CORS headers
func TestGrpcBackendPreflightCors(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsAllowOriginValue := "http://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "content-type,x-grpc-web"
	corsExposeHeadersValue := "custom-header-1,custom-header-2"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestGrpcBackendPreflightCors, platform.GrpcBookstoreSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := struct {
		desc          string
		respHeaderMap map[string]string
	}{
		desc: "Succeed, response has CORS headers",
		respHeaderMap: map[string]string{
			"Access-Control-Allow-Origin":      corsAllowOriginValue,
			"Access-Control-Allow-Methods":     corsAllowMethodsValue,
			"Access-Control-Allow-Headers":     corsAllowHeadersValue,
			"Access-Control-Expose-Headers":    corsExposeHeadersValue,
			"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
		},
	}

	url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/v1/shelves/200")
	respHeader, err := client.DoCorsPreflightRequest(url, corsAllowOriginValue, corsRequestMethod, "", "")
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range testData.respHeaderMap {
		if respHeader.Get(key) != value {
			t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
		}
	}
}

// Preflight CORS request with allowCors to allow backends to receive and respond to OPTIONS requests
func TestPreflightRequestWithAllowCors(t *testing.T) {
	t.Parallel()

	serviceName := "echo-api.endpoints.cloudesf-testing.cloud.goog"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsOrigin := "http://cloud.google.com"
	corsAllowOriginValue := "*"
	corsAllowMethodsValue := "GET, OPTIONS"
	corsAllowHeadersValue := "Authorization"
	corsExposeHeadersValue := "Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestPreflightRequestWithAllowCors, platform.EchoSidecar)
	s.SetAllowCors()
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		url           string
		respHeaderMap map[string]string
	}{
		{
			// when allowCors, apiproxy passes preflight CORS request to backend
			desc: "Succeed, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/simplegetcors"),
			respHeaderMap: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
			},
		},
		{
			// when allowCors, apiproxy passes preflight CORS request without valid jwt token to backend,
			// even the origin method requires authentication
			desc: "Succeed without jwt token, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/auth/info/firebase"),
			respHeaderMap: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
			},
		},
	}
	for _, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader, "")
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tc.respHeaderMap {
			if respHeader.Get(key) != value {
				t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
			}
		}
	}
}

func TestServiceControlRequestWithAllowCors(t *testing.T) {
	t.Parallel()

	serviceName := "echo-api.endpoints.cloudesf-testing.cloud.goog"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	referer := "http://google.com/bootstore/root"
	corsOrigin := "http://cloud.google.com"
	corsAllowOriginValue := "*"
	corsAllowMethodsValue := "GET, OPTIONS"
	corsAllowHeadersValue := "Authorization"
	corsExposeHeadersValue := "Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlRequestWithAllowCors, platform.EchoSidecar)
	s.SetAllowCors()
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ListShelves",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/bookstore/shelves",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: "OPTIONS",
					Path: "/bookstore/shelves",
				},
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/bookstore/shelves/{shelf}",
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                string
		url                 string
		respHeaderMap       map[string]string
		checkServiceControl bool
		wantScRequests      []interface{}
	}{
		{
			desc: "Succeed, response has CORS headers, service control sends check and report request",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/bookstore/shelves?key=api-key"),
			respHeaderMap: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
			},
			checkServiceControl: true,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					CallerIp:        platform.GetLoopbackAddress(),
					Referer:         referer,
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/bookstore/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "OPTIONS",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves is called",
					Referer:                      referer,
					StatusCode:                   "0",
					ResponseCode:                 204,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc: "Succeed, request without API key, response has CORS headers, service control sends report request only",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/bookstore/shelves/1"),
			respHeaderMap: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
			},
			checkServiceControl: true,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/bookstore/shelves/1",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_bookstore_shelves_shelf",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "OPTIONS",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_bookstore_shelves_shelf is called",
					Referer:           referer,
					StatusCode:        "0",
					ResponseCode:      204,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader, referer)
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tc.respHeaderMap {
			if respHeader.Get(key) != value {
				t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
			}
		}
		if tc.checkServiceControl {
			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("Test Desc(%s): GetRequests returns error: %v", tc.desc, err)
			}
			for i, wantScRequest := range tc.wantScRequests {
				reqBody := scRequests[i].ReqBody
				switch wantScRequest.(type) {
				case *utils.ExpectedCheck:
					if scRequests[i].ReqType != utils.CheckRequest {
						t.Errorf("Test Desc(%s): service control request %v: should be Check", tc.desc, i)
					}
					if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
						t.Error(err)
					}
				case *utils.ExpectedReport:
					if scRequests[i].ReqType != utils.ReportRequest {
						t.Errorf("Test Desc(%s): service control request %v: should be Report", tc.desc, i)
					}
					if err := utils.VerifyReport(reqBody, wantScRequest.(*utils.ExpectedReport)); err != nil {
						t.Errorf("Test Desc(%s): got err %v", tc.desc, err)
					}
				default:
					t.Fatalf("Test Desc(%s): unknown service control response type", tc.desc)
				}
			}
		}
	}
}

func TestServiceControlRequestWithoutAllowCors(t *testing.T) {
	t.Parallel()

	serviceName := "echo-api.endpoints.cloudesf-testing.cloud.goog"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	referer := "http://google.com/bootstore/root"
	corsOrigin := "http://cloud.google.com"
	corsAllowOriginValue := "*"
	corsAllowMethodsValue := "GET, OPTIONS"
	corsAllowHeadersValue := "Authorization"
	corsExposeHeadersValue := "Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlRequestWithoutAllowCors, platform.EchoSidecar)
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ListShelves",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/bookstore/shelves",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Kind: "OPTIONS",
					Path: "/bookstore/shelves",
				},
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/bookstore/shelves/{shelf}",
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                  string
		url                   string
		respHeaderMap         map[string]string
		checkServiceControl   bool
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc: "Succeed, request has API key, response has CORS headers, service control sends check and report request",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/bookstore/shelves?key=api-key"),
			respHeaderMap: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
			},
			checkServiceControl: true,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					CallerIp:        platform.GetLoopbackAddress(),
					Referer:         referer,
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/bookstore/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ProducerProjectID:            "producer-project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					HttpMethod:                   "OPTIONS",
					LogMessage:                   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves is called",
					Referer:                      referer,
					StatusCode:                   "0",
					ResponseCode:                 204,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader, referer)
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tc.respHeaderMap {
			if respHeader.Get(key) != value {
				t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
			}
		}
		if tc.checkServiceControl {
			if tc.wantGetScRequestError != nil {
				scRequests, err := s.ServiceControlServer.GetRequests(1)
				if err == nil || err.Error() != tc.wantGetScRequestError.Error() {
					t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err)
					t.Errorf("got service control requests: %v", scRequests)
				}
				continue
			}
			scRequests, err := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err != nil {
				t.Fatalf("Test Desc(%s): GetRequests returns error: %v", tc.desc, err)
			}
			for i, wantScRequest := range tc.wantScRequests {
				reqBody := scRequests[i].ReqBody
				switch wantScRequest.(type) {
				case *utils.ExpectedCheck:
					if scRequests[i].ReqType != utils.CheckRequest {
						t.Errorf("Test Desc(%s): service control request %v: should be Check", tc.desc, i)
					}
					if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
						t.Error(err)
					}
				case *utils.ExpectedReport:
					if scRequests[i].ReqType != utils.ReportRequest {
						t.Errorf("Test Desc(%s): service control request %v: should be Report", tc.desc, i)
					}
					if err := utils.VerifyReport(reqBody, wantScRequest.(*utils.ExpectedReport)); err != nil {
						t.Error(err)
					}
				default:
					t.Fatalf("Test Desc(%s): unknown service control response type", tc.desc)
				}
			}
		}
	}
}

// Test case to reproduce: https://github.com/GoogleCloudPlatform/esp-v2/issues/254
func TestStartupDuplicatedPathsWithAllowCors(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(comp.TestStartupDuplicatedPathsWithAllowCors, platform.EchoSidecar)
	s.SetAllowCors()
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/bookstore/shelves/{shelf}",
			},
		},
		{
			// URL is exactly the same even though method differs.
			// When the bug was reported, our code already handles this.
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.UpdateShelf",
			Pattern: &annotationspb.HttpRule_Patch{
				Patch: "/bookstore/shelves/{shelf}",
			},
		},
		{
			// URL is semantically the same, but path parameter names differ.
			// When the bug was reported, our code did NOT handle this.
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.DeleteShelf",
			Pattern: &annotationspb.HttpRule_Delete{
				Delete: "/bookstore/shelves/{shelf_with_different_path_parameter}",
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
}
