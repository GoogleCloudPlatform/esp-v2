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
		{
			desc:           "successful with query params ignored",
			url:            "https://abcde.youtube.com/api?query=ignored&query2=ignored2",
			wantedScheme:   "https",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     443,
			wantURI:        "/api",
			wantErr:        "",
		},
		{
			desc:           "successful for http IP with default port",
			url:            "http://127.0.0.1",
			wantedScheme:   "http",
			wantedHostname: "127.0.0.1",
			wantedPort:     80,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https IP with default port",
			url:            "https://127.0.0.1",
			wantedScheme:   "https",
			wantedHostname: "127.0.0.1",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for http IP with custom port",
			url:            "http://127.0.0.1:8080",
			wantedScheme:   "http",
			wantedHostname: "127.0.0.1",
			wantedPort:     8080,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https IP with custom port",
			url:            "https://127.0.0.1:8080",
			wantedScheme:   "https",
			wantedHostname: "127.0.0.1",
			wantedPort:     8080,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for grpc IP with default port",
			url:            "grpc://127.0.0.1",
			wantedScheme:   "grpc",
			wantedHostname: "127.0.0.1",
			wantedPort:     80,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for grpcs IP with default port",
			url:            "grpcs://127.0.0.1",
			wantedScheme:   "grpcs",
			wantedHostname: "127.0.0.1",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for grpc IP with custom port",
			url:            "grpc://127.0.0.1:8080",
			wantedScheme:   "grpc",
			wantedHostname: "127.0.0.1",
			wantedPort:     8080,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for grpcs IP with custom port",
			url:            "grpcs://127.0.0.1:8080",
			wantedScheme:   "grpcs",
			wantedHostname: "127.0.0.1",
			wantedPort:     8080,
			wantURI:        "",
			wantErr:        "",
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
		desc           string
		scheme         string
		httpProtocol   string
		wantedProtocol BackendProtocol
		wantedTLS      bool
		wantErr        string
	}{
		{
			desc:           "Good scheme: http",
			scheme:         "http",
			httpProtocol:   "http/1.1",
			wantedProtocol: HTTP1,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme and HTTP/2: http",
			scheme:         "http",
			httpProtocol:   "h2",
			wantedProtocol: HTTP2,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme and default http protocol: http",
			scheme:         "http",
			httpProtocol:   "",
			wantedProtocol: HTTP1,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: https",
			scheme:         "https",
			httpProtocol:   "http/1.1",
			wantedProtocol: HTTP1,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Good scheme and HTTP/2: https",
			scheme:         "https",
			httpProtocol:   "h2",
			wantedProtocol: HTTP2,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Good scheme and default http protocol: https",
			scheme:         "https",
			httpProtocol:   "",
			wantedProtocol: HTTP1,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: HTTP",
			scheme:         "HTTP",
			httpProtocol:   "http/1.1",
			wantedProtocol: HTTP1,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: HTTPS",
			scheme:         "HTTPS",
			httpProtocol:   "http/1.1",
			wantedProtocol: HTTP1,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: grpc",
			scheme:         "grpc",
			httpProtocol:   "http/1.1",
			wantedProtocol: GRPC,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: grpcs",
			scheme:         "grpcs",
			httpProtocol:   "http/1.1",
			wantedProtocol: GRPC,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: Upper case GRPC",
			scheme:         "GRPC",
			httpProtocol:   "http/1.1",
			wantedProtocol: GRPC,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme and http protocol ignored: grpc",
			scheme:         "grpc",
			httpProtocol:   "h2",
			wantedProtocol: GRPC,
			wantedTLS:      false,
			wantErr:        "",
		},
		{
			desc:           "Good scheme: upper case GRPCS",
			scheme:         "GRPCS",
			httpProtocol:   "http/1.1",
			wantedProtocol: GRPC,
			wantedTLS:      true,
			wantErr:        "",
		},
		{
			desc:           "Wrong scheme: rrr",
			scheme:         "rrr",
			httpProtocol:   "http/1.1",
			wantedProtocol: UNKNOWN,
			wantedTLS:      false,
			wantErr:        `unknown backend scheme [rrr], should be one of "http(s)" or "grpc(s)"`,
		},
		{
			desc:           "Wrong scheme: empty",
			scheme:         "",
			httpProtocol:   "http/1.1",
			wantedProtocol: UNKNOWN,
			wantedTLS:      false,
			wantErr:        `unknown backend scheme [], should be one of "http(s)" or "grpc(s)"`,
		},
		{
			desc:           "Wrong scheme rrrs but still TLS",
			scheme:         "rrrs",
			httpProtocol:   "http/1.1",
			wantedProtocol: UNKNOWN,
			wantedTLS:      true,
			wantErr:        `unknown backend scheme [rrr], should be one of "http(s)" or "grpc(s)"`,
		},
		{
			desc:           "Wrong http protocol: vvv",
			scheme:         "https",
			httpProtocol:   "vvv",
			wantedProtocol: UNKNOWN,
			wantedTLS:      true,
			wantErr:        `unknown backend http protocol [vvv], should be one of "http/1.1", "h2", or not set`,
		},
	}

	for i, tc := range testData {
		proto, tls, err := ParseBackendProtocol(tc.scheme, tc.httpProtocol)
		if proto != tc.wantedProtocol {
			t.Errorf("Test Desc(%d): %s, scheme is wrong, got: %v, want: %v", i, tc.desc, proto, tc.wantedProtocol)
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
			_, _ = w.Write(jwksUriEntry)
		}))
	openIDServer := httptest.NewServer(r)

	invalidOpenIDServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
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

