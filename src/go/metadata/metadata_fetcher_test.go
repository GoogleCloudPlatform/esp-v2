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

package metadata

import (
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
)

const (
	fakeToken            = `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	fakeIdentityJwtToken = "ya29.new"
	fakeServiceName      = "echo-service"
	fakeConfigID         = "canary-config"
	fakeZonePath         = "projects/4242424242/zones/us-west-1b"
	fakeZone             = "us-west-1b"
	fakeProjectID        = "gcpproxy"
)

type testToken struct {
	desc               string
	curToken           string
	curTokenTimeout    time.Time
	expectedToken      string
	expectedExpiration time.Duration
}

func TestFetchAccountTokenExpired(t *testing.T) {
	ts := util.InitMockServer(fakeToken)
	defer ts.Close()

	// Get a time stamp and use it through out the test.
	fakeNow := time.Now()

	mf := NewMockMetadataFetcher(ts.GetURL(), fakeNow)

	// Make sure the accessToken is empty before the test.
	mf.tokenInfo.accessToken = ""

	testData := []testToken{
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
			mf.tokenInfo.accessToken = tc.curToken
			mf.tokenInfo.tokenTimeout = tc.curTokenTimeout
		}
		token, expires, err := mf.FetchAccessToken()
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

func TestFetchIdentityJWTTokenBasic(t *testing.T) {
	ts := util.InitMockServer(fakeIdentityJwtToken)
	defer ts.Close()

	// Get a time stamp and use it through out the test.
	fakeNow := time.Now()

	mf := NewMockMetadataFetcher(ts.GetURL(), fakeNow)

	testData := []testToken{
		{
			desc:               "Empty token response",
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
		{
			desc:               "token has expired in token map",
			curToken:           "ya29.expired",
			curTokenTimeout:    fakeNow.Add(-1 * time.Hour),
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
		{
			desc:               "token is not expired in token map",
			curToken:           "ya29.nonexpired",
			curTokenTimeout:    fakeNow.Add(61 * time.Second),
			expectedToken:      "ya29.nonexpired",
			expectedExpiration: 61 * time.Second,
		},
		{
			desc:               "token valid time is below 60 seconds in token map",
			curToken:           "ya29.nonexpired",
			curTokenTimeout:    fakeNow.Add(59 * time.Second),
			expectedToken:      "ya29.new",
			expectedExpiration: 3599 * time.Second,
		},
	}

	fakeAudience := "audience"
	for i, tc := range testData {
		// Mocking the last-fetched token.
		if tc.curToken != "" {
			mf.audToToken.Store(fakeAudience,
				tokenInfo{
					accessToken:  tc.curToken,
					tokenTimeout: tc.curTokenTimeout,
				},
			)
		}

		token, expires, err := mf.FetchIdentityJWTToken(fakeAudience)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(token, tc.expectedToken) {
			t.Errorf("Test Desc(%d): %s, token got: %v, want: %s", i, tc.desc, token, tc.expectedToken)
		}
		if expires != tc.expectedExpiration {
			t.Errorf("Test Desc(%d): %s, expiration got: %s, want: %s", i, tc.desc, expires, tc.expectedExpiration)
		}
	}
}

func TestFetchServiceName(t *testing.T) {
	ts := util.InitMockServer(fakeServiceName)
	defer ts.Close()

	mf := NewMockMetadataFetcher(ts.GetURL(), time.Now())

	name, err := mf.FetchServiceName()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(name, fakeServiceName) {
		t.Errorf("fetchServiceName = %s, want %s", name, fakeServiceName)
	}
}

func TestFetchConfigId(t *testing.T) {
	ts := util.InitMockServer(fakeConfigID)
	defer ts.Close()

	mf := NewMockMetadataFetcher(ts.GetURL(), time.Now())

	name, err := mf.FetchConfigId()
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
				util.ProjectIDSuffix: fakeProjectID,
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				ProjectId: fakeProjectID,
				Platform:  util.GCE,
			},
		},
		{
			desc: "Valid Zone",
			mockedResp: map[string]string{
				util.ZoneSuffix: fakeZonePath,
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Zone:     fakeZone,
				Platform: util.GCE,
			},
		},
		{
			desc: "Invalid Zone - without '/'",
			mockedResp: map[string]string{
				util.ZoneSuffix: "noslash",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GCE,
			},
		},
		{
			desc: "Invalid Zone - ends with '/'",
			mockedResp: map[string]string{
				util.ZoneSuffix: "project/123123/",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GCE,
			},
		},
		{
			desc: "Platform - GAE_FLEX",
			mockedResp: map[string]string{
				util.GAEServerSoftwareSuffix: "foo",
			},
			expectedGCPAttributes: &scpb.GcpAttributes{
				Platform: util.GAEFlex,
			},
		},
		{
			desc: "Platform - GKE",
			mockedResp: map[string]string{
				util.KubeEnvSuffix: "foo",
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
				util.ProjectIDSuffix:         fakeProjectID,
				util.ZoneSuffix:              fakeZonePath,
				util.GAEServerSoftwareSuffix: "foo",
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

	errorTmpl := "Test: %s\n  Expected: %v\n  Actual: %v"
	for _, tc := range testData {
		ts := util.InitMockServerFromPathResp(tc.mockedResp)
		defer ts.Close()

		mockBaseUrl := ts.URL
		if tc.mockedResp == nil {
			mockBaseUrl = "non-existing-url-287924837"
		}

		mf := NewMockMetadataFetcher(mockBaseUrl, time.Now())

		attrs, err := mf.FetchGCPAttributes()
		if err != nil && tc.expectedGCPAttributes != nil {
			t.Errorf(errorTmpl, tc.desc, nil, err)
			continue
		}
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
