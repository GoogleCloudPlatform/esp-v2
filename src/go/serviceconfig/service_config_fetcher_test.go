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

package serviceconfig

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func TestServiceConfigFetcherTimeout(t *testing.T) {
	timeout := 1 * time.Second

	serviceConfigFetcher := ServiceConfigFetcher{
		client: http.Client{
			Timeout: timeout,
		},
	}

	server := util.InitMockServer(`{}`)
	_, err := serviceConfigFetcher.callWithAccessToken(server.GetURL(), "this-is-token")
	if err != nil {
		t.Errorf("TestServiceConfigFetcherTimeout: the service config fetcher should get the config but get the error %v", err)

	}
	server.SetSleepTime(2 * timeout)
	_, err = serviceConfigFetcher.callWithAccessToken(server.GetURL(), "this-is-token")
	if err == nil || !strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
		t.Errorf("TestServiceConfigFetcherTimeout: the service config fetcher get the config but should get timeout error")

	}
}
