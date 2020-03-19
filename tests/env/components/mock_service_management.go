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
	"net/http"
	"net/http/httptest"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// MockServiceMrg mocks the Service Management server.
// All requests must be ProtoOverHttp.
type MockServiceMrg struct {
	s                 *httptest.Server
	serviceName       string
	serviceConfig     *confpb.Service
	rolloutID         int
	configsHandler    http.Handler
	rolloutsHandler   http.Handler
	lastServiceConfig []byte
	serverCerts       *tls.Certificate
}

type configsHandler struct {
	m *MockServiceMrg
}

func (h *configsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceConfigByte, _ := proto.Marshal(h.m.serviceConfig)
	h.m.lastServiceConfig = serviceConfigByte
	_, _ = w.Write(serviceConfigByte)
}

// NewMockServiceMrg creates a new HTTP server.
func NewMockServiceMrg(serviceName string, serviceConfig *confpb.Service) *MockServiceMrg {
	m := &MockServiceMrg{
		serviceName:   serviceName,
		serviceConfig: serviceConfig,
	}
	m.configsHandler = &configsHandler{m: m}
	return m
}

// SetCert sets the server cert for ServiceMrg server, so it acts as a HTTPS server
func (m *MockServiceMrg) SetCert(serverCerts *tls.Certificate) {
	m.serverCerts = serverCerts
}

// Start launches a mock ServiceManagement server.
func (m *MockServiceMrg) Start() (URL string) {
	r := mux.NewRouter()
	configPath := "/v1/services/" + m.serviceName + "/configs/{configID}"
	r.Path(configPath).Methods("GET").Handler(m.configsHandler)
	m.s = httptest.NewUnstartedServer(r)

	if m.serverCerts != nil {
		m.s.TLS = &tls.Config{
			Certificates: []tls.Certificate{*m.serverCerts},
			NextProtos:   []string{"h2"},
		}
		m.s.StartTLS()
	} else {
		m.s.Start()
	}
	return m.s.URL
}

// ServeHTTP responds to requests with static service config message.
func (m *MockServiceMrg) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceConfigByte, _ := proto.Marshal(m.serviceConfig)
	_, _ = w.Write(serviceConfigByte)
}
