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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func initServiceControlForTestNewRolloutId(t *testing.T, serviceRolloutId *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := genFakeReport(*serviceRolloutId)
		if err != nil {
			t.Fatalf("fail to generate servicecontrol report response: %v", err)
		}
		_, _ = w.Write(resp)
	}))
}

func genFakeReport(serviceRolloutId string) ([]byte, error) {
	reportResp := new(scpb.ReportResponse)
	reportResp.ServiceRolloutId = serviceRolloutId
	return proto.Marshal(reportResp)
}

func TestFetchLatestRolloutId(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := initServiceControlForTestNewRolloutId(t, &serviceRolloutId)
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }

	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.URL, "service-name", accessToken)

	// Test success of fetching the latest rolloutId.
	rolloutId, _ := cif.fetchLatestRolloutId()
	if rolloutId != serviceRolloutId {
		t.Errorf("fail in fetchLatestRolloutId, want rolloutId %s, get rolloutId %s", rolloutId, serviceRolloutId)
	}

	// Fail due to calling googleapis.
	util.CallGooglelapis = func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, output proto.Message) error {
		return fmt.Errorf("error-from-CallGoogleapis")
	}
	_, err := cif.fetchLatestRolloutId()
	wantError := "fail to fetch new rollout id, error-from-CallGoogleapis"
	if err == nil || err.Error() != wantError {
		t.Errorf("fail in fetchLatestRolloutId, want error %v, get error %s", err, wantError)
	}
}

func TestRolloutIdChangeFetcherSetDetectRolloutIdChangeTimer(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := initServiceControlForTestNewRolloutId(t, &serviceRolloutId)
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }
	cif := NewRolloutIdChangeDetector(&http.Client{}, serviceControlServer.URL, "service-name", accessToken)

	cnt := 0
	wantCnt := 3
	wantRolloutId := fmt.Sprintf("test-rollout-id-%v", wantCnt)
	cif.SetDetectRolloutIdChangeTimer(time.Millisecond*50, func() {
		cnt += 1
		// Update rolloutId so the callback will be called.
		// It will be updated only three times.
		if cnt < wantCnt {
			serviceRolloutId = fmt.Sprintf("test-rollout-id-%v", cnt+1)
		}
	})

	time.Sleep(time.Millisecond * 300)
	if cnt != wantCnt {
		t.Fatalf("want callback called by 1 times, get %v times", cnt)
	}

	if cif.curRolloutId != wantRolloutId {
		t.Errorf("want curRolloutId: %s, get curRolloutId: %s", wantRolloutId, cif.curRolloutId)
	}
}
