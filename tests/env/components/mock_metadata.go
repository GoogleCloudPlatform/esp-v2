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
	"sync"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
)

const (
	fakeToken         = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	fakeIdentityToken = "ya29.new"
	fakeServiceName   = "test-service"
	fakeConfigID      = "test-config"
	fakeZonePath      = "projects/4242424242/zones/test-zone"
	FakeProjectID     = "test-project-id"
)

var defaultResp = map[string]string{
	util.ConfigIDPath:      fakeConfigID,
	util.ServiceNamePath:   fakeServiceName,
	util.AccessTokenPath:   fakeToken,
	util.IdentityTokenPath: fakeIdentityToken,
	util.ProjectIDPath:     FakeProjectID,
	util.ZonePath:          fakeZonePath,
}

// MockMetadataServer mocks the Metadata server.
type MockMetadataServer struct {
	s        *httptest.Server
	reqCache map[string]int
	mtx      sync.RWMutex

	// ID Token Subscribers make a call for each audience at the same time.
	// Debounce multiple requests with this.
	retryHandler *RetryHandler
}

// NewMockMetadata creates a new HTTP server.
func NewMockMetadata(pathResp map[string]string, wantNumFails int) *MockMetadataServer {
	mockPathResp := make(map[string]string)
	for k, v := range defaultResp {
		mockPathResp[k] = v
	}

	for k, v := range pathResp {
		mockPathResp[k] = v
	}

	m := &MockMetadataServer{
		reqCache:     make(map[string]int),
		retryHandler: NewRetryHandler(wantNumFails),
	}
	m.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		glog.Infof("Fake IMDS handling request: %v %v", r.Method, r.URL)
		w.Header().Set("Content-Type", "application/json")

		if m.retryHandler.handleRetryExceptFirst(w) {
			return
		}

		reqURI := r.URL.RequestURI()
		m.mtx.Lock()
		reqCnt, _ := m.reqCache[reqURI]
		m.reqCache[reqURI] = reqCnt + 1
		m.mtx.Unlock()
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Check if path + query exists in the response map.
		pathWithQuery := r.URL.Path + "?" + r.URL.RawQuery
		if resp, ok := mockPathResp[pathWithQuery]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
			return
		}

		if resp, ok := mockPathResp[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	fmt.Println("started metadata server at " + m.GetURL())
	return m
}

// GetURL returns the URL of the MockMetadataServer.
func (m *MockMetadataServer) GetURL() string {
	return m.s.URL
}

func (m *MockMetadataServer) GetReqCnt(reqURI string) int {
	m.mtx.RLock()
	reqCnt, _ := m.reqCache[reqURI]
	m.mtx.RUnlock()
	return reqCnt
}
