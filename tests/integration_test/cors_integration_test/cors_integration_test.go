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

package cors_integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
)

const (
	echoMsg = "hello"
)

// ESPv2 handles CORS with the basic preset.
// Tests only "simple requests". These do not trigger preflight OPTIONS in browsers.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#simple_requests
func TestProxyHandleCorsSimpleRequestsBasic(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(platform.TestProxyHandleCorsSimpleRequestsBasic, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc              string
		path              string
		httpMethod        string
		msg               string
		origin            string
		wantAllowOrigin   string
		wantExposeHeaders string
		wantMaxAge        string
	}{
		{
			desc:              "CORS simple request origin matches, so there are CORS headers in response.",
			path:              "/echo",
			httpMethod:        "POST",
			msg:               echoMsg,
			origin:            corsAllowOriginValue,
			wantAllowOrigin:   corsAllowOriginValue,
			wantExposeHeaders: corsExposeHeadersValue,
		},
		{
			desc:              "CORS simple request handled before frontend auth is checked for method.",
			path:              "/auth/info/googlejwt",
			httpMethod:        "GET",
			msg:               "",
			origin:            corsAllowOriginValue,
			wantAllowOrigin:   corsAllowOriginValue,
			wantExposeHeaders: corsExposeHeadersValue,
		},
		{
			desc:              "CORS simple request origin does not match, so CORS headers are NOT in the response.",
			path:              "/echo",
			httpMethod:        "POST",
			msg:               echoMsg,
			origin:            "https://some.unknown.origin.com",
			wantAllowOrigin:   "",
			wantExposeHeaders: "",
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, tc.path)
			respHeader, err := client.DoCorsSimpleRequest(url, tc.httpMethod, tc.origin, tc.msg)
			if err != nil {
				t.Fatal(err)
			}

			if respHeader.Get("Access-Control-Allow-Origin") != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", tc.wantAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
			}
			if respHeader.Get("Access-Control-Expose-Headers") != tc.wantExposeHeaders {
				t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", tc.wantExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
			}
		})
	}
}

// ESPv2 handles CORS with the regex preset.
// Tests only "simple requests". These do not trigger preflight OPTIONS in browsers.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#simple_requests
func TestProxyHandleCorsSimpleRequestsRegex(t *testing.T) {
	t.Parallel()

	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginRegex := "^https?://.+\\.google\\.com$"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=cors_with_regex",
		"--cors_allow_origin_regex=" + corsAllowOriginRegex,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(platform.TestProxyHandleCorsSimpleRequestsRegex, platform.EchoSidecar)
	// UseWrongBackendCertForDR shouldn't impact simple Cors.
	s.UseWrongBackendCertForDR(true)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc              string
		origin            string
		wantAllowOrigin   string
		wantExposeHeaders string
		wantMaxAge        string
	}{
		{
			desc:              "CORS simple request origin matches, so there are CORS headers in response.",
			origin:            "http://gcpproxy.cloud.google.com",
			wantAllowOrigin:   "http://gcpproxy.cloud.google.com",
			wantExposeHeaders: corsExposeHeadersValue,
		},
		{
			desc:              "CORS simple request origin does not match, so CORS headers are NOT in the response.",
			origin:            "http://some.unknown.origin.com",
			wantAllowOrigin:   "",
			wantExposeHeaders: "",
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo")
			respHeader, err := client.DoCorsSimpleRequest(url, "POST", tc.origin, echoMsg)
			if err != nil {
				t.Fatal(err)
			}

			if respHeader.Get("Access-Control-Allow-Origin") != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", tc.wantAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
			}
			if respHeader.Get("Access-Control-Expose-Headers") != tc.wantExposeHeaders {
				t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", tc.wantExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
			}
		})
	}
}

