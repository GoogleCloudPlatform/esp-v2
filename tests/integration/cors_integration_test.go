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
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestSimpleCorsWithBasicPreset, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

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
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, tc.path)
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestDifferentOriginSimpleCors, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := struct {
		desc       string
		corsOrigin string
	}{
		desc:       "Fail, response does not have CORS headers",
		corsOrigin: corsDifferentOriginValue,
	}
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/echo")
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestSimpleCorsWithRegexPreset, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := struct {
		desc              string
		corsAllowOrigin   string
		corsExposeHeaders string
	}{
		desc:              "Succeed, response has CORS headers",
		corsAllowOrigin:   corsAllowOriginValue,
		corsExposeHeaders: corsExposeHeadersValue,
	}
	url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/echo")
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestPreflightCorsWithBasicPreset, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

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

	url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/echo")
	respHeader, err := client.DoCorsPreflightRequest(url, corsAllowOriginValue, corsRequestMethod, corsRequestHeader)
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestDifferentOriginPreflightCors, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

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

	url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/echo")
	respHeader, err := client.DoCorsPreflightRequest(url, corsOrigin, corsRequestMethod, "")
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

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
	}

	if err := s.Setup(comp.TestPreflightRequestWithAllowCors, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	testData := []struct {
		desc          string
		url           string
		respHeaderMap map[string]string
	}{
		{
			// when allowCors, apiproxy passes preflight CORS request to backend
			desc: "Succeed, response has CORS headers",
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/simplegetcors"),
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
			url:  fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, "/auth/info/firebase"),
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
		respHeader, err := client.DoCorsPreflightRequest(tc.url, corsOrigin, corsRequestMethod, corsRequestHeader)
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

// TODO(jcwang) re-enable it later, probably it causes "bind address already in use" somehow on prow
//package integration
//
//import (
//	"testing"
//	"time"
//
//	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
//	"cloudesf.googlesource.com/gcpproxy/tests/env"
//)
//
//const (
//	bookstoreHost = "http://localhost:8080"
//)
//
//func TestGrpcBackendSimpleCors(t *testing.T) {
//	serviceName := "bookstore-service"
//	configId := "test-config-id"
//	corsAllowOriginValue := "http://cloud.google.com"
//	corsExposeHeadersValue := "custom-header-1,custom-header-2"
//
//	args := []string{"--service=" + serviceName, "--version=" + configId,
//		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--cors_preset=basic",
//		"--cors_allow_origin=" + corsAllowOriginValue,
//		"--cors_expose_headers=" + corsExposeHeadersValue}
//
//	s := env.TestEnv{
//		MockMetadata:          true,
//		MockServiceManagement: true,
//		MockServiceControl:    true,
//		MockJwtProviders:      nil,
//	}
//
//	if err := s.Setup("bookstore", args); err != nil {
//		t.Fatalf("fail to setup test env, %v", err)
//	}
//	defer s.TearDown()
//	time.Sleep(time.Duration(3 * time.Second))
//
//	testData := struct {
//		desc              string
//		corsAllowOrigin   string
//		corsExposeHeaders string
//	}{
//		desc:              "Succeed, response has CORS headers",
//		corsAllowOrigin:   corsAllowOriginValue,
//		corsExposeHeaders: corsExposeHeadersValue,
//	}
//	respHeader, err := client.DoCorsSimpleRequest(bookstoreHost+"/v1/shelves/200", "GET", corsAllowOriginValue, "")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if respHeader.Get("Access-Control-Allow-Origin") != testData.corsAllowOrigin {
//		t.Errorf("Access-Control-Allow-Origin expected: %s, got: %s", testData.corsAllowOrigin, respHeader.Get("Access-Control-Allow-Origin"))
//	}
//	if respHeader.Get("Access-Control-Expose-Headers") != testData.corsExposeHeaders {
//		t.Errorf("Access-Control-Expose-Headers expected: %s, got: %s", testData.corsExposeHeaders, respHeader.Get("Access-Control-Expose-Headers"))
//	}
//}
//
//func TestGrpcBackendPreflightCors(t *testing.T) {
//	serviceName := "test-echo"
//	configId := "test-config-id"
//	corsRequestMethod := "PATCH"
//	corsAllowOriginValue := "http://cloud.google.com"
//	corsAllowMethodsValue := "GET, PATCH, DELETE, OPTIONS"
//	corsAllowHeadersValue := "content-type,x-grpc-web"
//	corsExposeHeadersValue := "custom-header-1,custom-header-2"
//	corsAllowCredentialsValue := "true"
//
//	args := []string{"--service=" + serviceName, "--version=" + configId,
//		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--cors_preset=basic",
//		"--cors_allow_origin=" + corsAllowOriginValue, "--cors_allow_methods=" + corsAllowMethodsValue,
//		"--cors_allow_headers=" + corsAllowHeadersValue,
//		"--cors_expose_headers=" + corsExposeHeadersValue, "--cors_allow_credentials"}
//
//	s := env.TestEnv{
//		MockMetadata:          true,
//		MockServiceManagement: true,
//		MockServiceControl:    true,
//		MockJwtProviders:      nil,
//	}
//
//	if err := s.Setup("bookstore", args); err != nil {
//		t.Fatalf("fail to setup test env, %v", err)
//	}
//	defer s.TearDown()
//	time.Sleep(time.Duration(3 * time.Second))
//
//	testData := struct {
//		desc          string
//		respHeaderMap map[string]string
//	}{
//		desc:          "Succeed, response has CORS headers",
//		respHeaderMap: make(map[string]string),
//	}
//	testData.respHeaderMap["Access-Control-Allow-Origin"] = corsAllowOriginValue
//	testData.respHeaderMap["Access-Control-Allow-Methods"] = corsAllowMethodsValue
//	testData.respHeaderMap["Access-Control-Allow-Headers"] = corsAllowHeadersValue
//	testData.respHeaderMap["Access-Control-Expose-Headers"] = corsExposeHeadersValue
//	testData.respHeaderMap["Access-Control-Allow-Credentials"] = corsAllowCredentialsValue
//
//	respHeader, err := client.DoCorsPreflightRequest(bookstoreHost+"/v1/shelves/200", corsAllowOriginValue, corsRequestMethod, "")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	for key, value := range testData.respHeaderMap {
//		if respHeader.Get(key) != value {
//			t.Errorf("%s expected: %s, got: %s", key, value, respHeader.Get(key))
//		}
//	}
//}
