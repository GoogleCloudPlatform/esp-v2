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
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/gorilla/mux"
)

// MockJwtProvider mocks the Jwt provider.
type MockJwtProvider struct {
	s   *httptest.Server
	cnt *int32
}

// JwtProviders is used to refer all created provider object with issuer
var JwtProviders = make(map[string]*MockJwtProvider)

// NewMockJwtProvider creates a new Jwt provider.
func NewMockJwtProvider(issuer, jwks string) *MockJwtProvider {
	mockJwtProvider := &MockJwtProvider{
		cnt: new(int32),
	}
	mockJwtProvider.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(mockJwtProvider.cnt, 1)
		w.Write([]byte(jwks))
	}))
	JwtProviders[issuer] = mockJwtProvider
	return mockJwtProvider
}

// NewMockInvalidJwtProvider creates a new Jwt provider which returns error.
func NewMockInvalidJwtProvider(issuer string) *MockJwtProvider {
	mockJwtProvider := &MockJwtProvider{
		cnt: new(int32),
	}
	mockJwtProvider.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(mockJwtProvider.cnt, 1)
		http.Error(w, `{"code": 503, "message": "service not found"}`, 503)
	}))
	JwtProviders[issuer] = mockJwtProvider
	return mockJwtProvider
}

// NewOpenIDServer creates a new Jwt provider with fixed address.
func NewOpenIDServer(addr, jwksUriEntry string) (*MockJwtProvider, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Fail to create OpenIDServer %v", err)
	}
	r := mux.NewRouter()
	r.Path("/.well-known/openid-configuration/").Methods("GET").Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(jwksUriEntry))
		}))
	mockJwtProvider := &MockJwtProvider{
		s: httptest.NewUnstartedServer(r),
	}
	mockJwtProvider.s.Listener.Close()
	mockJwtProvider.s.Listener = l
	mockJwtProvider.s.Start()
	return mockJwtProvider, nil
}

func (m *MockJwtProvider) GetURL() string {
	return m.s.URL
}

func (m *MockJwtProvider) GetReqCnt() int {
	return int(atomic.LoadInt32(m.cnt))
}

func ResetReqCnt() {
	for _, pd := range JwtProviders {
		atomic.SwapInt32(pd.cnt, 0)
	}
}
