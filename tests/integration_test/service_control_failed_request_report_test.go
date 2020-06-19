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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestServiceControlFailedRequestReport(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}
	s := env.NewTestEnv(comp.TestServiceControlFailedRequestReport, platform.GrpcBookstoreSidecar)
	defer s.TearDown(t)

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		url            string
		httpMethod     string
		method         string
		token          string
		requestHeader  map[string]string
		message        string
		wantResp       string
		httpCallError  string
		wantScRequests []interface{}
	}{
		{
			desc:          "For the request(with api key) not matching any requests, return \"Not Found\" status and send report",
			url:           fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			httpMethod:    "GET",
			method:        "/noexistoperation?key=api-key",
			httpCallError: `404 Not Found, {"code":404,"message":"Path does not match any requirement URI template."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/noexistoperation?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "<Unknown Operation Name>",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "<Unknown Operation Name> is called",
					StatusCode:        "0",
					ResponseCode:      404,
					ErrorType:         "4xx",
					Platform:          util.GCE,
					Location:          "test-zone",
					BackendProtocol:   "grpc",
				},
			},
		},
		{
			desc:          "For the request(without api key) not matching any requests, return \"Not Found\" status and send report",
			url:           fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			httpMethod:    "GET",
			method:        "/noexistoperation",
			httpCallError: `404 Not Found, {"code":404,"message":"Path does not match any requirement URI template."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/noexistoperation",
					ApiMethod:         "<Unknown Operation Name>",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "<Unknown Operation Name> is called",
					StatusCode:        "0",
					ResponseCode:      404,
					ErrorType:         "4xx",
					Platform:          util.GCE,
					Location:          "test-zone",
					BackendProtocol:   "grpc",
				},
			},
		},
		{
			desc:          "For the request failed in Jwt Authn filter, return \"Unauthorized\" status and send report",
			url:           fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			httpMethod:    "GET",
			method:        "/v1/shelves?key=api-key",
			httpCallError: `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves?key=api-key",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:           "endpoints.examples.bookstore.Bookstore",
					ApiKey:            "api-key",
					ApiVersion:        "1.0.0",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					ResponseCode:      401,
					ErrorType:         "4xx",
					Platform:          util.GCE,
					Location:          "test-zone",
					BackendProtocol:   "grpc",
				},
			},
		},
		{
			desc:          "For the request without api key but required to have, return \"Unauthorized\" status and send report",
			url:           fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			token:         testdata.Es256Token,
			httpMethod:    "GET",
			method:        "/v1/shelves/0/books/0",
			httpCallError: `401 Unauthorized, {"code":401,"message":"UNAUTHENTICATED:Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves/0/books/0",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.GetBook",
					ApiName:           "endpoints.examples.bookstore.Bookstore",
					ApiVersion:        "1.0.0",
					ProducerProjectID: "producer project",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.GetBook is called",
					StatusCode:        "16",
					ResponseCode:      401,
					ErrorType:         "4xx",
					ErrorCause:        "Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API.",
					Platform:          util.GCE,
					Location:          "test-zone",
					BackendProtocol:   "grpc",
				},
			},
		},
	}
	for _, tc := range testData {
		_, err := client.MakeCall("http", tc.url, tc.httpMethod, tc.method, tc.token, nil)
		if err == nil || !strings.Contains(err.Error(), tc.httpCallError) {
			t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
