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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

func TestMetadataRequestsPerPlatform(t *testing.T) {
	t.Parallel()

	customSa, err := utils.NewServiceAccountForTest()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	defer customSa.MockTokenServer.Close()

	testData := []struct {
		desc                     string
		path                     string
		method                   string
		key                      string
		confArgs                 []string
		wantRequestsToMetaServer map[string]int
	}{
		{
			desc:     "For GCP deployment, IMDS provides access token and location.",
			path:     "/echo",
			method:   "POST",
			key:      "api-key",
			confArgs: utils.CommonArgs(),
			wantRequestsToMetaServer: map[string]int{
				util.AccessTokenPath: 2,
				util.ProjectIDPath:   1,
				util.ZonePath:        1,
			},
		},
		{
			desc:   "For GCP deployment with service account, IMDS only provides location (not access token).",
			path:   "/echo",
			method: "POST",
			key:    "api-key",
			confArgs: append([]string{
				"--service_account_key=" + customSa.FileName,
			}, utils.CommonArgs()...),
			wantRequestsToMetaServer: map[string]int{
				util.AccessTokenPath: 0,
				util.ProjectIDPath:   1,
				util.ZonePath:        1,
			},
		},
		{
			desc:   "For non-GCP deployment, IMDS is never called.",
			path:   "/echo",
			method: "POST",
			key:    "api-key",
			confArgs: append([]string{
				"--non_gcp",
				"--service_account_key=" + customSa.FileName,
			}, utils.CommonArgs()...),
			wantRequestsToMetaServer: map[string]int{
				util.AccessTokenPath: 0,
				util.ProjectIDPath:   0,
				util.ZonePath:        0,
			},
		},
	}
	for _, tc := range testData {

		// Place in closure to allow deferring in loop.
		func() {
			s := env.NewTestEnv(comp.TestMetadataRequestsPerPlatform, platform.EchoSidecar)
			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v?key=%v", s.Ports().ListenerPort, tc.path, tc.key)
			_, err := client.DoWithHeaders(url, tc.method, "message", nil)
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}

			expectNoRequests := true
			for path, wantCount := range tc.wantRequestsToMetaServer {
				if gotCnt := s.MockMetadataServer.GetReqCnt(path); gotCnt != wantCount {
					t.Errorf("Test (%v): failed, path(%v) on MetadataServer, got requests: %v, want requests: %v", tc.desc, path, gotCnt, wantCount)
				}

				if wantCount != 0 {
					expectNoRequests = false
				}
			}

			if expectNoRequests {
				// For the case where we expect no requests to IMDS, validate across all paths.
				// Don't do this for all tests, it gets too messy.
				gotTotalReq := s.MockMetadataServer.GetTotalReqCnt()
				if gotTotalReq != 0 {
					t.Errorf("Test (%v) failed: IMDS received %v total requests, want 0", tc.desc, gotTotalReq)
				}
			}
		}()

	}
}
