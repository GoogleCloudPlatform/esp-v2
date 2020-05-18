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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/testdata"
)

func TestGenerateAccessToken(t *testing.T) {

	fakeToken := `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	mockTokenServer := InitMockServer(fakeToken)
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
