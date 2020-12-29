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
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestServiceControlFailedRequestReport(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"
	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}
	s := env.NewTestEnv(platform.TestServiceControlBasic, platform.GrpcBookstoreSidecar)
	defer s.TearDown(t)

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		url            string
		clientProtocol string
		httpMethod     string
		method         string
		headers        http.Header
		token          string
		requestHeader  map[string]string
		message        string
		wantResp       string
		httpCallError  string
		wantScRequests []interface{}
	}{
		//{
		//	desc:           "Request with API Key does not match any operation. SC does report with untrusted API Key.",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	clientProtocol: "http",
		//	httpMethod:     "GET",
		//	method:         "/noexistoperation?key=api-key",
		//	httpCallError:  "404 Not Found, {\"code\":404,\"message\":\"The current request is not defined by this API.\"}",
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedReport{
		//			Version:         utils.ESPv2Version(),
		//			ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID: "test-config-id",
		//			URL:             "/noexistoperation?key=api-key",
		//			ApiMethod:       "<Unknown Operation Name>",
		//			// API Key is extracted but not trusted.
		//			ApiKeyInLogEntryOnly: "api-key",
		//			ApiKeyState:          "NOT CHECKED",
		//			ProducerProjectID:    "producer project",
		//			FrontendProtocol:     "http",
		//			HttpMethod:           "GET",
		//			LogMessage:           "<Unknown Operation Name> is called",
		//			StatusCode:           "0",
		//			ResponseCode:         404,
		//			Platform:             util.GCE,
		//			Location:             "test-zone",
		//			BackendProtocol:      "grpc",
		//			ResponseCodeDetail:   "direct_response",
		//		},
		//	},
		//},
		{
			desc:           "Request matches uri template(exact path) but not method.",
			url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			clientProtocol: "http",
			// "DELETE" is not defined for "/v1/shelves".
			httpMethod:    "DELETE",
			method:        "/v1/shelves?key=api-key",
			httpCallError: "405 Method Not Allowed, {\"code\":405,\"message\":\"The current request is matched to the defined url template \"/v1/shelves\" but its http method is not allowed\"}",
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/v1/shelves?key=api-key",
					ApiMethod:       "<Unknown Operation Name>",
					// API Key is extracted but not trusted.
					ApiKeyInLogEntryOnly: "api-key",
					ApiKeyState:          "NOT CHECKED",
					ProducerProjectID:    "producer project",
					FrontendProtocol:     "http",
					HttpMethod:           "DELETE",
					LogMessage:           "<Unknown Operation Name> is called",
					StatusCode:           "0",
					ResponseCode:         405,
					Platform:             util.GCE,
					Location:             "test-zone",
					BackendProtocol:      "grpc",
					ResponseCodeDetail:   "direct_response",
				},
			},
		},
		{
			desc:           "Request matches uri template(regex) but not method.",
			url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
			clientProtocol: "http",
			// "DELETE" is not defined for "/v1/shelves".
			httpMethod:    "POST",
			method:        "/v1/shelves/100?key=api-key",
			httpCallError: "405 Method Not Allowed, {\"code\":405,\"message\":\"The current request is matched to the defined url template \"/v1/shelves/{shelf}\" but its http method is not allowed\"}",
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/v1/shelves/100?key=api-key",
					ApiMethod:       "<Unknown Operation Name>",
					// API Key is extracted but not trusted.
					ApiKeyInLogEntryOnly: "api-key",
					ApiKeyState:          "NOT CHECKED",
					ProducerProjectID:    "producer project",
					FrontendProtocol:     "http",
					HttpMethod:           "POST",
					LogMessage:           "<Unknown Operation Name> is called",
					StatusCode:           "0",
					ResponseCode:         405,
					Platform:             util.GCE,
					Location:             "test-zone",
					BackendProtocol:      "grpc",
					ResponseCodeDetail:   "direct_response",
				},
			},
		},
		//{
		//	desc:           "Request withOUT API Key does not match any operation. SC does report withOUT API Key.",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	clientProtocol: "http",
		//	httpMethod:     "GET",
		//	method:         "/noexistoperation",
		//	httpCallError:  "404 Not Found, {\"code\":404,\"message\":\"The current request is not defined by this API.\"}",
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedReport{
		//			Version:         utils.ESPv2Version(),
		//			ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID: "test-config-id",
		//			URL:             "/noexistoperation",
		//			ApiMethod:       "<Unknown Operation Name>",
		//			// API Key is not present.
		//			ProducerProjectID:  "producer project",
		//			ApiKeyState:        "NOT CHECKED",
		//			FrontendProtocol:   "http",
		//			HttpMethod:         "GET",
		//			LogMessage:         "<Unknown Operation Name> is called",
		//			StatusCode:         "0",
		//			ResponseCode:       404,
		//			Platform:           util.GCE,
		//			Location:           "test-zone",
		//			BackendProtocol:    "grpc",
		//			ResponseCodeDetail: "direct_response",
		//		},
		//	},
		//},
		//{
		//	desc:           "For the request failed in Jwt Authn filter, return \"Unauthorized\" status and send report",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	clientProtocol: "http",
		//	httpMethod:     "GET",
		//	method:         "/v1/shelves?key=api-key",
		//	httpCallError:  `401 Unauthorized, {"code":401,"message":"Jwt is missing"}`,
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedReport{
		//			Version:         utils.ESPv2Version(),
		//			ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID: "test-config-id",
		//			URL:             "/v1/shelves?key=api-key",
		//			ApiMethod:       "endpoints.examples.bookstore.Bookstore.ListShelves",
		//			ApiName:         "endpoints.examples.bookstore.Bookstore",
		//			// API Key is not checked, only shows up in the log entry.
		//			ApiKeyInLogEntryOnly: "api-key",
		//			ApiKeyState:          "NOT CHECKED",
		//			ApiVersion:           "1.0.0",
		//			ProducerProjectID:    "producer project",
		//			FrontendProtocol:     "http",
		//			HttpMethod:           "GET",
		//			LogMessage:           "endpoints.examples.bookstore.Bookstore.ListShelves is called",
		//			StatusCode:           "0",
		//			ResponseCode:         401,
		//			Platform:             util.GCE,
		//			Location:             "test-zone",
		//			BackendProtocol:      "grpc",
		//			ResponseCodeDetail:   "jwt_authn_access_denied{Jwt_is_missing}",
		//		},
		//	},
		//},
		//{
		//	desc:           "For the request without api key but required to have, return \"Unauthorized\" status and send report",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	token:          testdata.Es256Token,
		//	clientProtocol: "http",
		//	httpMethod:     "GET",
		//	method:         "/v1/shelves/0/books/0",
		//	httpCallError:  `401 Unauthorized, {"code":401,"message":"UNAUTHENTICATED:Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API."}`,
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedReport{
		//			Version:            utils.ESPv2Version(),
		//			ServiceName:        "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID:    "test-config-id",
		//			URL:                "/v1/shelves/0/books/0",
		//			ApiMethod:          "endpoints.examples.bookstore.Bookstore.GetBook",
		//			ApiName:            "endpoints.examples.bookstore.Bookstore",
		//			ApiVersion:         "1.0.0",
		//			ApiKeyState:        "NOT CHECKED",
		//			ProducerProjectID:  "producer project",
		//			FrontendProtocol:   "http",
		//			HttpMethod:         "GET",
		//			LogMessage:         "endpoints.examples.bookstore.Bookstore.GetBook is called",
		//			StatusCode:         "16",
		//			ResponseCode:       401,
		//			ErrorCause:         "Method doesn't allow unregistered callers (callers without established identity). Please use API Key or other form of API consumer identity to call this API.",
		//			Platform:           util.GCE,
		//			Location:           "test-zone",
		//			BackendProtocol:    "grpc",
		//			ResponseCodeDetail: "service_control_bad_request{MISSING_API_KEY}",
		//		},
		//	},
		//},
		//{
		//	desc:           "gRPC backend returns canonical rpc status code",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	clientProtocol: "grpc",
		//	headers:        http.Header{"x-api-key": []string{"api-key"}},
		//	method:         "GetShelfInvalid",
		//	httpCallError:  `code = NotFound`,
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedCheck{
		//			Version:         utils.ESPv2Version(),
		//			ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID: "test-config-id",
		//			ConsumerID:      "api_key:api-key",
		//			OperationName:   "endpoints.examples.bookstore.Bookstore.GetShelf",
		//			CallerIp:        platform.GetLoopbackAddress(),
		//		},
		//		&utils.ExpectedReport{
		//			Version:                      utils.ESPv2Version(),
		//			ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID:              "test-config-id",
		//			URL:                          "/endpoints.examples.bookstore.Bookstore/GetShelf",
		//			ApiMethod:                    "endpoints.examples.bookstore.Bookstore.GetShelf",
		//			ApiName:                      "endpoints.examples.bookstore.Bookstore",
		//			ApiVersion:                   "1.0.0",
		//			ApiKeyInOperationAndLogEntry: "api-key",
		//			ApiKeyState:                  "VERIFIED",
		//			ProducerProjectID:            "producer project",
		//			ConsumerProjectID:            "123456",
		//			FrontendProtocol:             "grpc",
		//			HttpMethod:                   "POST",
		//			LogMessage:                   "endpoints.examples.bookstore.Bookstore.GetShelf is called",
		//			StatusCode:                   "0",
		//			// Final status code reflects converted gRPC status.
		//			ResponseCode:   404,
		//			HttpStatusCode: 200,
		//			GrpcStatusCode: "NotFound",
		//			Platform:       util.GCE,
		//			Location:       "test-zone",
		//		},
		//	},
		//},
		//{
		//	desc:           "gRPC backend returns NON-canonical rpc status code",
		//	url:            fmt.Sprintf("localhost:%v", s.Ports().ListenerPort),
		//	clientProtocol: "grpc",
		//	headers:        http.Header{"x-api-key": []string{"api-key"}},
		//	method:         "ReturnBadStatus",
		//	httpCallError:  `code = Code(74)`,
		//	wantScRequests: []interface{}{
		//		&utils.ExpectedCheck{
		//			Version:         utils.ESPv2Version(),
		//			ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID: "test-config-id",
		//			ConsumerID:      "api_key:api-key",
		//			OperationName:   "endpoints.examples.bookstore.Bookstore.ReturnBadStatus",
		//			CallerIp:        platform.GetLoopbackAddress(),
		//		},
		//		&utils.ExpectedReport{
		//			Version:                      utils.ESPv2Version(),
		//			ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
		//			ServiceConfigID:              "test-config-id",
		//			URL:                          "/endpoints.examples.bookstore.Bookstore/ReturnBadStatus",
		//			ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ReturnBadStatus",
		//			ApiName:                      "endpoints.examples.bookstore.Bookstore",
		//			ApiVersion:                   "1.0.0",
		//			ApiKeyInOperationAndLogEntry: "api-key",
		//			ApiKeyState:                  "VERIFIED",
		//			ProducerProjectID:            "producer project",
		//			ConsumerProjectID:            "123456",
		//			FrontendProtocol:             "grpc",
		//			HttpMethod:                   "POST",
		//			LogMessage:                   "endpoints.examples.bookstore.Bookstore.ReturnBadStatus is called",
		//			StatusCode:                   "0",
		//			// Final status code falls back to HTTP status because gRPC status code is non canonical.
		//			ResponseCode: 200,
		//			Platform:     util.GCE,
		//			Location:     "test-zone",
		//		},
		//	},
		//},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := client.MakeCall(tc.clientProtocol, tc.url, tc.httpMethod, tc.method, tc.token, tc.headers)
			if err == nil || !strings.Contains(err.Error(), tc.httpCallError) {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}

			scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err1 != nil {
				t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
			}
			utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
		})
	}
}
