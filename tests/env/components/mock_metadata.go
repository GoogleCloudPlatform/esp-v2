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

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
)

const (
	fakeToken       = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	fakeServiceName = "test-service"
	fakeConfigID    = "test-config"
	fakeZonePath    = "projects/4242424242/zones/test-zone"
	fakeProjectID   = "test-project-id"
)

var defaultResp = map[string]string{
	util.ConfigIDSuffix:            fakeConfigID,
	util.ServiceNameSuffix:         fakeServiceName,
	util.ServiceAccountTokenSuffix: fakeToken,
	util.ProjectIDSuffix:           fakeProjectID,
	util.ZoneSuffix:                fakeZonePath,
}

// MockMetadata mocks the Metadata server.
type MockMetadataServer struct {
	s    *httptest.Server
	resp map[string]string
}

// NewMockMetadata creates a new HTTP server.
func NewMockMetadata(pathResp map[string]string) *MockMetadataServer {
	mockPathResp := make(map[string]string)
	for k, v := range defaultResp {
		mockPathResp[k] = v
	}

	if pathResp != nil {
		for k, v := range pathResp {
			mockPathResp[k] = v
		}
	}

	return &MockMetadataServer{
		s: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// Root is used to tell if the sever is healthy or not.
			if r.URL.Path == "" || r.URL.Path == "/" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if resp, ok := mockPathResp[r.URL.Path]; ok {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(resp))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))}
}

// GetURL returns the URL of the MockMetadataServer.
func (m *MockMetadataServer) GetURL() string {
	return m.s.URL
}
