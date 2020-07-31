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

package tokengenerator

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestGenerateAccessToken(t *testing.T) {

	fakeToken := `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	mockTokenServer := util.InitMockServer(fakeToken)
	defer mockTokenServer.Close()

	fakeKey := strings.Replace(testdata.FakeServiceAccountKeyData, "FAKE-TOKEN-URI", mockTokenServer.GetURL(), 1)
	fakeKeyData := []byte(fakeKey)

	token, duration, err := generateAccessTokenFromData(fakeKeyData)
	if token != "ya29.new" || duration.Seconds() < 3598 || err != nil {
		t.Errorf("Test : Fail to make access token, got token: %s, duration: %v, err: %v", token, duration, err)
	}

	latestFakeToken := `{"access_token": "ya29.latest", "expires_in":3599, "token_type":"Bearer"}`
	mockTokenServer.SetResp(latestFakeToken)

	// The token is cached so the old token gets returned.
	token, duration, err = generateAccessTokenFromData([]byte("Invalid data, not a service account"))
	if token != "ya29.new" || err != nil {
		t.Errorf("Test : Fail to make access token, got token: %s, duration: %v, err: %v", token, duration, err)
	}
}

func TestMakeLatsHandler(t *testing.T) {

	s := httptest.NewServer(MakeLatsHandler(platform.GetFilePath(platform.FakeServiceAccountFile)))

	testCases := []struct {
		desc                   string
		path                   string
		genAccessTokenFromFile func(saFilePath string) (string, time.Duration, error)
		method                 string
		wantResp               string
		wantError              string
	}{
		{
			desc: "success, get access token",
			genAccessTokenFromFile: func(saFilePath string) (string, time.Duration, error) {
				return "ya29.new", time.Duration(time.Second * 100), nil
			},
			path:     "/v1/instance/service-accounts/default/token",
			method:   "GET",
			wantResp: `{"access_token": "ya29.new", "expires_in": 100}`,
		},
		{
			desc: "fail, error in generating access token",
			genAccessTokenFromFile: func(saFilePath string) (string, time.Duration, error) {
				return "", 0, fmt.Errorf("gen-access-token-error")
			},
			path:      "/v1/instance/service-accounts/default/token",
			method:    "GET",
			wantError: "500 Internal Server Error, gen-access-token-error",
		},
		{
			desc: "fail, wrong path",
			genAccessTokenFromFile: func(saFilePath string) (string, time.Duration, error) {
				return "ya29.new", time.Duration(time.Second * 100), nil
			},
			path:      "/wrong-path",
			method:    "GET",
			wantError: "404 Not Found",
		},
	}

	for _, tc := range testCases {
		GenerateAccessTokenFromFile = tc.genAccessTokenFromFile
		_, resp, err := utils.DoWithHeaders(s.URL+tc.path, "GET", "", nil)
		if tc.wantError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("test(%s): get error: %v, want error: %s", tc.desc, err, tc.wantError)
			}
		}

		if tc.wantResp != "" && tc.wantResp != string(resp) {
			t.Errorf("test(%s): get resp: %s, want resp %s", tc.desc, string(resp), tc.wantResp)

		}

	}
}
