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
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// MockServiceMrg mocks the Service Management server.
type MockServiceMrg struct {
	s                        *httptest.Server
	serviceName              string
	serviceConfig            *conf.Service
	rolloutID                int
	configsHandler           http.Handler
	rolloutsHandler          http.Handler
	lastServiceConfigJsonStr string
}

type configsHandler struct {
	m *MockServiceMrg
}

func (h *configsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	marshaller := &jsonpb.Marshaler{}
	serviceConfigJsonStr, _ := marshaller.MarshalToString(h.m.serviceConfig)
	h.m.lastServiceConfigJsonStr = serviceConfigJsonStr
	w.Write([]byte(serviceConfigJsonStr))
}

type rolloutsHandler struct {
	m *MockServiceMrg
}

func (h *rolloutsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	marshaller := &jsonpb.Marshaler{}
	serviceConfigJsonStr, _ := marshaller.MarshalToString(h.m.serviceConfig)
	if serviceConfigJsonStr != h.m.lastServiceConfigJsonStr {
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
	serviceConfigRolloutJsonStr, _ := marshaller.MarshalToString(serviceConfigRollout)
	w.Write([]byte(serviceConfigRolloutJsonStr))
}

// NewMockServiceMrg creates a new HTTP server.
func NewMockServiceMrg(serviceName string, serviceConfig *conf.Service) *MockServiceMrg {
	m := &MockServiceMrg{
		serviceName:   serviceName,
		serviceConfig: serviceConfig,
	}
	m.configsHandler = &configsHandler{m: m}
	m.rolloutsHandler = &rolloutsHandler{m: m}
	return m
}

// Start launches a mock ServiceManagement server.
func (m *MockServiceMrg) Start() (URL string) {
	r := mux.NewRouter()
	configPath := "/v1/services/" + m.serviceName + "/configs/{configID}"
	rolloutsPath := "/v1/services/" + m.serviceName + "/rollouts"
	r.Path(configPath).Methods("GET").Handler(m.configsHandler)
	r.Path(rolloutsPath).Methods("GET").Handler(m.rolloutsHandler).Queries("filter", "{filter}")
	m.s = httptest.NewServer(r)
	return m.s.URL
}

// ServeHTTP responds to requests with static service config message.
func (m *MockServiceMrg) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	marshaller := &jsonpb.Marshaler{}
	serviceConfigJsonStr, _ := marshaller.MarshalToString(m.serviceConfig)
	w.Write([]byte(serviceConfigJsonStr))
}
