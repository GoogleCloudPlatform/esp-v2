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

package service_control_quota_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsClient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func TestServiceControlQuota(t *testing.T) {
	t.Parallel()

	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(platform.TestServiceControlQuota, platform.GrpcBookstoreSidecar)
	s.OverrideQuota(&confpb.Quota{
		MetricRules: []*confpb.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                string
		clientProtocol      string
		method              string
		httpMethod          string
		token               string
		requestHeader       map[string]string
		message             string
		mockedCheckResponse *scpb.CheckResponse
		wantResp            string
		wantError           string
		wantScRequests      []interface{}
	}{
		{
			desc:           "succeed, quota allocation works well",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "api_key:api-key",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       scpb.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ApiVersion:                   "1.0.0",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:           "Quota not called when Check fails with invalid API Key",
			clientProtocol: "http",
			method:         "/v1/shelves?key=invalid-api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			mockedCheckResponse: &scpb.CheckResponse{
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_API_KEY_INVALID,
					},
				},
			},
			wantError: `400 Bad Request, {"code":400,"message":"INVALID_ARGUMENT:API key not valid. Please pass a valid API key."}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:invalid-api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedReport{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					URL:             "/v1/shelves?key=invalid-api-key",
					// API Key is invalid, so only in log entry.
					ApiKeyInLogEntryOnly: "invalid-api-key",
					ApiKeyState:          "INVALID",
					// API Key is invalid, so JWT is used as credential_id instead.
					JwtAuthCredentialId: "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw",
					ApiMethod:           "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:             "endpoints.examples.bookstore.Bookstore",
					ApiVersion:          "1.0.0",
					ErrorCause:          "API key not valid. Please pass a valid API key.",
					ProducerProjectID:   "producer project",
					FrontendProtocol:    "http",
					BackendProtocol:     "grpc",
					HttpMethod:          "GET",
					LogMessage:          "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:          "3",
					ResponseCode:        400,
					Platform:            util.GCE,
					Location:            "test-zone",
					ResponseCodeDetail:  "service_control_check_error{API_KEY_INVALID}",
				},
			},
		},
	}

	for _, tc := range testData {
		if tc.mockedCheckResponse != nil {
			s.ServiceControlServer.SetCheckResponse(tc.mockedCheckResponse)
		}

		addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed\nexpected: %v\ngot: %v", tc.desc, tc.wantError, err)
		} else if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed\nexpected: %s\ngot: %s", tc.desc, tc.wantResp, string(resp))
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}

type unavailableQuotaServiceHandler struct {
	m *comp.MockServiceCtrl
}

func (h *unavailableQuotaServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &utils.ServiceRequest{
		ReqType: utils.QuotaRequest,
	}
	req.ReqBody, _ = ioutil.ReadAll(r.Body)
	h.m.CacheRequest(req)
	h.m.IncrementRequestCount()

	w.WriteHeader(404)
}

func TestServiceControlQuotaFailOpen(t *testing.T) {
	t.Parallel()

	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(platform.TestServiceControlQuotaUnavailable, platform.GrpcBookstoreSidecar)
	s.OverrideQuota(&confpb.Quota{
		MetricRules: []*confpb.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	s.ServiceControlServer.OverrideQuotaHandler(&unavailableQuotaServiceHandler{m: s.ServiceControlServer})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc                  string
		clientProtocol        string
		method                string
		httpMethod            string
		token                 string
		requestHeader         map[string]string
		message               string
		wantResp              string
		wantScRequestCount    int
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:           "first request is granted with 3 service calls: check, quota and report",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "api_key:api-key",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       scpb.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiVersion:                   "1.0.0",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			// TODO(b/194517193): Consider fail close for client-side error (4xx errors)
			// quota server return 404, with fail-open policy, cached quota result is positive.
			// check use cache,  use cached quota, but aggregated quota is flushed out before report
			desc:           "second call, request is granted with 2 service control: quota, report",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "api_key:api-key",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       scpb.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiVersion:                   "1.0.0",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			// the third call should be the same as the second one.
			desc:           "third call, request is granted with 2 service control: quota, report",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "api_key:api-key",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       scpb.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiVersion:                   "1.0.0",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}

func TestServiceControlQuotaExhausted(t *testing.T) {
	t.Parallel()

	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(platform.TestServiceControlQuotaExhausted, platform.GrpcBookstoreSidecar)
	s.OverrideQuota(&confpb.Quota{
		MetricRules: []*confpb.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	s.ServiceControlServer.SetQuotaResponse(
		&scpb.AllocateQuotaResponse{
			AllocateErrors: []*scpb.QuotaError{
				{
					Code:    scpb.QuotaError_RESOURCE_EXHAUSTED,
					Subject: "Insufficient tokens for quota group and limit 'apiWriteQpsPerProject_LOW' of service 'test.appspot.com', using the limit by ID 'container:123123'.",
				},
			},
		})
	testData := []struct {
		desc                  string
		clientProtocol        string
		method                string
		httpMethod            string
		token                 string
		requestHeader         map[string]string
		message               string
		wantResp              string
		httpCallError         string
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:           "succeed, the first request of failed quota allocation is replied with success",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.ESPv2Version(),
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        platform.GetLoopbackAddress(),
				},
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "api_key:api-key",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       scpb.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiVersion:                   "1.0.0",
					ApiKeyState:                  "VERIFIED",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "0",
					ResponseCode:                 200,
					Platform:                     util.GCE,
					Location:                     "test-zone",
				},
			},
		},
		{
			desc:           "failed, the requests after failed qutoa allocation request will be denied",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			httpCallError:  `429 Too Many Requests, {"code":429,"message":"RESOURCE_EXHAUSTED"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:                      utils.ESPv2Version(),
					ServiceName:                  "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:              "test-config-id",
					URL:                          "/v1/shelves?key=api-key",
					ApiKeyInOperationAndLogEntry: "api-key",
					ApiKeyState:                  "VERIFIED",
					ApiVersion:                   "1.0.0",
					ApiName:                      "endpoints.examples.bookstore.Bookstore",
					ApiMethod:                    "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID:            "producer project",
					ConsumerProjectID:            "123456",
					FrontendProtocol:             "http",
					BackendProtocol:              "grpc",
					HttpMethod:                   "GET",
					LogMessage:                   "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:                   "8",
					ResponseCode:                 429,
					Platform:                     util.GCE,
					Location:                     "test-zone",
					ResponseCodeDetail:           "service_control_quota_error{RESOURCE_EXHAUSTED}",
				},
			},
		},
	}
	for _, tc := range testData {
		addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if tc.httpCallError == "" {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if err == nil || !strings.Contains(err.Error(), tc.httpCallError) {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		// If cached quota response is negative, it send CHECK_ONLY quota call every second.
		// Such Check_only calls should be removed when verifying ScRequests.
		// Otherwise number of quota calls depends on the time, not easy to verify scRequests.
		scRequests, err1 := s.ServiceControlServer.GetRequestsWithoutCheckOnlyQuota(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequestsWithoutCheckOnlyQuota returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
