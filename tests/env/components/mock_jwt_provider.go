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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"github.com/golang/glog"
	"github.com/gorilla/mux"

	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// These addresses must be hardcoded to match the keys generated in fake_jwt.go
const (
	openIDProviderAddr         = "127.0.0.1:32025"
	openIDInvalidProviderAddr  = "127.0.0.1:32026"
	openIDNonexistProviderAddr = "127.0.0.1:32027"
)

type FakeJwtService struct {
	ProviderMap map[string]*MockJwtProvider
}

// MockJwtProvider mocks the Jwt provider.
type MockJwtProvider struct {
	s            *httptest.Server
	cnt          *int32
	AuthProvider *scpb.AuthProvider
}

// Returns a FakeJwtService that is ready to use. All servers will be started.
func NewFakeJwtService() *FakeJwtService {
	return &FakeJwtService{
		ProviderMap: make(map[string]*MockJwtProvider, 20),
	}
}

// Setup non-OpenId providers.
func (fjs *FakeJwtService) SetupJwt() error {

	// Setup non-OpenID providers
	for _, config := range testdata.ProviderConfigs {
		var provider *MockJwtProvider

		// Create fake provider
		if config.IsInvalid {
			provider = newMockInvalidJwtProvider()
		} else if config.IsNonexistent {
			provider = &MockJwtProvider{}
		} else {
			provider = newMockJwtProvider(config.Keys)
		}

		// Set auth id and issuer
		provider.AuthProvider = &scpb.AuthProvider{
			Id:     config.Id,
			Issuer: config.Issuer,
		}

		// Set auth uri
		if config.IsNonexistent {
			provider.AuthProvider.JwksUri = config.HardcodedJwksUri
		} else {
			provider.AuthProvider.JwksUri = provider.GetURL()
		}

		// Save provider
		fjs.ProviderMap[config.Id] = provider
		glog.Infof("Setup JWT provider %v with JwksUri %v", config.Id, provider.AuthProvider.JwksUri)
	}

	return nil
}

// Setup OpenId providers.
func (fjs *FakeJwtService) SetupOpenId() error {
	// Test Jwks and Jwt Tokens are generated following
	// https://github.com/istio/istio/tree/master/security/tools/jwt/samples.
	openID := newMockJwtProvider(testdata.ServiceControlJwtPayloadPubKeys)
	glog.Infof("Setup JWT provider open_id with address %v", openID.GetURL())

	// OpenIdProvider
	jwksUriEntry, err := json.Marshal(map[string]string{"jwks_uri": openID.GetURL()})
	if err != nil {
		return err
	}
	provider, err := newOpenIDServer(openIDProviderAddr, string(jwksUriEntry))
	if err != nil {
		return fmt.Errorf("fail to init provider %s, %v", testdata.OpenIdProvider, err)
	}
	provider.AuthProvider = &scpb.AuthProvider{
		Id:     testdata.OpenIdProvider,
		Issuer: provider.GetURL(),
	}
	fjs.ProviderMap[provider.AuthProvider.Id] = provider
	glog.Infof("Setup OpenID JWT provider server %v with Issuer %v", provider.AuthProvider.Id, provider.AuthProvider.Issuer)

	// OpenIdInvalidProvider
	jwksUriEntry, err = json.Marshal(map[string]string{"issuer": openID.GetURL()})
	if err != nil {
		return err
	}
	provider, err = newOpenIDServer(openIDInvalidProviderAddr, string(jwksUriEntry))
	if err != nil {
		return fmt.Errorf("fail to init provider %s, %v", "openID_invalid_provier", err)
	}
	provider.AuthProvider = &scpb.AuthProvider{
		Id:     testdata.OpenIdInvalidProvider,
		Issuer: provider.GetURL(),
	}
	fjs.ProviderMap[provider.AuthProvider.Id] = provider
	glog.Infof("Setup OpenID JWT provider server %v with Issuer %v", provider.AuthProvider.Id, provider.AuthProvider.Issuer)

	// OpenIdNonexistentProvider
	provider = &MockJwtProvider{}
	provider.AuthProvider = &scpb.AuthProvider{
		Id:     testdata.OpenIdNonexistentProvider,
		Issuer: fmt.Sprintf("http://%v", openIDNonexistProviderAddr),
	}
	fjs.ProviderMap[provider.AuthProvider.Id] = provider
	glog.Infof("Setup OpenID JWT provider server %v with Issuer %v", provider.AuthProvider.Id, provider.AuthProvider.Issuer)

	return nil
}

func (fjs *FakeJwtService) ResetReqCnt(provider string) {
	mockJwtProvider := fjs.ProviderMap[provider]
	atomic.SwapInt32(mockJwtProvider.cnt, 0)
}

// newMockJwtProvider creates a new Jwt provider.
func newMockJwtProvider(jwks string) *MockJwtProvider {
	mockJwtProvider := &MockJwtProvider{
		cnt: new(int32),
	}
	mockJwtProvider.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(mockJwtProvider.cnt, 1)
		w.Write([]byte(jwks))
	}))
	return mockJwtProvider
}

// newMockInvalidJwtProvider creates a new Jwt provider which returns error.
func newMockInvalidJwtProvider() *MockJwtProvider {
	mockJwtProvider := &MockJwtProvider{
		cnt: new(int32),
	}
	mockJwtProvider.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(mockJwtProvider.cnt, 1)
		http.Error(w, `{"code": 503, "message": "service not found"}`, 503)
	}))
	return mockJwtProvider
}

// newOpenIDServer creates a new Jwt provider with fixed address.
func newOpenIDServer(addr, jwksUriEntry string) (*MockJwtProvider, error) {
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