func TestFetchConfigRelatedUrl(t *testing.T) {
	sm := "https://servicemanagement.googleapis.com"
	sn := "service-name"
	sc := "https://servicecontrol.googleapis.com"
	ci := "config-id"

	wantFetchRolloutIdUrl := "https://servicecontrol.googleapis.com/v1/services/service-name:report"
	if getFetchRolloutIdUrl := FetchRolloutIdURL(sc, sn); getFetchRolloutIdUrl != wantFetchRolloutIdUrl {
		t.Errorf("wantFetchRolloutIdUrl: %v, getFetchRolloutIdUrl: %v", wantFetchRolloutIdUrl, getFetchRolloutIdUrl)
	}

	wantFetchRolloutsUrl := "https://servicemanagement.googleapis.com/v1/services/service-name/rollouts?filter=status=SUCCESS"
	if getFetchRolloutsUrl := FetchRolloutsURL(sm, sn); getFetchRolloutsUrl != wantFetchRolloutsUrl {
		t.Errorf("wantFetchRolloutUrl: %v, getFetchRolloutUrl: %v", wantFetchRolloutsUrl, getFetchRolloutsUrl)
	}

	wantFetchConfigUrl := "https://servicemanagement.googleapis.com/v1/services/service-name/configs/config-id?view=FULL"
	if getFetchConfigUrl := FetchConfigURL(sm, sn, ci); getFetchConfigUrl != wantFetchConfigUrl {
		t.Errorf("wantFetchConfigUrl: %v, getFetchConfigUrl: %v", wantFetchConfigUrl, getFetchConfigUrl)
	}

}

func TestWildcardMatcherForPath(t *testing.T) {
	testData := []struct {
		desc        string
		uri         string
		wantMatcher string
	}{
		{
			desc:        "No path params",
			uri:         "/shelves",
			wantMatcher: "",
		},
		{
			desc:        "Path params with fieldpath-only bindings",
			uri:         "/shelves/{shelf_id}/books/{book.id}",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+$`,
		},
		{
			desc:        "Path params with fieldpath-only bindings and verb",
			uri:         "/shelves/{shelf_id}/books/{book.id}:checkout",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+:checkout$`,
		},
		{
			desc:        "Path param with wildcard segments",
			uri:         "/test/*/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with wildcard segments and verb",
			uri:         "/test/*/test/**:upload",
			wantMatcher: `^/test/[^\/]+/test/.*:upload$`,
		},
		{
			desc:        "Path param with wildcard in segment binding",
			uri:         "/test/{x=*}/test/{y=**}",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with mixed wildcards",
			uri:         "/test/{name=*}/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Invalid http template, not preceded by '/' ",
			uri:         "**",
			wantMatcher: "",
		},
		{
			desc:        "Path params with full segment binding",
			uri:         "/v1/{name=books/*}",
			wantMatcher: `^/v1/books/[^\/]+$`,
		},
		{
			desc:        "Path params with multiple field path segment bindings",
			uri:         "/v1/{test=a/b/*}/route/{resource_id=shelves/*/books/**}:upload",
			wantMatcher: `^/v1/a/b/[^\/]+/route/shelves/[^\/]+/books/.*:upload$`,
		},
		{
			// TODO(nareddyt): How can we improve validation once we remove path matcher?
			desc:        "BUG - Incorrect http template syntax is not validated",
			uri:         "/v1/{name=/books/*}",
			wantMatcher: `^/v1//books/[^\/]+$`,
		},
	}

	for _, tc := range testData {
		got := WildcardMatcherForPath(tc.uri)

		if tc.wantMatcher != got {
			t.Errorf("Test (%v): \n got %v \nwant %v", tc.desc, got, tc.wantMatcher)
		}
	}
}

func TestSnakeNameToJsonNameInPathParam(t *testing.T) {

	testCases := []struct {
		desc                 string
		uri                  string
		snakeNameToJsonNames map[string]string
		wantUri              string
		wantError            string
	}{
		{
			desc: "variable type {x}",
			uri:  "/a/{x_y}/b",
			snakeNameToJsonNames: map[string]string{
				"x_y": "xY",
			},
			wantUri: "/a/{xY}/b",
		},
		{
			desc: "variable type {x=*}",
			uri:  "/a/{x_y=*}/b",
			snakeNameToJsonNames: map[string]string{
				"x_y": "xY",
			},
			wantUri: "/a/{xY=*}/b",
		},
		{
			desc: "variable type {x.y.z=*}",
			uri:  "/a/{x_y.a_b=*}/b",
			snakeNameToJsonNames: map[string]string{
				"x_y": "xY",
				"a_b": "aB",
			},
			wantUri: "/a/{xY.aB=*}/b",
		},
		{
			desc: "snake name not found",
			uri:  "/a/{x_y}/b",
			snakeNameToJsonNames: map[string]string{
				"a_b": "aB",
			},
			wantUri: "/a/{x_y}/b",
		},
		{
			desc: "snake name found but not as variable",
			uri:  "/x_y/{x_y_foo}/{x_y_bar=*}",
			snakeNameToJsonNames: map[string]string{
				"x_y": "xY",
			},
			wantUri: "/x_y/{x_y_foo}/{x_y_bar=*}",
		},
	}

	for _, tc := range testCases {
		getUri := SnakeNamesToJsonNamesInPathParam(tc.uri, tc.snakeNameToJsonNames)

		if getUri != tc.wantUri {
			t.Errorf("Test(%s) fail, want uri: %s, get uri: %s", tc.desc, tc.wantUri, getUri)
		}

	}

}
