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

// MockJwtProvider mocks the Jwt provider.
type MockJwtProvider struct {
	s *httptest.Server
}

// NewMockJwtProvider creates a new Jwt provider.
func NewMockJwtProvider(jwks string) *MockJwtProvider {
	return &MockJwtProvider{
		s: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(jwks))
		}))}
}

func (m *MockJwtProvider) GetURL() string {
	return m.s.URL
}
