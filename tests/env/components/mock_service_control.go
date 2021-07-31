// Copyright 2019 Google LLC
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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

const defaultTimeout = 2500 * time.Millisecond

type serviceResponse struct {
	reqType        utils.ServiceRequestType
	respBody       []byte
	respStatusCode int
}

// MockServiceMrg mocks the Service Management server.
type MockServiceCtrl struct {
	s                  *httptest.Server
	ch                 chan *utils.ServiceRequest
	serverCerts        *tls.Certificate
	url                string
	count              *int32
	serviceName        string
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
	req := &utils.ServiceRequest{
		ReqType:   h.resp.reqType,
		ReqHeader: r.Header,
	}
	atomic.AddInt32(h.m.count, 1)
	req.ReqBody, _ = ioutil.ReadAll(r.Body)
	h.m.ch <- req

	if h.resp.respStatusCode != 0 {
		w.WriteHeader(h.resp.respStatusCode)
		return
	}
	_, _ = w.Write(h.resp.respBody)
}

func setOKCheckResponse() []byte {
	req := &scpb.CheckResponse{
		CheckInfo: &scpb.CheckResponse_CheckInfo{
			ConsumerInfo: &scpb.CheckResponse_ConsumerInfo{
				ProjectNumber:  123456,
				ConsumerNumber: 123456,
				Type:           scpb.CheckResponse_ConsumerInfo_PROJECT,
			},
		},
	}
	req_b, _ := proto.Marshal(req)
	return req_b
}

func setReportResponse(serviceRolloutId string) []byte {
	req := &scpb.ReportResponse{
		ServiceRolloutId: serviceRolloutId,
	}
	req_b, _ := proto.Marshal(req)
	return req_b
}

// NewMockServiceCtrl creates a new HTTP server.
func NewMockServiceCtrl(serviceName, rolloutId string) *MockServiceCtrl {
	m := &MockServiceCtrl{
		ch:                 make(chan *utils.ServiceRequest, 100),
		count:              new(int32),
		serviceName:        serviceName,
		getRequestsTimeout: defaultTimeout,
	}

	m.checkHandler = &serviceHandler{
		m: m,
		resp: &serviceResponse{
			reqType:  utils.CheckRequest,
			respBody: setOKCheckResponse(),
		},
	}

	m.quotaHandler = &serviceHandler{
		m: m,
		resp: &serviceResponse{
			reqType:  utils.QuotaRequest,
			respBody: []byte(""),
		},
	}

	m.reportHandler = &serviceHandler{
		m: m,
		resp: &serviceResponse{
			reqType:  utils.ReportRequest,
			respBody: setReportResponse(rolloutId),
		},
	}

	return m
}

// SetCert sets the server cert for ServiceControl server, so it acts as a HTTPS server
func (m *MockServiceCtrl) SetCert(serverCerts *tls.Certificate) {
	m.serverCerts = serverCerts
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
	m.s = httptest.NewUnstartedServer(r)

	if m.serverCerts != nil {
		m.s.TLS = &tls.Config{
			Certificates: []tls.Certificate{*m.serverCerts},
			// NextProtos:   []string{"h2"},
		}
		m.s.StartTLS()
	} else {
		m.s.Start()
	}
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

func (m *MockServiceCtrl) CacheRequest(req *utils.ServiceRequest) {
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
func (m *MockServiceCtrl) SetCheckResponse(checkResponse *scpb.CheckResponse) {
	req_b, _ := proto.Marshal(checkResponse)
	(m.checkHandler).(*serviceHandler).resp.respBody = req_b
}

// SetCheckResponseStatus sets the response status code for the check of the service control.
func (m *MockServiceCtrl) SetCheckResponseStatus(status int) {
	(m.checkHandler).(*serviceHandler).resp.respStatusCode = status
}

// SetQuotaResponseStatus sets the response status code for the quota of the service control.
func (m *MockServiceCtrl) SetQuotaResponseStatus(status int) {
	(m.quotaHandler).(*serviceHandler).resp.respStatusCode = status
}

// SetCheckResponse sets the response for the check of the service control.
func (m *MockServiceCtrl) SetQuotaResponse(quotaResponse *scpb.AllocateQuotaResponse) {
	req_b, _ := proto.Marshal(quotaResponse)
	(m.quotaHandler).(*serviceHandler).resp.respBody = req_b
}

// SetReportResponseStatus sets the status of the report response of the service control.
func (m *MockServiceCtrl) SetReportResponseStatus(statusCode int) {
	(m.reportHandler).(*serviceHandler).resp.respStatusCode = statusCode
}

// SetReportResponseStatus sets the status of the report response of the service control.
func (m *MockServiceCtrl) SetRolloutIdConfigIdInReport(newRolloutId string) {
	(m.reportHandler).(*serviceHandler).resp.respBody = setReportResponse(newRolloutId)
}

// GetRequests returns a slice of requests received.
func (m *MockServiceCtrl) GetRequests(n int) ([]*utils.ServiceRequest, error) {
	r := make([]*utils.ServiceRequest, n)
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

func isCheckOnlyQuota(in *utils.ServiceRequest) bool {
	if in.ReqType != utils.QuotaRequest {
		return false
	}
	got, err := utils.UnmarshalQuotaRequest(in.ReqBody)
	// If not QuotaRequest, not to ignore
	if err != nil {
		return false
	}
	return got.AllocateOperation.QuotaMode == scpb.QuotaOperation_CHECK_ONLY
}

// GetRequestsWithoutCheckOnlyQuota returns a slice of requests received.
func (m *MockServiceCtrl) GetRequestsWithoutCheckOnlyQuota(n int) ([]*utils.ServiceRequest, error) {
	r := make([]*utils.ServiceRequest, n)
	for i := 0; i < n; {
		select {
		case d := <-m.ch:
			if !isCheckOnlyQuota(d) {
				r[i] = d
				i++
			}
		case <-time.After(m.getRequestsTimeout):
			return nil, fmt.Errorf("Timeout got %d, expected: %d", i, n)
		}
	}
	return r, nil
}

// GetRequests returns a slice of requests received.
func (m *MockServiceCtrl) GetAllRequests() []*utils.ServiceRequest {
	r := []*utils.ServiceRequest{}
	for {
		select {
		case d := <-m.ch:
			r = append(r, d)
		case <-time.After(m.getRequestsTimeout):
			return r
		}
	}

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
