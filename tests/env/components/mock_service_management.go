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
)

// MockServiceMrg mocks the Service Management server.
type MockServiceMrg struct {
	s             *httptest.Server
	serviceConfig string
}

// NewMockServiceMrg creates a new HTTP server.
func NewMockServiceMrg() *MockServiceMrg {
	m := &MockServiceMrg{}
	m.s = httptest.NewUnstartedServer(m)
	return m
}

// Start launches a mock ServiceManagement server.
func (m *MockServiceMrg) Start(serviceConfig string) (URL string) {
	m.serviceConfig = serviceConfig
	m.s.Start()
	return m.s.URL
}

// ServeHTTP responds to requests with static service config message.
func (m *MockServiceMrg) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(m.serviceConfig))
}
