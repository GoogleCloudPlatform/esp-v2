// Copyright 2019 Google Cloud Platform Proxy Authors
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

package service_control_api_key_location_test

import (
	"fmt"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlAPIKeyDefaultLocation(t *testing.T) {

	configId := "test-config-id"
	args := []string{"--service_config_id=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlAPIKeyDefaultLocation, "echo")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                  string
		url                   string
		method                string
		requestHeader         map[string]string
		message               string
		wantResp              string
		wantApiKey            string
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:       "succeed, use the default apiKey location(key in query)",
			url:        fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "api-key",
		},
		{
			desc:       "succeed, use the default apiKey location(api_key in query)",
			url:        fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?api_key=api-key-1"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "api-key-1",
		},
		{
			desc:       "succeed, use two apiKey locations in the same time(api_key and key in query)",
			url:        fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?api_key=api-key-2&key=key-2"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-2",
		},
		{
			desc:    "succeed, use the default apiKey location(X-API-KEY in header)",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:  "POST",
			message: "hello",
			requestHeader: map[string]string{
				"X-API-KEY": "key-3",
			},
			wantApiKey: "key-3",
		},
	}
	for _, tc := range testData {
		resp, err := client.DoWithHeaders(tc.url, tc.method, tc.message, tc.requestHeader)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(2)
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckAPIKey(t, scRequests[0], tc.wantApiKey, tc.desc)
	}
}

func TestServiceControlAPIKeyCustomLocation(t *testing.T) {

	serviceName := "test-echo"
	configId := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlAPIKeyCustomLocation, "echo")
	s.OverrideSystemParameters(&conf.SystemParameters{
		Rules: []*conf.SystemParameterRule{
			{
				Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				Parameters: []*conf.SystemParameter{
					{
						Name:              "api_key",
						HttpHeader:        "Header-Name-1",
						UrlQueryParameter: "query_name_1",
					},
					{
						Name:              "api_key",
						HttpHeader:        "Header-Name-2",
						UrlQueryParameter: "query_name_2",
					},
				},
			},
		},
	})
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc          string
		url           string
		method        string
		requestHeader map[string]string
		message       string
		wantResp      string
		wantApiKey    string
	}{
		{
			desc:       "Succeed, single apikey passed by url query",
			url:        fmt.Sprintf("http://localhost:%v%v?query_name_1=key-1", s.Ports().ListenerPort, "/echo"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-1",
		},
		{
			desc:       "succeed, single apikey passed by url query",
			url:        fmt.Sprintf("http://localhost:%v%v?query_name_2=api-key-1", s.Ports().ListenerPort, "/echo"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "api-key-1",
		},
		// In the SystemParameters, query_name_1 is defined before query_name_2 so query_name_1=key-31 is applied first.
		{
			desc:       "succeed, two apikeys are passed by url query",
			url:        fmt.Sprintf("http://localhost:%v%v?query_name_1=key-31&query_name_2=key-32", s.Ports().ListenerPort, "/echo"),
			method:     "POST",
			message:    "hello",
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-31",
		},
		{
			desc:    "succeed, single apikey passed by headers",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:  "POST",
			message: "hello",
			requestHeader: map[string]string{
				"HEADER-NAME-1": "key-4",
			},
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-4",
		},
		{
			desc:    "succeed, single apike passed by headers",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:  "POST",
			message: "hello",
			requestHeader: map[string]string{
				"HEADER-NAME-2": "key-5",
			},
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-5",
		},
		// In the SystemParameters, HEADER-NAME-1 is defined before HEADER-NAME-2 so HEADER-NAME-1=key-61 is applied first.
		{
			desc:    "succeed, two apikeys are passed by headers",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:  "POST",
			message: "hello",
			requestHeader: map[string]string{
				"HEADER-NAME-1": "key-61",
				"HEADER-NAME-2": "key-62",
			},
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-61",
		},
		// The proxy will look into all the custom-defined apikey locations in the url query and then those in the header.
		// The query_name_1 is the first location for the url query so it will be applied.
		{
			desc:    "succeed, four apikeys are passed by both url query and headers",
			url:     fmt.Sprintf("http://localhost:%v%v?query_name_2=api-key-72&query_name_1=key-71", s.Ports().ListenerPort, "/echo"),
			method:  "POST",
			message: "hello",
			requestHeader: map[string]string{
				"HEADER-NAME-1": "key-73",
				"HEADER-NAME-2": "key-74",
			},
			wantResp:   `{"message":"hello"}`,
			wantApiKey: "key-71",
		},
	}
	for _, tc := range testData {
		resp, err := client.DoWithHeaders(tc.url, tc.method, tc.message, tc.requestHeader)
		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(2)
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckAPIKey(t, scRequests[0], tc.wantApiKey, tc.desc)
	}
}
