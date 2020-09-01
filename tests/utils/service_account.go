// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/testdata"
)

type ServiceAccountForTest struct {
	MockTokenServer *util.MockServer
	FileName        string
}

func NewServiceAccountForTest() (*ServiceAccountForTest, error) {
	// Setup token server which will be queried by config manager.
	fakeToken := `{"access_token": "this-is-sa_gen_token", "expires_in":3599, "token_type":"Bearer"}`
	mockTokenServer := util.InitMockServer(fakeToken)

	// Update service account file template with server URI.
	fakeKey := strings.Replace(testdata.FakeServiceAccountKeyData, "FAKE-TOKEN-URI", mockTokenServer.GetURL(), 1)
	serviceAccountFile, err := ioutil.TempFile(os.TempDir(), "sa-cred-")
	if err != nil {
		return nil, fmt.Errorf("fail to create a temp service account file")
	}

	// Write actual service account file and return file name.
	_ = ioutil.WriteFile(serviceAccountFile.Name(), []byte(fakeKey), 0644)
	return &ServiceAccountForTest{
		MockTokenServer: mockTokenServer,
		FileName:        serviceAccountFile.Name(),
	}, nil
}
