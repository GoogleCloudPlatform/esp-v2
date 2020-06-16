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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"
)

// MockIamServer mocks the Metadata server.
type MockIamServer struct {
	s          *httptest.Server
	reqBodyCh  chan string
	reqTokenCh chan string

	// ID Token Subscribers make a call for each audience at the same time.
	// Debounce multiple requests with this.
	retryHandler *RetryHandler
}

// NewMockMetadata creates a new HTTP server.
func NewIamMetadata(pathResp map[string]string, wantNumFails int, respTime time.Duration) *MockIamServer {
	mockPathResp := make(map[string]string)
	for k, v := range pathResp {
		mockPathResp[k] = v
	}

	m := &MockIamServer{
		reqTokenCh:   make(chan string, 100),
		reqBodyCh:    make(chan string, 100),
		retryHandler: NewRetryHandler(wantNumFails),
	}
	m.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Test timeouts and retries.
		time.Sleep(respTime)
		if m.retryHandler.handleRetry(w) {
			return
		}

		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if matched, _ := regexp.Match(`^Bearer .`, []byte(authHeader)); !matched {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		m.reqTokenCh <- authHeader
		if r.Body != nil {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			m.reqBodyCh <- string(bodyBytes)
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
			return
		}

		// To allow envoy to start-up when fetching all id tokens, default to some OK response.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":  "default-test-id-token"}`))
	}))
	fmt.Println("started iam server at " + m.GetURL())
	return m
}

// GetURL returns the URL of the MockIamServer.
func (m *MockIamServer) GetURL() string {
	return m.s.URL
}

func (m *MockIamServer) GetRequestToken() (string, error) {
	select {
	case d := <-m.reqTokenCh:
		return d, nil
	case <-time.After(2500 * time.Millisecond):
		return "", fmt.Errorf("Timeout")
	}
}

func (m *MockIamServer) GetRequestBody() (string, error) {
	select {
	case d := <-m.reqBodyCh:
		return d, nil
	case <-time.After(2500 * time.Millisecond):
		return "", fmt.Errorf("Timeout")
	}
}
