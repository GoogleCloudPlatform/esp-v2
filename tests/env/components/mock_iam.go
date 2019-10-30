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
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
)

// MockIamServer mocks the Metadata server.
type MockIamServer struct {
	s        *httptest.Server
	ch       chan string
	reqCache map[string]int
	mtx      sync.RWMutex
}

// NewMockMetadata creates a new HTTP server.
func NewIamMetadata(pathResp map[string]string) *MockIamServer {
	mockPathResp := make(map[string]string)
	for k, v := range pathResp {
		mockPathResp[k] = v
	}

	m := &MockIamServer{
		ch:       make(chan string, 100),
		reqCache: make(map[string]int),
	}
	m.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reqURI := r.URL.RequestURI()
		m.mtx.Lock()
		reqCnt, _ := m.reqCache[reqURI]
		m.reqCache[reqURI] = reqCnt + 1
		m.mtx.Unlock()
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		accessToken := r.Header.Get("Authorization")
		if matched, _ := regexp.Match(`^Bearer .`, []byte(accessToken)); !matched {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		m.ch <- accessToken

		// Check if path + query exists in the response map.
		pathWithQuery := r.URL.Path + "?" + r.URL.RawQuery

		if resp, ok := mockPathResp[pathWithQuery]; ok {
			//fmt.Printf("In mock metadatasever: found pathWithQuery %v", pathWithQuery)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
			return
		}

		if resp, ok := mockPathResp[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	fmt.Println("started iam server at " + m.GetURL())
	return m
}

// GetURL returns the URL of the MockIamServer.
func (m *MockIamServer) GetURL() string {
	return m.s.URL
}
