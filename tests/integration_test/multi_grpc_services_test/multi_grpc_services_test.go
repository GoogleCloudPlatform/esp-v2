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

package multi_grpc_services_test

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

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestMultiGrpcServices(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID, "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestMultiGrpcServices, platform.GrpcBookstoreSidecar)
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	tests := []struct {
		desc           string
		clientProtocol string
		service        string
		method         string
		header         http.Header
		wantResp       string
		wantError      string
		wantScRequests []interface{}
	}{
		{
			desc:           "gRPC client calling bookstore/v1",
			clientProtocol: "grpc",
			method:         "GetShelf",
			header:         http.Header{"x-api-key": []string{"api-key-1"}},
			wantResp:       `{"id":"100","theme":"Kids"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-1",
					OperationName:   "endpoints.examples.bookstore.Bookstore.GetShelf",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/endpoints.examples.bookstore.Bookstore/GetShelf",
					ApiKey:            "api-key-1",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.GetShelf",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "grpc",
					HttpMethod:        "POST",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.GetShelf is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "Http client calling bookstore/v1",
			clientProtocol: "http",
			method:         "/v1/shelves/100?key=api-key-2",
			wantResp:       `{"id":"100","theme":"Kids"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-2",
					OperationName:   "endpoints.examples.bookstore.Bookstore.GetShelf",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves/100?key=api-key-2",
					ApiKey:            "api-key-2",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.GetShelf",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.GetShelf is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "gRPC client calling bookstore/v2",
			clientProtocol: "grpc",
			service:        "BookstoreV2",
			method:         "GetShelf",
			header:         http.Header{"x-api-key": []string{"api-key-1"}},
			wantResp:       `{"id":"100","theme":"Kids"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-1",
					OperationName:   "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/endpoints.examples.bookstore.v2.Bookstore/GetShelf",
					ApiKey:            "api-key-1",
					ApiMethod:         "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "grpc",
					HttpMethod:        "POST",
					LogMessage:        "endpoints.examples.bookstore.v2.Bookstore.GetShelf is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "Http client calling bookstore/v2",
			clientProtocol: "http",
			service:        "BookstoreV2",
			method:         "/v2/shelves/100?key=api-key-2",
			wantResp:       `{"id":"100","theme":"Kids"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key-2",
					OperationName:   "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:           utils.ESPv2Version(),
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v2/shelves/100?key=api-key-2",
					ApiKey:            "api-key-2",
					ApiMethod:         "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.v2.Bookstore.GetShelf is called",
					StatusCode:        "0",
					ResponseCode:      200,
					RequestMsgCounts:  1,
					ResponseMsgCounts: 1,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}
	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		var resp string
		var err error
		if tc.service == "BookstoreV2" && tc.clientProtocol == "grpc" {
			resp, err = client.MakeBookstoreV2GrpcCall(addr, tc.method, tc.header)
		} else {
			resp, err = client.MakeCall(tc.clientProtocol, addr, "GET", tc.method, "", tc.header)
		}
		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected: %s, got: %v", tc.desc, tc.wantError, err)
		}

		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}
		if len(tc.wantScRequests) != 0 {
			scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
			if err1 != nil {
				t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
			}
			utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
		}

	}
}
