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

	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/golang/protobuf/proto"
)

const (
	fakeToken       = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	fakeServiceName = "echo-service"
	fakeConfigID    = "canary-config"
	fakeZonePath    = "projects/4242424242/zones/us-west-1b"
	fakeZone        = "us-west-1b"
	fakeProjectID   = "gcpproxy"
)

func TestFetchAccountTokenExpired(t *testing.T) {
	ts := initMockMetadataServer(fakeToken)
	defer ts.Close()
	fetchMetadataURL = func(_ string) string {
		return ts.URL
	}
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

func TestFetchServiceName(t *testing.T) {
	ts := initMockMetadataServer(fakeServiceName)
	defer ts.Close()

	fetchMetadataURL = func(_ string) string {
		return ts.URL
	}
	name, err := fetchServiceName()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(name, fakeServiceName) {
		t.Errorf("fetchServiceName = %s, want %s", name, fakeServiceName)
	}
}

func TestFetchConfigId(t *testing.T) {
	ts := initMockMetadataServer(fakeConfigID)
	defer ts.Close()

	fetchMetadataURL = func(_ string) string {
		return ts.URL
	}
	name, err := fetchConfigId()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(name, fakeConfigID) {
		t.Errorf("fetchServiceName = %s, want %s", name, fakeConfigID)
	}
}

func TestFetchGCPAttributes(t *testing.T) {
	testData := []struct {
		desc                  string
		mockedResp            map[string]string
		expectedGCPAttributes *scpb.GcpAttributes
	}{
		{
			desc: "ProjectID",
			mockedResp: map[string]string{
				projectIDSuffix: fakeProjectID,
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				ProjectId: fakeProjectID,
				Platform:  util.GCE,
			},
		},
		{
			desc: "Valid Zone",
			mockedResp: map[string]string{
				zoneSuffix: fakeZonePath,
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Zone:     fakeZone,
				Platform: util.GCE,
			},
		},
		{
			desc: "Invalid Zone - without '/'",
			mockedResp: map[string]string{
				zoneSuffix: "noslash",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GCE,
			},
		},
		{
			desc: "Invalid Zone - ends with '/'",
			mockedResp: map[string]string{
				zoneSuffix: "project/123123/",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GCE,
			},
		},
		{
			desc: "Platform - GAE_FLEX",
			mockedResp: map[string]string{
				gaeServerSoftwareSuffix: "foo",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GAEFlex,
			},
		},
		{
			desc: "Platform - GKE",
			mockedResp: map[string]string{
				kubeEnvSuffix: "foo",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GKE,
			},
		},
		{
			desc:       "Platform - GCE",
			mockedResp: map[string]string{},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GCE,
			},
		},
		{
			desc: "Combined - ProjectID, Zone, and Platform",
			mockedResp: map[string]string{
				projectIDSuffix:         fakeProjectID,
				zoneSuffix:              fakeZonePath,
				gaeServerSoftwareSuffix: "foo",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				ProjectId: fakeProjectID,
				Zone:      fakeZone,
				Platform:  util.GAEFlex,
			},
		},
		{
			desc:                  "No MetadataServer",
			mockedResp:            nil,
			expectedGCPAttributes: nil,
		},
	}

	errorTmpl := "Test: %s\n  Expected: %s\n  Actual: %s"
	for _, tc := range testData {
		ts := initMockMetadataServerFromPathResp(tc.mockedResp)
		defer ts.Close()
		if tc.mockedResp == nil {
			fetchMetadataURL = func(suffix string) string {
				return "non-existing-url" + suffix
			}
		} else {
			fetchMetadataURL = func(suffix string) string {
				return ts.URL + suffix
			}
		}

		attrs := fetchGCPAttributes()
		if tc.expectedGCPAttributes == nil && attrs == nil {
			continue
		}

		if attrs == nil {
			t.Errorf(errorTmpl, tc.desc, tc.expectedGCPAttributes, attrs)
			continue
		}

		if !proto.Equal(attrs, tc.expectedGCPAttributes) {
			t.Errorf(errorTmpl, tc.desc, tc.expectedGCPAttributes, attrs)
		}
	}

}

func initMockMetadataServer(resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
}

func initMockMetadataServerFromPathResp(pathResp map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Root is used to tell if the sever is healthy or not.
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if resp, ok := pathResp[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
