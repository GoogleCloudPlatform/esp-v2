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
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
)

const (
	echoMsg  = "hello"
	echoHost = "http://localhost:8080"
)

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

	if err := s.Setup("echo", args); err != nil {
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
	respHeader, err := client.DoCorsSimpleRequest(echoHost, corsAllowOriginValue, echoMsg)
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

	if err := s.Setup("echo", args); err != nil {
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
	respHeader, err := client.DoCorsSimpleRequest(echoHost, corsAllowOriginValue, echoMsg)
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

// TODO (jcwang) call a method requires JWT
// TODO (jcwang) using gRPC backend and making http calls

