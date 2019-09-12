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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	bsClient "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sc "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func TestServiceControlQuota(t *testing.T) {
	t.Parallel()
	configId := "test-config-id"

	args := []string{"--service_config_id=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlQuota, "bookstore")
	s.OverrideQuota(&conf.Quota{
		MetricRules: []*conf.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc           string
		clientProtocol string
		method         string
		httpMethod     string
		token          string
		requestHeader  map[string]string
		message        string
		wantResp       string
		wantScRequests []interface{}
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
					Version:         utils.APIProxyVersion,
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        "127.0.0.1",
				},
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "project:producer project",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       sc.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					RequestSize:       430,
					ResponseSize:      291,
					RequestBytes:      430,
					ResponseBytes:     291,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
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

type unavailableQuotaServiceHandler struct {
	m *comp.MockServiceCtrl
}

func (h *unavailableQuotaServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &comp.ServiceRequest{
		ReqType: comp.QUOTA_REQUEST,
	}
	req.ReqBody, _ = ioutil.ReadAll(r.Body)
	h.m.CacheRequest(req)
	h.m.IncrementRequestCount()

	w.WriteHeader(404)
}

func TestServiceControlQuotaUnavailable(t *testing.T) {
	t.Parallel()
	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlQuotaUnavailable, "bookstore")
	s.OverrideQuota(&conf.Quota{
		MetricRules: []*conf.MetricRule{
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
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	type testType struct {
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
	}

	tc := testType{
		desc:               "succeed, when the service control quota api is unavailable, the request still passes and works well",
		clientProtocol:     "http",
		method:             "/v1/shelves?key=api-key",
		token:              testdata.FakeCloudTokenMultiAudiences,
		httpMethod:         "GET",
		wantResp:           `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		wantScRequestCount: 3,
	}

	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if err != nil {
			t.Fatalf("Test (%s): failed, %v", tc.desc, err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
		}

		err = s.ServiceControlServer.VerifyRequestCount(tc.wantScRequestCount)
		if err != nil {
			t.Fatalf("Test (%s): failed, %s", tc.desc, err.Error())
		}
	}
}

func TestServiceControlQuotaExhausted(t *testing.T) {
	t.Parallel()
	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--service_config_id=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlQuotaExhausted, "bookstore")
	s.OverrideQuota(&conf.Quota{
		MetricRules: []*conf.MetricRule{
			{
				Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
				MetricCosts: map[string]int64{
					"metrics_first":  2,
					"metrics_second": 1,
				},
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	s.ServiceControlServer.SetQuotaResponse(
		&sc.AllocateQuotaResponse{
			AllocateErrors: []*sc.QuotaError{
				{
					Code:    sc.QuotaError_RESOURCE_EXHAUSTED,
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
		httpCallError         error
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
					Version:         utils.APIProxyVersion,
					ServiceName:     "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "endpoints.examples.bookstore.Bookstore.ListShelves",
					CallerIp:        "127.0.0.1",
				},

				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "project:producer project",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       sc.QuotaOperation_BEST_EFFORT,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					RequestSize:       430,
					ResponseSize:      291,
					RequestBytes:      430,
					ResponseBytes:     291,
					// It always allow the first request, then cache its cost, accumulate all costs for 1 second,
					// then call remote allocateQuota,  if fail, the next request will be failed with 429.
					// Here is the first request.
					ResponseCode: 200,
					Platform:     util.GCE,
					Location:     "test-zone",
				},
			},
		},
		{
			desc:           "failed, the requests after failed qutoa allocation request will be denied",
			clientProtocol: "http",
			method:         "/v1/shelves?key=api-key",
			token:          testdata.FakeCloudTokenMultiAudiences,
			httpMethod:     "GET",
			httpCallError:  fmt.Errorf("429 Too Many Requests, RESOURCE_EXHAUSTED"),
			wantScRequests: []interface{}{
				&utils.ExpectedQuota{
					ServiceName: "bookstore.endpoints.cloudesf-testing.cloud.goog",
					MethodName:  "endpoints.examples.bookstore.Bookstore.ListShelves",
					ConsumerID:  "project:producer project",
					QuotaMetrics: map[string]int64{
						"metrics_first":  2,
						"metrics_second": 1,
					},
					QuotaMode:       sc.QuotaOperation_NORMAL,
					ServiceConfigID: "test-config-id",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					ErrorType:         "4xx",
					StatusCode:        "8",
					RequestSize:       430,
					ResponseSize:      110,
					RequestBytes:      430,
					ResponseBytes:     110,
					// It always allow the first request, then cache its cost, accumulate all costs for 1 second,
					// then call remote allocateQuota,  if fail, the next request will be failed with 429.
					// Here is the second request.
					ResponseCode: 429,
					Platform:     util.GCE,
					Location:     "test-zone",
				},
			},
		},
	}
	for _, tc := range testData {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if !strings.Contains(err.Error(), tc.httpCallError.Error()) {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}
