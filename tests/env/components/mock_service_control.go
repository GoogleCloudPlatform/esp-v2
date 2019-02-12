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

type ServiceRequest struct {
	ReqType ServiceRequestType
	ReqBody []byte
}

type serviceResponse struct {
	req_type  ServiceRequestType
	resp_body []byte
}

// MockServiceMrg mocks the Service Management server.
type MockServiceCtrl struct {
	s              *httptest.Server
	ch             chan *ServiceRequest
	count          int
	check_resp     *serviceResponse
	report_resp    *serviceResponse
	check_handler  http.Handler
	report_handler http.Handler
}

type serviceHandler struct {
	m    *MockServiceCtrl
	resp *serviceResponse
}

func (h *serviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Mock service control handler: %v", h.resp.req_type)

	req := &ServiceRequest{
		ReqType: h.resp.req_type,
	}
	req.ReqBody, _ = ioutil.ReadAll(r.Body)

	h.m.ch <- req
	h.m.count++
	w.Write(h.resp.resp_body)
}

func setOKCheckResponse() []byte {
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
		ch: make(chan *ServiceRequest, 100),
	}

	m.check_resp = &serviceResponse{
		req_type:  CHECK_REQUEST,
		resp_body: setOKCheckResponse(),
	}
	m.check_handler = &serviceHandler{
		m:    m,
		resp: m.check_resp,
	}

	m.report_resp = &serviceResponse{
		req_type:  REPORT_REQUEST,
		resp_body: []byte(""),
	}
	m.report_handler = &serviceHandler{
		m:    m,
		resp: m.report_resp,
	}

	check_path := "/v1/services/" + service + ":check"
	report_path := "/v1/services/" + service + ":report"
	r := mux.NewRouter()
	r.Path(check_path).Methods("POST").Handler(m.check_handler)
	r.Path(report_path).Methods("POST").Handler(m.report_handler)

	glog.Infof("Start mock service control server for service: %s\n", service)
	m.s = httptest.NewServer(r)
	return m
}

func (m *MockServiceCtrl) GetURL() string {
	return m.s.URL
}

func (m *MockServiceCtrl) GetRequests(n int, timeout time.Duration) ([]*ServiceRequest, error) {
	r := make([]*ServiceRequest, n)
	for i := 0; i < n; i++ {
		select {
		case d := <-m.ch:
			r[i] = d
		case <-time.After(timeout):
			return nil, fmt.Errorf("Timeout got %d, expected: %d", i, n)
		}
	}
	return r, nil
}
