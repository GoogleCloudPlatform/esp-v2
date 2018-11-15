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

package configmanager

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	fakeToken = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
)

func TestFetchAccountTokenExpired(t *testing.T) {
	ts := initMockMetadataServer()
	defer ts.Close()
	serviceAccountTokenURL = ts.URL
	// Get a time stamp and use it through out the test.
	fakeNow := time.Now()
	// Make sure the accessToken is empty before the test.
	metadata.accessToken = ""
	// Mock the timeNow function.
	timeNow = func() time.Time { return fakeNow }

	testData := []struct {
		desc               string
		curToken           string
		curTokenTimeout    time.Time
		expectedToken      string
		expectedExpiration time.Duration
	}{
		{
			desc:               "Empty metadata",
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
		{
			desc:               "token has expired in metadata",
			curToken:           "ya29.expired",
			curTokenTimeout:    fakeNow.Add(-1 * time.Hour),
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
		{
			desc:               "token is not expired in metadata",
			curToken:           "ya29.nonexpired",
			curTokenTimeout:    fakeNow.Add(61 * time.Second),
			expectedToken:      "ya29.nonexpired",
			expectedExpiration: 61 * time.Second,
		},
		{
			desc:               "token valid time is below 60 seconds in metadata",
			curToken:           "ya29.nonexpired",
			curTokenTimeout:    fakeNow.Add(59 * time.Second),
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
	}
	for i, tc := range testData {
		if tc.curToken != "" {
			metadata.accessToken = tc.curToken
			metadata.tokenTimeout = tc.curTokenTimeout
		}
		token, expires, err := fetchAccessToken()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(token, tc.expectedToken) {
			t.Errorf("Test Desc(%d): %s, FetchServiceAccountToken = %s, want %s", i, tc.desc, token, tc.expectedToken)
		}
		if expires != tc.expectedExpiration {
			t.Errorf("Test Desc(%d): %s, Actual expiration = %s, want %s", i, tc.desc, expires, tc.expectedExpiration)
		}
	}
}

func initMockMetadataServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeToken))
	}))
}
