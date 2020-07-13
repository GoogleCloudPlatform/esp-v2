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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func genFakeReport(serviceRolloutId string) string {
	reportResp := new(scpb.ReportResponse)
	reportResp.ServiceRolloutId = serviceRolloutId
	reportRespBytes, _ := proto.Marshal(reportResp)
	return string(reportRespBytes)
}

type getCallGoogleapisFunc func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, output proto.Message) error

func TestFetchLatestRolloutId(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := util.InitMockServer(genFakeReport(serviceRolloutId))
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }

	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.GetURL(), "service-name", accessToken)

	callGoogleapis := util.CallGoogleapis

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
			callGoogleapis: func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, output proto.Message) error {
				return fmt.Errorf("error-from-CallGoogleapis")
			},
			wantError: "fail to fetch new rollout id, error-from-CallGoogleapis",
		},
	}
	for _, tc := range testCases {
		util.CallGoogleapis = tc.callGoogleapis
		rolloutId, err := cif.fetchLatestRolloutId()
		if tc.wantRolloutId != "" && tc.wantRolloutId != rolloutId {
			t.Errorf("Test(%s): fail in fetchLatestRolloutId, want rolloutId %s, get rolloutId %s", tc.desc, tc.wantRolloutId, rolloutId)
		}

		if tc.wantError != "" {
			if err == nil {
				t.Errorf("Test(%s): fail in fetchLatestRolloutId, want error %v, get error %s", tc.desc, tc.wantError, err)
			} else if err.Error() != tc.wantError {
				t.Errorf("Test(%s): fail in fetchLatestRolloutId, want error %v, get error %s", tc.desc, tc.wantError, err)
			}
		}
	}

	// Recover util.CallGoogleapis.
	util.CallGoogleapis = callGoogleapis
}

func TestRolloutIdChangeFetcherSetDetectRolloutIdChangeTimer(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := util.InitMockServer(genFakeReport(serviceRolloutId))
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }
	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.GetURL(), "service-name", accessToken)

	cnt := 0
	wantCnt := 3
	wantRolloutId := fmt.Sprintf("test-rollout-id-%v", wantCnt)
	cif.SetDetectRolloutIdChangeTimer(time.Millisecond*50, func() {
		cnt += 1
		// Update rolloutId so the callback will be called.
		// It will be updated only three times.
		if cnt < wantCnt {
			serviceRolloutId = fmt.Sprintf("test-rollout-id-%v", cnt+1)
			serviceControlServer.SetResp(genFakeReport(serviceRolloutId))
		}
	})

	// Sleep long enough to make sure the callback is called 3 times so that `cnt`
	// won't be updated in callback since no update on rolloutId. Otherwise, it
	// will cause data race on `cnt`.
	time.Sleep(time.Millisecond * 1000)

	if cnt != wantCnt {
		t.Fatalf("want callback called by %v times, get %v times", wantCnt, cnt)
	}

	if cif.curRolloutId != wantRolloutId {
		t.Errorf("want curRolloutId: %s, get curRolloutId: %s", wantRolloutId, cif.curRolloutId)
	}
}
