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

package auth_pkey_cache_test

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
)

const (
	wantLongToken = `{"aud":["admin.cloud.goog","bookstore_test_client.cloud.goog"],"exp":4832605522,"iat":1679005522,"iss":"api-proxy-testing@cloud.goog","scope":"xyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxyz","sub":"api-proxy-testing@cloud.goog"}`
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
		jwksFetchNumRetries    int
		wantRequestsToProvider *expectedRequestCount
		wantResp               string
	}{
		{
			desc:                   "Success, the default jwks cache duration is 300s so only 1 request to the jwks provider will be made",
			path:                   "/auth/info/auth0",
			apiKey:                 "api-key",
			method:                 "GET",
			token:                  testdata.FakeCloudTokenLongClaims,
			wantRequestsToProvider: &expectedRequestCount{provider, 1},
			wantResp:               wantLongToken,
		},
		{
			desc:                   "Success, the customized jwks cache duration is 1s so 5 request to the jwks provider will be made",
			path:                   "/auth/info/auth0",
			apiKey:                 "api-key",
			method:                 "GET",
			jwksCacheDurationInS:   1,
			token:                  testdata.FakeCloudTokenLongClaims,
			wantRequestsToProvider: &expectedRequestCount{provider, 5},
			wantResp:               wantLongToken,
		},
		{
			desc:                   "Success, the customized jwks succeeds once every five requests so 5 request to the jwks provider will be made",
			path:                   "/auth/info/auth0",
			apiKey:                 "api-key",
			method:                 "GET",
			jwksFetchNumRetries:    5,
			token:                  testdata.FakeCloudTokenLongClaims,
			wantRequestsToProvider: &expectedRequestCount{provider, 5},
			wantResp:               wantLongToken,
		},
	}
	for _, tc := range testData {
		func() {
			args := utils.CommonArgs()

			s := env.NewTestEnv(platform.TestAuthJwksCache, platform.EchoSidecar)
			args = append(args, "--disable_jwks_async_fetch")
			// Need to disable jwt cache as the test is counting of number of jwks fetching.
			args = append(args, "--jwt_cache_size=0")
			if tc.jwksCacheDurationInS != 0 {
				args = append(args, fmt.Sprintf("--jwks_cache_duration_in_s=%v", tc.jwksCacheDurationInS))
			}
			if tc.jwksFetchNumRetries != 0 {
				args = append(args, fmt.Sprintf("--jwks_fetch_num_retries=%v", tc.jwksFetchNumRetries))
				for _, config := range testdata.ProviderConfigs {
					if config.Id == provider {
						// alter hard-coded provider's configuration just for this test
						// all requests will only succeed on the last retry.
						config.IsPeriodicallyFailing = true
						config.SuccessPeriod = tc.jwksFetchNumRetries
					}
				}
			} else {
				for _, config := range testdata.ProviderConfigs {
					if config.Id == provider {
						// restore hard-coded provider's configuration do default
						config.IsPeriodicallyFailing = false
						config.SuccessPeriod = 0
					}
				}
			}

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			// Verify jwks fetching count is 0 for on-demand fetching
			if tc.wantRequestsToProvider != nil {
				provider, ok := s.FakeJwtService.ProviderMap[tc.wantRequestsToProvider.key]
				if !ok {
					t.Errorf("Test (%s): failed, the provider is not inited.", tc.desc)
				} else if realCnt := provider.GetReqCnt(); realCnt != 0 {
					t.Errorf("Test (%s): failed, pubkey of %s shoud be fetched 0 times instead of %v times.", tc.desc, tc.wantRequestsToProvider.key, realCnt)
				}
			}

			var resp []byte
			for i := 0; i < 5; i++ {
				resp, _ = client.DoJWT(fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort), tc.method, tc.path, tc.apiKey, "", tc.token)
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

func TestAuthJwksAsyncFetch(t *testing.T) {
	t.Parallel()

	provider := testdata.GoogleJwtProvider
	type expectedRequestCount struct {
		key    string
		minCnt int
	}
	testData := []struct {
		desc                 string
		path                 string
		method               string
		token                string
		apiKey               string
		jwksCacheDurationInS int
		jwksFetchNumRetries  int
		initProviderCnt      *expectedRequestCount
		sleep                int
		afterProviderCnt     *expectedRequestCount
		wantResp             string
	}{
		{
			desc:   "Success, the default jwks cache duration is 300s so only 1 request to the jwks provider will be made",
			path:   "/auth/info/auth0",
			apiKey: "api-key",
			method: "GET",
			token:  testdata.FakeCloudTokenLongClaims,
			// one fetch should happen at init
			initProviderCnt: &expectedRequestCount{provider, 1},
			// default cache expiration is 5 minutes, still 1 fetch.
			afterProviderCnt: &expectedRequestCount{provider, 1},
			wantResp:         wantLongToken,
		},
		{
			desc:                 "Success, the customized jwks cache duration is 1s so at least 5 request to the jwks provider will be made",
			path:                 "/auth/info/auth0",
			apiKey:               "api-key",
			method:               "GET",
			jwksCacheDurationInS: 1,
			token:                testdata.FakeCloudTokenLongClaims,
			// one fetch should happen at init
			initProviderCnt: &expectedRequestCount{provider, 1},
			// Only sleep 5 seconds, so at least 5 fetch since cache expiration is 1s
			afterProviderCnt: &expectedRequestCount{provider, 5},
			wantResp:         wantLongToken,
		},
		{
			desc:                "Success, the jwks provider succeeds only once every 5 requests so at least 5 request to the jwks provider will be made",
			path:                "/auth/info/auth0",
			apiKey:              "api-key",
			method:              "GET",
			jwksFetchNumRetries: 5,
			token:               testdata.FakeCloudTokenLongClaims,
			// minimum number of requests ( successful or otherwise ) done after envoy is started, but before a valid, JWT authorized, service request is issued )
			// one successful fetch at init
			initProviderCnt: &expectedRequestCount{provider, 5},
			// minimum number of requests ( successful or otherwise ) done after a JWT authorized service request succeeded )
			// one successful fetch. so 5 minimum since using a provider that will fail 4 times before succeeding once.
			afterProviderCnt: &expectedRequestCount{provider, 5},
			wantResp:         wantLongToken,
		},
	}
	for _, tc := range testData {
		func() {
			args := utils.CommonArgs()

			s := env.NewTestEnv(platform.TestAuthJwksAsyncFetch, platform.EchoSidecar)
			if tc.jwksCacheDurationInS != 0 {
				args = append(args, fmt.Sprintf("--jwks_cache_duration_in_s=%v", tc.jwksCacheDurationInS))
			}
			if tc.jwksFetchNumRetries != 0 {
				args = append(args, fmt.Sprintf("--jwks_fetch_num_retries=%v", tc.jwksFetchNumRetries))
				for _, config := range testdata.ProviderConfigs {
					if config.Id == provider {
						// alter hard-coded provider's configuration just for this test
						// all requests will only succeed on the last retry.
						config.IsPeriodicallyFailing = true
						config.SuccessPeriod = tc.jwksFetchNumRetries
					}
				}
			} else {
				for _, config := range testdata.ProviderConfigs {
					if config.Id == provider {
						// restore hard-coded provider's configuration do default
						config.IsPeriodicallyFailing = false
						config.SuccessPeriod = 0
					}
				}
			}

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			verifyProviderCnt := func(expected *expectedRequestCount) {
				provider, ok := s.FakeJwtService.ProviderMap[expected.key]
				if !ok {
					t.Errorf("Test (%s): failed, the provider is not inited.", tc.desc)
				} else if realCnt := provider.GetReqCnt(); realCnt < expected.minCnt {
					t.Errorf("Test (%s): failed, pubkey of %s shoud be fetched at least %v times, but got %v times.", tc.desc, expected.key, expected.minCnt, realCnt)
				}
			}
			verifyProviderCnt(tc.initProviderCnt)

			var resp []byte
			resp, _ = client.DoJWT(fmt.Sprintf("http://%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort), tc.method, tc.path, tc.apiKey, "", tc.token)
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed\nexpected: %s\ngot: %s", tc.desc, tc.wantResp, string(resp))
			}

			// sleep 5 seconds
			time.Sleep(time.Duration(5) * time.Second)

			verifyProviderCnt(tc.afterProviderCnt)
		}()
	}
}
