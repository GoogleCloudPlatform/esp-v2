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

package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestParseURI(t *testing.T) {
	testData := []struct {
		desc           string
		url            string
		wantedScheme   string
		wantedHostname string
		wantedPort     uint32
		wantURI        string
		wantErr        string
	}{
		{
			desc:           "successful for https url, ends without slash",
			url:            "https://abc.example.org",
			wantedScheme:   "https",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https url, ends with slash",
			url:            "https://abcde.google.org/",
			wantedScheme:   "https",
			wantedHostname: "abcde.google.org",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https url, ends with path",
			url:            "https://abcde.youtube.com/api/",
			wantedScheme:   "https",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     443,
			wantURI:        "/api",
			wantErr:        "",
		},
		{
			desc:           "successful for https url with custom port",
			url:            "https://abcde.youtube.com:8989/api/",
			wantedScheme:   "https",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     8989,
			wantURI:        "/api",
			wantErr:        "",
		},
		{
			desc:           "successful for http url",
			url:            "http://abcde.youtube.com:8989/api/",
			wantedScheme:   "http",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     8989,
			wantURI:        "/api",
		},
		{
			desc:           "successful for https url, path ends with slash",
			url:            "https://abc.example.org/path/to/",
			wantedScheme:   "https",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "/path/to",
		},
		{
			desc:           "successful for https url, path ends without slash",
			url:            "https://abc.example.org/path",
			wantedScheme:   "https",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "/path",
		},
		{
			desc:           "successful for grpc",
			url:            "grpc://abc.example.org",
			wantedScheme:   "grpc",
			wantedHostname: "abc.example.org",
			wantedPort:     80,
			wantURI:        "",
		},
		{
			desc:           "successful for grpcs",
			url:            "grpcs://abc.example.org",
			wantedScheme:   "grpcs",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "",
		},
		{
			desc:           "default port depends on last char",
			url:            "r://abc.example.org",
			wantedScheme:   "r",
			wantedHostname: "abc.example.org",
			wantedPort:     80,
			wantURI:        "",
		},
		{
			desc:           "default port depends on last char",
			url:            "s://abc.example.org",
			wantedScheme:   "s",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "",
		},
		{
			desc:           "successful for https url with port and path in the same time",
			url:            "https://abc.example.org:8000/path",
			wantedScheme:   "https",
			wantedHostname: "abc.example.org",
			wantedPort:     8000,
			wantURI:        "/path",
		},
		{
			desc:           "successful for url without scheme",
			url:            "abc.example.org",
			wantedScheme:   "https",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "",
		},
	}

	for i, tc := range testData {
		scheme, hostname, port, uri, err := ParseURI(tc.url)
		if scheme != tc.wantedScheme {
			t.Errorf("Test Desc(%d): %s, extract backend address scheme, got: %v, want: %v", i, tc.desc, scheme, tc.wantedScheme)
		}
		if hostname != tc.wantedHostname {
			t.Errorf("Test Desc(%d): %s, extract backend address hostname got: %v, want: %v", i, tc.desc, hostname, tc.wantedHostname)
		}
		if port != tc.wantedPort {
			t.Errorf("Test Desc(%d): %s, extract backend address port got: %v, want: %v", i, tc.desc, port, tc.wantedPort)
		}
		if uri != tc.wantURI {
			t.Errorf("Test Desc(%d): %s, extract backend address uri got: %v, want: %v", i, tc.desc, uri, tc.wantURI)
		}
		if (err == nil && tc.wantErr != "") || (err != nil && err.Error() != tc.wantErr) {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, err, tc.wantErr)
		}
	}
}

