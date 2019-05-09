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
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/golang/protobuf/proto"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
)

type ServiceRequestType int

const (
	CHECK_REQUEST = 1 + iota
	REPORT_REQUEST
)

const defaultTimeout = 2500 * time.Millisecond

type ServiceRequest struct {
	ReqType ServiceRequestType
	ReqBody []byte
}

type serviceResponse struct {
	reqType  ServiceRequestType
	respBody []byte
}

// MockServiceMrg mocks the Service Management server.
type MockServiceCtrl struct {
	s                  *httptest.Server
	ch                 chan *ServiceRequest
	count              int
	checkResp          *serviceResponse
	reportResp         *serviceResponse
	checkHandler       http.Handler
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
	req.ReqBody, _ = ioutil.ReadAll(r.Body)

	h.m.ch <- req
	h.m.count++
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
func NewMockServiceCtrl(service string) *MockServiceCtrl {
	m := &MockServiceCtrl{
		ch:                 make(chan *ServiceRequest, 100),
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

	m.reportResp = &serviceResponse{
		reqType:  REPORT_REQUEST,
		respBody: []byte(""),
	}
	m.reportHandler = &serviceHandler{
		m:    m,
		resp: m.reportResp,
	}

	check_path := "/v1/services/" + service + ":check"
	report_path := "/v1/services/" + service + ":report"
	r := mux.NewRouter()
	r.Path(check_path).Methods("POST").Handler(m.checkHandler)
	r.Path(report_path).Methods("POST").Handler(m.reportHandler)

	glog.Infof("Start mock service control server for service: %s\n", service)
	m.s = httptest.NewServer(r)
	return m
}

// GetURL returns the URL of MockServiceCtrl.
func (m *MockServiceCtrl) GetURL() string {
	return m.s.URL
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
