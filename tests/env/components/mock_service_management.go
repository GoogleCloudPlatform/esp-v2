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
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// MockServiceMrg mocks the Service Management server.
// All requests must be ProtoOverHttp.
type MockServiceMrg struct {
	s                 *httptest.Server
	serviceName       string
	ServiceConfig     *confpb.Service
	rolloutId         string
	ConfigsHandler    http.Handler
	rolloutsHandler   http.Handler
	LastServiceConfig []byte
	serverCerts       *tls.Certificate
}

type configsHandler struct {
	m *MockServiceMrg
}

func (h *configsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceConfigByte, _ := proto.Marshal(h.m.ServiceConfig)
	h.m.LastServiceConfig = serviceConfigByte
	_, _ = w.Write(serviceConfigByte)
}

type rolloutsHandler struct {
	m *MockServiceMrg
}

func (h *rolloutsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := "/v1/services/" + h.m.serviceName + "/rollouts"
	if r.URL.Path != s {
		w.WriteHeader(http.StatusNotFound)
	}

	serviceConfigRollouts := &sm.ListServiceRolloutsResponse{
		Rollouts: []*sm.Rollout{
			{
				RolloutId: h.m.rolloutId,
				Strategy: &sm.Rollout_TrafficPercentStrategy_{
					TrafficPercentStrategy: &sm.Rollout_TrafficPercentStrategy{
						Percentages: map[string]float64{
							h.m.rolloutId: 1.0,
						},
					},
				},
			},
		},
	}
	serviceConfigRolloutsBytes, _ := proto.Marshal(serviceConfigRollouts)
	_, _ = w.Write(serviceConfigRolloutsBytes)
}

// NewMockServiceMrg creates a new HTTP server.
func NewMockServiceMrg(serviceName, rolloutId string, serviceConfig *confpb.Service) *MockServiceMrg {
	m := &MockServiceMrg{
		serviceName:   serviceName,
		ServiceConfig: serviceConfig,
		rolloutId:     rolloutId,
	}
	m.ConfigsHandler = &configsHandler{m: m}
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
	r.Path(configPath).Methods("GET").Handler(m.ConfigsHandler)
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
	serviceConfigByte, _ := proto.Marshal(m.ServiceConfig)
	_, _ = w.Write(serviceConfigByte)
}

func (m *MockServiceMrg) SetRolloutId(newRolloutId string) {
	m.rolloutId = newRolloutId
}