func TestParseBackendProtocol(t *testing.T) {
	testData := []struct {
		desc        string
		proto       string
		wantedProto BackendProtocol
		wantedTLS   bool
		wantErr     string
	}{
		{
			desc:        "Good scheme: http",
			proto:       "http",
			wantedProto: HTTP1,
			wantedTLS:   false,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: https",
			proto:       "https",
			wantedProto: HTTP1,
			wantedTLS:   true,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: HTTP",
			proto:       "HTTP",
			wantedProto: HTTP1,
			wantedTLS:   false,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: HTTPS",
			proto:       "HTTPS",
			wantedProto: HTTP1,
			wantedTLS:   true,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: grpc",
			proto:       "grpc",
			wantedProto: GRPC,
			wantedTLS:   false,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: grpcs",
			proto:       "grpcs",
			wantedProto: GRPC,
			wantedTLS:   true,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: Upper case GRPC",
			proto:       "GRPC",
			wantedProto: GRPC,
			wantedTLS:   false,
			wantErr:     "",
		},
		{
			desc:        "Good scheme: upper case GRPCS",
			proto:       "GRPCS",
			wantedProto: GRPC,
			wantedTLS:   true,
			wantErr:     "",
		},
		{
			desc:        "Wrong scheme: rrr",
			proto:       "rrr",
			wantedProto: HTTP1,
			wantedTLS:   false,
			wantErr:     `unknown backend scheme [rrr], should be one of "http(s)" or "grpc(s)"`,
		},
		{
			desc:        "Wrong scheme: empty",
			proto:       "",
			wantedProto: HTTP1,
			wantedTLS:   false,
			wantErr:     `unknown backend scheme [], should be one of "http(s)" or "grpc(s)"`,
		},
	}

	for i, tc := range testData {
		proto, tls, err := ParseBackendProtocol(tc.proto)
		if proto != tc.wantedProto {
			t.Errorf("Test Desc(%d): %s, proto is wrong, got: %v, want: %v", i, tc.desc, proto, tc.wantedProto)
		}
		if tls != tc.wantedTLS {
			t.Errorf("Test Desc(%d): %s, TLS is wrong, got: %v, want: %v", i, tc.desc, tls, tc.wantedTLS)
		}
		if (err == nil && tc.wantErr != "") || (err != nil && err.Error() != tc.wantErr) {
			t.Errorf("Test Desc(%d): %s, error is wrong, got: %v, want: %v", i, tc.desc, err, tc.wantErr)
		}
	}
}

func TestResolveJwksUriUsingOpenID(t *testing.T) {
	r := mux.NewRouter()
	jwksUriEntry, _ := json.Marshal(map[string]string{"jwks_uri": "this-is-jwksUri"})
	r.Path(OpenIDDiscoveryCfgURLSuffix).Methods("GET").Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(jwksUriEntry)
		}))
	openIDServer := httptest.NewServer(r)

	invalidOpenIDServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{}"))
	}))

	testData := []struct {
		desc    string
		issuer  string
		wantUri string
		wantErr string
	}{
		{
			desc:    "Success, with correct jwks_uri",
			issuer:  openIDServer.URL,
			wantUri: "this-is-jwksUri",
		},
		{
			desc:    "Fail, with wrong jwks_uri entry in openIDServer",
			issuer:  invalidOpenIDServer.URL,
			wantErr: "Invalid jwks_uri",
		},
		{
			desc:    "Fail, with non-exist server referred by issuer using openID",
			issuer:  "http://aaaaaaa.bbbbbbbbbbbbb.cccccccccc",
			wantErr: "Failed to fetch jwks_uri from http://aaaaaaa.bbbbbbbbbbbbb.cccccccccc/.well-known/openid-configuration",
		},
	}
	for i, tc := range testData {
		uri, err := ResolveJwksUriUsingOpenID(tc.issuer)
		if uri != tc.wantUri {
			t.Errorf("Test Desc(%d): %s, resolve jwksUri by openID got: %v, want: %v", i, tc.desc, uri, tc.wantUri)
		}
		if (err == nil && tc.wantErr != "") || (err != nil && !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("Test Desc(%d): %s, resolve jwksUri by openID got: %v, want: %v", i, tc.desc, err, tc.wantErr)
		}
	}

}

func TestExtraAddressFromURI(t *testing.T) {
	testData := []struct {
		desc          string
		uri           string
		wantedAddress string
		wantedError   string
	}{
		{
			desc:          "Succeeded to parse uri",
			uri:           "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com",
			wantedAddress: "www.googleapis.com:443",
		},
		{
			desc:        "Failed with wrong-format uri",
			uri:         "%",
			wantedError: "Fail to parse uri %",
		},
	}

	for i, tc := range testData {
		generatedClusterName, err := ExtraAddressFromURI(tc.uri)
		if generatedClusterName != tc.wantedAddress {
			t.Errorf("Test Desc(%d): %s, ExtraAddressFromURI got: %v, want: %v", i, tc.desc, generatedClusterName, tc.wantedAddress)
		}
		if err != nil && !strings.Contains(err.Error(), tc.wantedError) {
			t.Errorf("Test Desc(%d): %s, ExtraAddressFromURI got: %v, want: %v", i, tc.desc, err.Error(), tc.wantedError)
		}
	}
}
