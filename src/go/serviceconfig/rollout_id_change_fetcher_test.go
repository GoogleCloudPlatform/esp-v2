// Copyright 2020 Google LLC
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
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"google.golang.org/protobuf/proto"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func genFakeReport(serviceRolloutId string) string {
	reportResp := new(scpb.ReportResponse)
	reportResp.ServiceRolloutId = serviceRolloutId
	reportRespBytes, _ := proto.Marshal(reportResp)
	return string(reportRespBytes)
}

type getCallGoogleapisFunc func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, retryConfigs map[int]util.RetryConfig, output proto.Message) error

func TestFetchLatestRolloutId(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := util.InitMockServer(genFakeReport(serviceRolloutId))
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }

	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.GetURL(), "service-name", accessToken)
	util.CallGoogleapisMu.RLock()
	callGoogleapis := util.CallGoogleapis
	util.CallGoogleapisMu.RUnlock()
	testCases := []struct {
		desc           string
		callGoogleapis getCallGoogleapisFunc
		wantRolloutId  string
		wantError      string
	}{
		{
			desc:           "success of fetching the latest rolloutId",
			callGoogleapis: callGoogleapis,
			wantRolloutId:  serviceRolloutId,
		},
		{
			desc: "failure due to call googleapis",
			callGoogleapis: func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, retryConfigs map[int]util.RetryConfig, output proto.Message) error {
				return fmt.Errorf("error-from-CallGoogleapis")
			},
			wantError: "fail to fetch new rollout id, error-from-CallGoogleapis",
		},
	}
	for _, tc := range testCases {
		util.CallGoogleapisMu.RLock()
		util.CallGoogleapis = tc.callGoogleapis
		util.CallGoogleapisMu.RUnlock()
		rolloutId, err := cif.fetchLatestRolloutId()
		if tc.wantRolloutId != "" && tc.wantRolloutId != rolloutId {
			t.Errorf("Test(%s): fail in fetchLatestRolloutId, want rolloutId %s, get rolloutId %s", tc.desc, tc.wantRolloutId, rolloutId)
		}

		if tc.wantError != "" {
			if err == nil || err.Error() != tc.wantError {
				t.Errorf("Test(%s): fail in fetchLatestRolloutId, want error %v, get error %s", tc.desc, tc.wantError, err)
			}
		}
	}

	// Recover util.CallGoogleapis.
	util.CallGoogleapisMu.RLock()
	util.CallGoogleapis = callGoogleapis
	util.CallGoogleapisMu.RUnlock()
}

func TestSetDetectRolloutIdChangeTimer(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := util.InitMockServer(genFakeReport(serviceRolloutId))
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }
	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.GetURL(), "service-name", accessToken)

	var cnt, wantCnt int32
	cnt = 0
	wantCnt = 3

	wantRolloutId := fmt.Sprintf("test-rollout-id-%v", wantCnt)
	cif.SetDetectRolloutIdChangeTimer(time.Millisecond*50, func() {
		atomic.AddInt32(&cnt, 1)

		// Update rolloutId so the callback will be called.
		// It will be updated only three times.
		if cnt < wantCnt {
			serviceRolloutId = fmt.Sprintf("test-rollout-id-%v", atomic.LoadInt32(&cnt)+1)
			serviceControlServer.SetResp(genFakeReport(serviceRolloutId))
		}
	})

	// Sleep long enough to make sure the callback is called 3 times.
	time.Sleep(time.Millisecond * 1000)

	if atomic.LoadInt32(&cnt) != wantCnt {
		t.Fatalf("want callback called by %v times, get %v times", wantCnt, cnt)
	}

	if cif.curRolloutId != wantRolloutId {
		t.Errorf("want curRolloutId: %s, get curRolloutId: %s", wantRolloutId, cif.curRolloutId)
	}
}
