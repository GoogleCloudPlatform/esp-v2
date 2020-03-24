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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

func initServerForTestCallWithAccessToken(t *testing.T, desc, wantMethod, wantToken string, respBody []byte, respStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method; got != wantMethod {
			t.Errorf("test(%v) fail, want Method: %s, get Method: %s", desc, wantMethod, got)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer "+wantToken {
			t.Errorf("test(%v) fail, want Authorization: %s, get Authorization: Bearer %v", desc, wantToken, got)
		}

		if got := r.Header.Get("Content-Type"); got != "application/x-protobuf" {
			t.Errorf("test(%v) fail, want Content-Type: application/x-protobuf, get Content-Type: %s", desc, got)
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

func TestCallGoogleapis(t *testing.T) {
	normalTokenFunc := func() (string, time.Duration, error) { return "this-is-token", time.Duration(100), nil }
	testCase := []struct {
		desc          string
		method        string
		token         GetAccessTokenFunc
		respBody      []byte
		respStatus    int
		unmarshalFunc func(input []byte, output proto.Message) error
		wantError     string
	}{
		{
			desc:     "success",
			method:   "GET",
			token:    normalTokenFunc,
			respBody: []byte("this-is-resp-body"),
		},
		{
			desc:   "fail to get access token",
			method: "GET",
			token: func() (string, time.Duration, error) {
				return "", time.Duration(100), fmt.Errorf("fail to talk to imds")
			},
			wantError: "fail to get access token: fail to talk to imds",
		},
		{
			desc:       "fail to talk to googleapis service",
			method:     "GET",
			token:      normalTokenFunc,
			respStatus: http.StatusForbidden,
			wantError:  "http call to GET %URL returns not 200 OK: 403 Forbidden",
		},
		{
			desc:   "fail to unmarshal response",
			method: "GET",
			token:  normalTokenFunc,
			unmarshalFunc: func(input []byte, output proto.Message) error {
				return fmt.Errorf("fail to unmarshal")
			},
			wantError: "fail to unmarshal",
		},
	}

	for _, tc := range testCase {
		token, _, _ := tc.token()
		s := initServerForTestCallWithAccessToken(t, tc.desc, tc.method, token, tc.respBody, tc.respStatus)
		if tc.unmarshalFunc == nil {
			UnmarshalBytesToPbMessage = func(gotBody []byte, output proto.Message) error {
				if string(gotBody) != string(tc.respBody) {
					return fmt.Errorf("test(%v) fail, want response body: %v, get response body: %v", tc.desc, tc.respBody, gotBody)
				}
				return nil
			}
		} else {
			UnmarshalBytesToPbMessage = tc.unmarshalFunc
		}

		err := CallGoogleapis(&http.Client{}, s.URL, tc.method, tc.token, nil)

		if err != nil {
			if tc.wantError == "" {
				t.Errorf("test(%v) fail, get response error: %v", tc.desc, err)
				continue
			}

			tc.wantError = strings.Replace(tc.wantError, "%URL", s.URL, 1)
			if err.Error() != tc.wantError {
				t.Errorf("test(%v) fail, want response error: %v, get response error: %v", tc.desc, tc.wantError, err)
			}
			continue
		}
	}
}
