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

package components

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	sc "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

type ServiceRequestType int

const (
	CHECK_REQUEST = 1 + iota
	QUOTA_REQUEST
	REPORT_REQUEST
)

const defaultTimeout = 2500 * time.Millisecond

type ServiceRequest struct {
	ReqType ServiceRequestType
	ReqBody []byte
}

type serviceResponse struct {
	reqType        ServiceRequestType
	respBody       []byte
	respStatusCode int
}

// MockServiceMrg mocks the Service Management server.
type MockServiceCtrl struct {
	s                  *httptest.Server
	ch                 chan *ServiceRequest
	url                string
	count              *int32
	serviceName        string
	checkResp          *serviceResponse
	quotaResp          *serviceResponse
	reportResp         *serviceResponse
	checkHandler       http.Handler
	quotaHandler       http.Handler
	reportHandler      http.Handler
	getRequestsTimeout time.Duration
}

type serviceHandler struct {
	m    *MockServiceCtrl
	resp *serviceResponse
}

func (h *serviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Mock service control handler: %v", h.resp.reqType)

	req := &ServiceRequest{
		ReqType: h.resp.reqType,
	}
	atomic.AddInt32(h.m.count, 1)
	req.ReqBody, _ = ioutil.ReadAll(r.Body)
	h.m.ch <- req

	if h.resp.respStatusCode != 0 {
		w.WriteHeader(h.resp.respStatusCode)
	}
	w.Write(h.resp.respBody)
}

func SetOKCheckResponse() []byte {
	req := &sc.CheckResponse{
		CheckInfo: &sc.CheckResponse_CheckInfo{
			ConsumerInfo: &sc.CheckResponse_ConsumerInfo{
				ProjectNumber: 123456,
			},
		},
	}
	req_b, _ := proto.Marshal(req)
	return req_b
}

// NewMockServiceCtrl creates a new HTTP server.
func NewMockServiceCtrl(serviceName string) *MockServiceCtrl {
	m := &MockServiceCtrl{
		ch:                 make(chan *ServiceRequest, 100),
		count:              new(int32),
		serviceName:        serviceName,
		getRequestsTimeout: defaultTimeout,
	}

	m.checkResp = &serviceResponse{
		reqType:  CHECK_REQUEST,
		respBody: SetOKCheckResponse(),
	}
	m.checkHandler = &serviceHandler{
		m:    m,
		resp: m.checkResp,
	}

	m.quotaResp = &serviceResponse{
		reqType:  QUOTA_REQUEST,
		respBody: []byte(""),
	}
	m.quotaHandler = &serviceHandler{
		m:    m,
		resp: m.quotaResp,
	}

	m.reportResp = &serviceResponse{
		reqType:  REPORT_REQUEST,
		respBody: []byte(""),
	}
	m.reportHandler = &serviceHandler{
		m:    m,
		resp: m.reportResp,
	}

	return m
}

func (m *MockServiceCtrl) Setup() {
	r := mux.NewRouter()
	checkPath := "/v1/services/" + m.serviceName + ":check"
	quotaPath := "/v1/services/" + m.serviceName + ":allocateQuota"
	reportPath := "/v1/services/" + m.serviceName + ":report"

	r.Path(checkPath).Methods("POST").Handler(m.checkHandler)
	r.Path(quotaPath).Methods("POST").Handler(m.quotaHandler)
	r.Path(reportPath).Methods("POST").Handler(m.reportHandler)

	glog.Infof("Start mock service control server for service: %s\n", m.serviceName)
	m.s = httptest.NewServer(r)
}

// OverrideCheckHandler overrides the service control check handler before setup.
func (m *MockServiceCtrl) OverrideCheckHandler(checkHandler http.Handler) {
	m.checkHandler = checkHandler
}

// OverrideQuoatHandler overrides the service control quota handler before setup.
func (m *MockServiceCtrl) OverrideQuotaHandler(quotaHandler http.Handler) {
	m.quotaHandler = quotaHandler
}

// OverrideReportHandler overrides the service control report handler before setup.
func (m *MockServiceCtrl) OverrideReportHandler(reportHandler http.Handler) {
	m.reportHandler = reportHandler
}

// GetURL returns the URL of MockServiceCtrl.
func (m *MockServiceCtrl) GetURL() string {
	if m.url != "" {
		return m.url
	}
	return m.s.URL
}

// GetURL returns the URL of MockServiceCtrl.
func (m *MockServiceCtrl) SetURL(url string) {
	m.url = url
}

func (m *MockServiceCtrl) GetRequestCount() int {
	return int(atomic.LoadInt32(m.count))
}

func (m *MockServiceCtrl) CacheRequest(req *ServiceRequest) {
	m.ch <- req
}

// ResetRequestCount resets the request count of MockServiceCtrl.
func (m *MockServiceCtrl) ResetRequestCount() {
	atomic.StoreInt32(m.count, 0)
}

// IncrementRequestCount increments the request count of MockServiceCtrl.
func (m *MockServiceCtrl) IncrementRequestCount() {
	atomic.AddInt32(m.count, 1)
}

// SetGetRequestsTimeout sets the timeout for GetRequests.
func (m *MockServiceCtrl) SetGetRequestsTimeout(timeout time.Duration) {
	m.getRequestsTimeout = timeout
}

// SetCheckResponse sets the response for the check of the service control.
func (m *MockServiceCtrl) SetCheckResponse(checkResponse *sc.CheckResponse) {
	req_b, _ := proto.Marshal(checkResponse)
	m.checkResp.respBody = req_b
}

// SetCheckResponse sets the response for the check of the service control.
func (m *MockServiceCtrl) SetQuotaResponse(quotaResponse *sc.AllocateQuotaResponse) {
	fmt.Println(quotaResponse)
	req_b, _ := proto.Marshal(quotaResponse)
	m.quotaResp.respBody = req_b
}

// SetReportResponseStatus sets the status of the report response of the service control.
func (m *MockServiceCtrl) SetReportResponseStatus(statusCode int) {
	m.reportResp.respStatusCode = statusCode
}

// GetRequests returns a slice of requests received.
func (m *MockServiceCtrl) GetRequests(n int) ([]*ServiceRequest, error) {
	r := make([]*ServiceRequest, n)
	for i := 0; i < n; i++ {
		select {
		case d := <-m.ch:
			r[i] = d
		case <-time.After(m.getRequestsTimeout):
			return nil, fmt.Errorf("Timeout got %d, expected: %d", i, n)
		}
	}
	return r, nil
}

// VerifyRequestCount Verifies the current exact request count with the want request count
func (m *MockServiceCtrl) VerifyRequestCount(wantRequestCount int) error {
	_, err := m.GetRequests(wantRequestCount)
	if err != nil {
		return fmt.Errorf("expected service count request count: %v, got %v", wantRequestCount, m.GetRequestCount())
	}
	_, err = m.GetRequests(1)
	if err == nil {
		return fmt.Errorf("expected service count request count: %v, got %v", wantRequestCount, m.GetRequestCount())
	}
	return nil
}
