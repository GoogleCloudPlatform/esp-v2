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

  testData := []struct {
    desc            string
    curToken        string
    curTokenTimeout time.Time
    expectedToken   string
  }{
    {
      desc:          "Empty metadata",
      expectedToken: "ya29.new",
    },
    {
      desc:            "token has expired in metadata",
      curToken:        "ya29.expired",
      curTokenTimeout: time.Now().Add(-1 * time.Hour),
      expectedToken:   "ya29.new",
    },
    {
      desc:            "token is not expired in metadata",
      curToken:        "ya29.nonexpired",
      curTokenTimeout: time.Now().Add(1 * time.Hour),
      expectedToken:   "ya29.nonexpired",
    },
  }
  for i, tc := range testData {
    if tc.curToken != "" {
      metadata.accessToken = tc.curToken
      metadata.tokenTimeout = tc.curTokenTimeout
    }
    token, err := fetchAccessToken()
    if err != nil {
      t.Fatal(err)
    }
    if !strings.EqualFold(token, tc.expectedToken) {
      t.Errorf("Test Desc(%d): %s, FetchServiceAccountToken = %s, want %s", i, tc.desc, token, tc.expectedToken)
    }
  }
}

func initMockMetadataServer() *httptest.Server {
  return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(fakeToken))
  }))
}
