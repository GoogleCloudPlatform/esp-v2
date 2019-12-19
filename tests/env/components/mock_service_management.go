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
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
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
	w.Write(serviceConfigByte)
}

type rolloutsHandler struct {
	m *MockServiceMrg
}

func (h *rolloutsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceConfigByte, _ := proto.Marshal(h.m.serviceConfig)
	if !bytes.Equal(serviceConfigByte, h.m.lastServiceConfig) {
		h.m.rolloutID += 1
	}
	serviceConfigRollout := &sm.ListServiceRolloutsResponse{
		Rollouts: []*sm.Rollout{
			{
				RolloutId: strconv.Itoa(h.m.rolloutID),
				Strategy: &sm.Rollout_TrafficPercentStrategy_{
					TrafficPercentStrategy: &sm.Rollout_TrafficPercentStrategy{
						Percentages: map[string]float64{
							strconv.Itoa(h.m.rolloutID): 1.0,
						},
					},
				},
			},
		},
	}
	serviceConfigRolloutBytes, _ := proto.Marshal(serviceConfigRollout)
	w.Write(serviceConfigRolloutBytes)
}

// NewMockServiceMrg creates a new HTTP server.
func NewMockServiceMrg(serviceName string, serviceConfig *confpb.Service) *MockServiceMrg {
	m := &MockServiceMrg{
		serviceName:   serviceName,
		serviceConfig: serviceConfig,
	}
	m.configsHandler = &configsHandler{m: m}
	m.rolloutsHandler = &rolloutsHandler{m: m}
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
	rolloutsPath := "/v1/services/" + m.serviceName + "/rollouts"
	r.Path(configPath).Methods("GET").Handler(m.configsHandler)
	r.Path(rolloutsPath).Methods("GET").Handler(m.rolloutsHandler).Queries("filter", "{filter}")
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
	w.Write(serviceConfigByte)
}
