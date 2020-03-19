// Copyright 2020 Google LLC
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func initServerForTestCallWithAccessToken(t *testing.T, desc, expectMethod, expectToken string, respBody []byte, respStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method; got != expectMethod {
			t.Errorf("test(%v) fail, expect Method: %s, get Method: %s", desc, expectMethod, got)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer "+expectToken {
			t.Errorf("test(%v) fail, expect Authorization: %s, get Authorization: Bearer %v", desc, expectToken, got)
		}

		if got := r.Header.Get("Content-Type"); got != "application/x-protobuf" {
			t.Errorf("test(%v) fail, expect Content-Type: application/x-protobuf, get Content-Type: %s", desc, got)
		}

		if respBody != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write(respBody)
			if err != nil {
				t.Fatalf("test(%s) fail, fail to write response: %v", desc, err)
			}
		}

		if respStatus != 0 {
			w.WriteHeader(http.StatusForbidden)
		}
	}))
}

func TestCallWithAccessToken(t *testing.T) {

	testCase := []struct {
		desc        string
		method      string
		expectToken string
		respBody    []byte
		respStatus  int
		expectError string
	}{
		{
			desc:        "successful request",
			method:      "GET",
			expectToken: "this-is-token",
			respBody:    []byte("this-is-resp-body"),
		},
		{
			desc:        "failed request with 403",
			method:      "GET",
			expectToken: "this-is-token",
			respStatus:  http.StatusForbidden,
			expectError: "returns not 200 OK: 403 Forbidden",
		},
	}

	for _, tc := range testCase {
		s := initServerForTestCallWithAccessToken(t, tc.desc, tc.method, tc.expectToken, tc.respBody, tc.respStatus)

		gotBody, err := CallWithAccessToken(http.Client{}, tc.method, s.URL, tc.expectToken)

		if err != nil {
			if tc.expectError == "" {
				t.Errorf("test(%v) fail, get response error: %v", tc.desc, err)

			} else if !strings.Contains(err.Error(), tc.expectError) {
				t.Errorf("test(%v) fail, expect response error: %v, get response error: %v", tc.desc, tc.expectError, err)
			}
			continue
		}

		if string(gotBody) != string(tc.respBody) {
			t.Errorf("test(%v) fail, expect response body: %v, get response body: %v", tc.desc, tc.respBody, gotBody)
		}
	}
}
