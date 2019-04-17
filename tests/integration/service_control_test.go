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
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
	"google.golang.org/genproto/googleapis/api/annotations"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlBasic(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlBasic, "echo", []string{"google_jwt"})
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
			Pattern: &annotations.HttpRule_Get{
				Get: "/simpleget",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
			Pattern: &annotations.HttpRule_Get{
				Get: "/echo/nokey/OverrideAsGet",
			},
		},
	})
	s.AppendUsageRules(
		[]*conf.UsageRule{
			{
				Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
				AllowUnregisteredCalls: true,
			},
		})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc                  string
		url                   string
		method                string
		requestHeader         map[string]string
		message               string
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:     "succeed GET, no Jwt required, service control sends check request and report request for GET request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/simpleget", "?key=api-key"),
			method:   "GET",
			message:  "",
			wantResp: "simple get message",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					CallerIp:        "127.0.0.1",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/simpleget?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget is called",
					StatusCode:        "0",
					RequestSize:       178,
					ResponseSize:      125,
					RequestBytes:      178,
					ResponseBytes:     125,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, service control sends check request and report request for POST request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        "127.0.0.1",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:        "0",
					RequestSize:       238,
					ResponseSize:      126,
					RequestBytes:      238,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:          "succeed, no Jwt required, service control sends report request only for request that does not have API key but is actually required",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:        "POST",
			message:       "hello",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 401 Unauthorized"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo",
					ErrorType:         "4xx",
					StatusCode:        "16",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					RequestSize:       206,
					ResponseSize:      281,
					RequestBytes:      206,
					ResponseBytes:     281,
					ResponseCode:      401,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, allow no api key (unregistered request), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					RequestSize:       232,
					ResponseSize:      126,
					RequestBytes:      232,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, allow no api key (there is API key in request though it is not required), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo/nokey", "?key=api-key"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey?key=api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					RequestSize:       244,
					ResponseSize:      126,
					RequestBytes:      244,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:    "succeed with request with referer header, no Jwt required, allow no api key (unregistered request), service control sends report (with referer information) only",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
			message: "hi",
			method:  "POST",
			requestHeader: map[string]string{
				"Referer": "http://google.com/bookstore/root",
			},
			wantResp: `{"message":"hi"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					Referer:           "http://google.com/bookstore/root",
					StatusCode:        "0",
					RequestSize:       268,
					ResponseSize:      123,
					RequestBytes:      268,
					ResponseBytes:     123,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:    "succeed, no Jwt required,no api key required, with X-HTTP-Method-Override as GET, service control sends report only and it has GET as HTTP method",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey/OverrideAsGet"),
			message: "hello hello",
			method:  "POST",
			requestHeader: map[string]string{
				"X-HTTP-Method-Override": "GET",
			},
			wantResp: `{"message":"hello hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey/OverrideAsGet",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "GET",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get is called",
					StatusCode:        "0",
					RequestSize:       277,
					ResponseSize:      132,
					RequestBytes:      277,
					ResponseBytes:     132,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed for unconfigured requests with any path (/**) and POST method, no JWT required, service control sends report request only",
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/anypath/x/y/z"),
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/anypath/x/y/z",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath is called",
					StatusCode:        "0",
					RequestSize:       235,
					ResponseSize:      126,
					RequestBytes:      235,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:                  "fail for not allowing unconfigured GET method, service control does not send report request since path matcher filter has already reject the request",
			url:                   fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/unconfiguredRequest/get"),
			method:                "GET",
			httpCallError:         fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
			wantScRequests:        []interface{}{},
			wantGetScRequestError: fmt.Errorf("Timeout got 0, expected: 1"),
		},
	}
	for _, tc := range testData {
		var resp []byte
		var err error
		if tc.method == "POST" {
			resp, err = client.DoPostWithHeaders(tc.url, tc.message, tc.requestHeader)
		} else if tc.method == "GET" {
			resp, err = client.DoGet(tc.url)
		} else {
			t.Fatal("unknown HTTP method to call")
		}

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
			}
		} else {
			if tc.httpCallError.Error() != err.Error() {
				t.Errorf("expected Http call error: %v, got: %v", tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1, 3*time.Second)
			if err1 == nil || err1.Error() != tc.wantGetScRequestError.Error() {
				t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err1)
				t.Errorf("got service control requests: %v", scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests), 3*time.Second)
		if err1 != nil {
			t.Fatalf("GetRequests returns error: %v", err1)
		}

		utils.CheckScRequest(t, scRequests, tc.wantScRequests)
	}
}

func TestServiceControlCache(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlCache, "echo", []string{"google_jwt"})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	url := fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key")
	message := "hello"
	num := 10
	wantResp := `{"message":"hello"}`
	for i := 0; i < num; i++ {
		resp, err := client.DoPost(url, message)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(resp), wantResp) {
			t.Errorf("expected: %s, got: %s", wantResp, string(resp))
		}
	}

	wantScRequests := []interface{}{
		&utils.ExpectedCheck{
			Version:         utils.APIProxyVersion,
			ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:api-key",
			OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			CallerIp:        "127.0.0.1",
		},
		&utils.ExpectedReport{
			Aggregate:         int64(num),
			Version:           utils.APIProxyVersion,
			ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:   "test-config-id",
			URL:               "/echo?key=api-key",
			ApiKey:            "api-key",
			ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			ProducerProjectID: "producer-project",
			ConsumerProjectID: "123456",
			FrontendProtocol:  "http",
			HttpMethod:        "POST",
			LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
			StatusCode:        "0",
			RequestSize:       238,
			ResponseSize:      126,
			RequestBytes:      238,
			ResponseBytes:     126,
			ResponseCode:      200,
			Platform:          util.GCE,
			Location:          "test-zone",
		},
	}

	scRequests, err := s.ServiceControlServer.GetRequests(len(wantScRequests), 3*time.Second)
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}

	utils.CheckScRequest(t, scRequests, wantScRequests)
}