// ESPv2 handles CORS with the basic preset.
// Tests preflight requests. These are actual OPTIONS requests.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#preflighted_requests
func TestProxyHandlesCorsPreflightRequestsBasic(t *testing.T) {
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
	corsMaxAgeValue := "7200"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials",
	        "--cors_max_age=2h"}

	s := env.NewTestEnv(platform.TestProxyHandlesCorsPreflightRequestsBasic, platform.EchoSidecar)
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc            string
		reqHeaders      map[string]string
		wantError       string
		wantRespHeaders map[string]string
	}{
		{
			desc: "CORS preflight request is valid.",
			reqHeaders: map[string]string{
				"Origin":                         corsAllowOriginValue,
				"Access-Control-Request-Method":  corsRequestMethod,
				"Access-Control-Request-Headers": corsRequestHeader,
			},
			wantRespHeaders: map[string]string{
				"Access-Control-Allow-Origin":      corsAllowOriginValue,
				"Access-Control-Allow-Methods":     corsAllowMethodsValue,
				"Access-Control-Allow-Headers":     corsAllowHeadersValue,
				"Access-Control-Expose-Headers":    corsExposeHeadersValue,
				"Access-Control-Allow-Credentials": corsAllowCredentialsValue,
				"Access-Control-Max-Age":           corsMaxAgeValue,
			},
		},
		{
			// TODO(nareddyt): The response code here is a minor bug.
			// It's coming from the SC filter, as the CORS filters just continues
			// the pipeline when the origin mismatches.
			desc: "CORS preflight request is invalid because the origin does not match.",
			reqHeaders: map[string]string{
				"Origin":                         "https://some.unknown.origin.com",
				"Access-Control-Request-Method":  corsRequestMethod,
				"Access-Control-Request-Headers": corsRequestHeader,
			},
			wantError: `405 Method Not Allowed`,
			wantRespHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Expose-Headers":    "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			desc: "CORS preflight request is invalid because the origin is missing.",
			reqHeaders: map[string]string{
				"Access-Control-Request-Method":  corsRequestMethod,
				"Access-Control-Request-Headers": corsRequestHeader,
			},
			wantError: `{"code":400,"message":"The CORS preflight request is missing one (or more) of the following required headers: Origin, Access-Control-Request-Method"}`,
			wantRespHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Expose-Headers":    "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			desc: "CORS preflight request is invalid because the Access-Control-Request-Method is missing.",
			reqHeaders: map[string]string{
				"Origin":                         corsAllowOriginValue,
				"Access-Control-Request-Headers": corsRequestHeader,
			},
			wantError: `{"code":400,"message":"The CORS preflight request is missing one (or more) of the following required headers: Origin, Access-Control-Request-Method"}`,
			wantRespHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Expose-Headers":    "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/echo")
			respHeaders, _, err := utils.DoWithHeaders(url, "OPTIONS", "", tc.reqHeaders)

			if err != nil && tc.wantError == "" {
				t.Fatal(err)
			} else if err == nil && tc.wantError != "" {
				t.Fatalf("Want error, got no error")
			} else if err != nil && !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("\nwant error: %v, \ngot  error: %v", tc.wantError, err)
			}

			if respHeaders == nil {
				t.Fatalf("could not read response headers")
			}

			for key, value := range tc.wantRespHeaders {
				if respHeaders.Get(key) != value {
					t.Errorf("%s expected: %s, got: %s", key, value, respHeaders.Get(key))
				}
			}
		})
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

	s := env.NewTestEnv(platform.TestGrpcBackendSimpleCors, platform.GrpcBookstoreSidecar)
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
	url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/v1/shelves/200")
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

	s := env.NewTestEnv(platform.TestGrpcBackendPreflightCors, platform.GrpcBookstoreSidecar)
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
			"Access-Control-Max-Age":           "1728000",
		},
	}

	url := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/v1/shelves/200")
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

	s := env.NewTestEnv(platform.TestPreflightRequestWithAllowCors, platform.EchoSidecar)
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
			url:  fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/simplegetcors"),
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
			url:  fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/auth/info/firebase"),
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

	s := env.NewTestEnv(platform.TestServiceControlRequestWithAllowCors, platform.EchoSidecar)
	s.SetAllowCors()
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
			url:  fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/bookstore/shelves?key=api-key"),
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
					ApiKeyState:                  "VERIFIED",
					ApiVersion:                   "1.0.0",
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
			url:  fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/bookstore/shelves/1"),
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
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_GetShelf",
					ApiName:           "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiKeyState:       "NOT CHECKED",
					ApiVersion:        "1.0.0",
					ProducerProjectID: "producer-project",
					FrontendProtocol:  "http",
					HttpMethod:        "OPTIONS",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_GetShelf is called",
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

	s := env.NewTestEnv(platform.TestServiceControlRequestWithoutAllowCors, platform.EchoSidecar)
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
			url:  fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort, "/bookstore/shelves?key=api-key"),
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
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					ApiName:                      "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion:                   "1.0.0",
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

	s := env.NewTestEnv(platform.TestStartupDuplicatedPathsWithAllowCors, platform.EchoSidecar)
	s.SetAllowCors()
	s.AppendHttpRules([]*annotationspb.HttpRule{
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
