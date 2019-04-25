// Copyright 2018 Google Cloud Platform Proxy Authors
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

package integration

import (
	"fmt"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
	"google.golang.org/genproto/googleapis/api/annotations"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

const (
	echoMsg = "hello"
)

// Simple CORS request with basic preset in config manager, response should have CORS headers
func TestSimpleCorsWithBasicPreset(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestSimpleCorsWithBasicPreset, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsDifferentOriginValue := "http://www.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestDifferentOriginSimpleCors, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "test-echo"
	configId := "test-config-id"
	corsAllowOriginRegex := "^https?://.+\\.google\\.com$"
	corsAllowOriginValue := "http://gcpproxy.cloud.google.com"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--version=" + configId, "--backend_protocol=http1",
		"--rollout_strategy=fixed", "--cors_preset=cors_with_regex",
		"--cors_allow_origin_regex=" + corsAllowOriginRegex,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestSimpleCorsWithRegexPreset, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsRequestHeader := "X-PINGOTHER"
	corsAllowOriginValue := "http://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "DNT,User-Agent,Cache-Control,Content-Type,Authorization, X-PINGOTHER"
	corsExposeHeadersValue := "Content-Length,Content-Range"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestPreflightCorsWithBasicPreset, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsAllowOriginValue := "http://cloud.google.com"
	corsOrigin := "https://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization"
	corsExposeHeadersValue := "Content-Length,Content-Range"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestDifferentOriginPreflightCors, "echo", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "bookstore-service"
	configId := "test-config-id"
	corsAllowOriginValue := "http://cloud.google.com"
	corsExposeHeadersValue := "custom-header-1,custom-header-2"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue,
		"--cors_expose_headers=" + corsExposeHeadersValue}

	s := env.NewTestEnv(comp.TestGrpcBackendSimpleCors, "bookstore", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
	serviceName := "test-echo"
	configId := "test-config-id"
	corsRequestMethod := "PATCH"
	corsAllowOriginValue := "http://cloud.google.com"
	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
	corsAllowHeadersValue := "content-type,x-grpc-web"
	corsExposeHeadersValue := "custom-header-1,custom-header-2"
	corsAllowCredentialsValue := "true"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--cors_preset=basic",
		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
		"--cors_allow_headers=" + corsAllowHeadersValue,
		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}

	s := env.NewTestEnv(comp.TestGrpcBackendPreflightCors, "bookstore", nil)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestPreflightRequestWithAllowCors, "echo", nil)
	s.SetAllowCors()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlRequestWithAllowCors, "echo", nil)
	s.SetAllowCors()
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ListShelves",
			Pattern: &annotations.HttpRule_Get{
				Get: "/bookstore/shelves",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
			Pattern: &annotations.HttpRule_Custom{
				Custom: &annotations.CustomHttpPattern{
					Kind: "OPTIONS",
					Path: "/bookstore/shelves",
				},
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
			Pattern: &annotations.HttpRule_Get{
				Get: "/bookstore/shelves/{shelf}",
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					CallerIp:        "127.0.0.1",
					Referer:         referer,
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/bookstore/shelves?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "OPTIONS",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves is called",
					Referer:           referer,
					StatusCode:        "0",
					RequestSize:       333,
					ResponseSize:      281,
					RequestBytes:      333,
					ResponseBytes:     281,
					ResponseCode:      204,
					Platform:          util.GCE,
					Location:          "test-zone",
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
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/bookstore/shelves/1",
					ApiKey:            "",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_7",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "OPTIONS",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_7 is called",
					Referer:           referer,
					StatusCode:        "0",
					RequestSize:       323,
					ResponseSize:      281,
					RequestBytes:      323,
					ResponseBytes:     281,
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
					if scRequests[i].ReqType != comp.CHECK_REQUEST {
						t.Errorf("Test Desc(%s): service control request %v: should be Check", tc.desc, i)
					}
					if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
						t.Error(err)
					}
				case *utils.ExpectedReport:
					if scRequests[i].ReqType != comp.REPORT_REQUEST {
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

func TestServiceControlRequestWithoutAllowCors(t *testing.T) {
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

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlRequestWithoutAllowCors, "echo", nil)
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.ListShelves",
			Pattern: &annotations.HttpRule_Get{
				Get: "/bookstore/shelves",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
			Pattern: &annotations.HttpRule_Custom{
				Custom: &annotations.CustomHttpPattern{
					Kind: "OPTIONS",
					Path: "/bookstore/shelves",
				},
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
			Pattern: &annotations.HttpRule_Get{
				Get: "/bookstore/shelves/{shelf}",
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

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
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					CallerIp:        "127.0.0.1",
					Referer:         referer,
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/bookstore/shelves?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "OPTIONS",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves is called",
					Referer:           referer,
					StatusCode:        "0",
					RequestSize:       333,
					ResponseSize:      281,
					RequestBytes:      333,
					ResponseBytes:     281,
					ResponseCode:      204,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:                  "Succeed, request without API key, response should fail, service control does not send report request since path matcher filter has already reject the request",
			url:                   fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/bookstore/shelves/1"),
			respHeaderMap:         map[string]string{},
			checkServiceControl:   true,
			wantScRequests:        []interface{}{},
			wantGetScRequestError: fmt.Errorf("Timeout got 0, expected: 1"),
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
					if scRequests[i].ReqType != comp.CHECK_REQUEST {
						t.Errorf("Test Desc(%s): service control request %v: should be Check", tc.desc, i)
					}
					if err := utils.VerifyCheck(reqBody, wantScRequest.(*utils.ExpectedCheck)); err != nil {
						t.Error(err)
					}
				case *utils.ExpectedReport:
					if scRequests[i].ReqType != comp.REPORT_REQUEST {
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
