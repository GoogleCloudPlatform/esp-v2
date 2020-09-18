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
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
)

type ConfiguredDeadline int

const (
	Short ConfiguredDeadline = iota
	Default
)

// Tests the deadlines configured in backend rules for a HTTP/1.x backend (no streaming).
func TestDeadlinesForDynamicRouting(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestDeadlinesForDynamicRouting, platform.EchoRemote)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		reqDuration    time.Duration
		deadlineToTest ConfiguredDeadline
		wantErr        string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc:           "Success after 10s due to ESPv2 default response timeout being 15s",
			reqDuration:    time.Second * 10,
			deadlineToTest: Default,
		},
		{
			desc:           "Fail before 20s due to ESPv2 default response timeout being 15s",
			reqDuration:    time.Second * 20,
			deadlineToTest: Default,
			wantErr:        `504 Gateway Timeout, {"code":504,"message":"upstream request timeout"}`,
		},
		{
			desc:           "Success after 2s due to user-configured deadline being 5s",
			reqDuration:    time.Second * 2,
			deadlineToTest: Short,
		},
		{
			desc:           "Fail before 8s due to user-configured deadline being 5s",
			reqDuration:    time.Second * 8,
			deadlineToTest: Short,
			wantErr:        `504 Gateway Timeout, {"code":504,"message":"upstream request timeout"}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow efficient measuring of elapsed time.
		// Elapsed time is not checked in the test, it's just for debugging.
		func() {
			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

			// Decide which path to call based on which configured deadline to test against.
			var basePath string
			switch tc.deadlineToTest {
			case Default:
				basePath = "/sleepDefault"
			case Short:
				basePath = "/sleepShort"
			}

			path := fmt.Sprintf("%v?duration=%v", basePath, tc.reqDuration.String())
			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, path)

			_, err := client.DoWithHeaders(url, "GET", "", nil)

			if tc.wantErr == "" && err != nil {
				t.Errorf("Test (%s): failed, expected no err, got err (%v)", tc.desc, err)
			}

			if tc.wantErr != "" && err == nil {
				t.Errorf("Test (%s): failed, got no err, expected err (%v)", tc.desc, tc.wantErr)
			}

			if err != nil && !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Test (%s): failed, got err (%v), expected err (%v)", tc.desc, err, tc.wantErr)
			}

		}()
	}
}

// Tests the default deadline in the catch-all HTTP/1.x backend (no streaming).
func TestDeadlinesForCatchAllBackend(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(platform.TestDeadlinesForCatchAllBackend, platform.EchoSidecar)

	defer s.TearDown(t)
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc        string
		reqDuration time.Duration
		wantErr     string
	}{
		// Please be cautious about adding too many time-based tests here.
		// This can slow down our CI system if we sleep for too long.
		{
			desc:        "Success after 10s due to ESPv2 default response timeout being 15s",
			reqDuration: time.Second * 10,
		},
		{
			desc:        "Fail before 20s due to ESPv2 default response timeout being 15s",
			reqDuration: time.Second * 20,
			wantErr:     `504 Gateway Timeout, {"code":504,"message":"upstream request timeout"}`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow efficient measuring of elapsed time.
		// Elapsed time is not checked in the test, it's just for debugging.
		func() {
			defer utils.Elapsed(fmt.Sprintf("Test (%s):", tc.desc))()

			path := fmt.Sprintf("/sleep?duration=%v", tc.reqDuration.String())
			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, path)

			_, err := client.DoWithHeaders(url, "GET", "", nil)

			if tc.wantErr == "" && err != nil {
				t.Errorf("Test (%s): failed, expected no err, got err (%v)", tc.desc, err)
			}

			if tc.wantErr != "" && err == nil {
				t.Errorf("Test (%s): failed, got no err, expected err (%v)", tc.desc, tc.wantErr)
			}

			if err != nil && !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Test (%s): failed, got err (%v), expected err (%v)", tc.desc, err, tc.wantErr)
			}

		}()
	}
}
