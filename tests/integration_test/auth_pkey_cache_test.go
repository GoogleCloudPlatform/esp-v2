// Copyright 2019 Google LLC
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

package integration_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestAuthJwksCache(t *testing.T) {
	t.Parallel()

	provider := testdata.GoogleJwtProvider
	type expectedRequestCount struct {
		key string
		cnt int
	}
	testData := []struct {
		desc                   string
		path                   string
		method                 string
		token                  string
		apiKey                 string
		jwksCacheDurationInS   int
		wantRequestsToProvider *expectedRequestCount
		wantResp               string
	}{
		{
			desc:                   "Success, the default jwks cache duration is 300s so only 1 request to the jwks provider will be made",
			path:                   "/auth/info/auth0",
			apiKey:                 "api-key",
			method:                 "GET",
			token:                  testdata.FakeCloudTokenMultiAudiences,
			wantRequestsToProvider: &expectedRequestCount{provider, 1},
			wantResp:               `{"aud":["admin.cloud.goog","bookstore_test_client.cloud.goog"],"exp":4698318999,"iat":1544718999,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
		{
			desc:                   "Success, the customized jwks cache duration is 1s so 10 request to the jwks provider will be made",
			path:                   "/auth/info/auth0",
			apiKey:                 "api-key",
			method:                 "GET",
			jwksCacheDurationInS:   1,
			token:                  testdata.FakeCloudTokenMultiAudiences,
			wantRequestsToProvider: &expectedRequestCount{provider, 5},
			wantResp:               `{"aud":["admin.cloud.goog","bookstore_test_client.cloud.goog"],"exp":4698318999,"iat":1544718999,"iss":"api-proxy-testing@cloud.goog","sub":"api-proxy-testing@cloud.goog"}`,
		},
	}
	for _, tc := range testData {
		func() {
			args := utils.CommonArgs()

			s := env.NewTestEnv(comp.TestAuthJwksCache, platform.EchoSidecar)
			if tc.jwksCacheDurationInS != 0 {
				args = append(args, fmt.Sprintf("--jwks_cache_duration_in_s=%v", tc.jwksCacheDurationInS))
			}

			defer s.TearDown()
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}
			s.FakeJwtService.ResetReqCnt(provider)

			var resp []byte
			for i := 0; i < 5; i++ {
				resp, _ = client.DoJWT(fmt.Sprintf("http://localhost:%v", s.Ports().ListenerPort), tc.method, tc.path, tc.apiKey, "", tc.token)
				// Sleep as long as the customized cache duration to make caches expires
				if tc.jwksCacheDurationInS != 0 {
					time.Sleep(time.Duration(tc.jwksCacheDurationInS) * time.Second)
				}
			}

			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed\nexpected: %s\ngot: %s", tc.desc, tc.wantResp, string(resp))
			}

			if tc.wantRequestsToProvider != nil {
				provider, ok := s.FakeJwtService.ProviderMap[tc.wantRequestsToProvider.key]
				if !ok {
					t.Errorf("Test (%s): failed, the provider is not inited.", tc.desc)
				} else if realCnt := provider.GetReqCnt(); realCnt != tc.wantRequestsToProvider.cnt {
					t.Errorf("Test (%s): failed, pubkey of %s shoud be fetched %v times instead of %v times.", tc.desc, tc.wantRequestsToProvider.key, tc.wantRequestsToProvider.cnt, realCnt)
				}
			}
		}()
	}
}
