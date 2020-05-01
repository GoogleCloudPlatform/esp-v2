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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

// Tests the deadlines configured in backend rules for a HTTP/1.x backend (no streaming).
func TestStatistics(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(comp.TestStatistics, platform.EchoRemote)

	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	const bufferPercent = 0.5

	testData := []struct {
		desc           string
		reqCnt         int
		reqDuration    time.Duration
		wantCounts     map[string]int
		wantHistograms map[string][]float64
	}{

		{
			desc:        "Success after 10s due to ESPv2 default response timeout being 15s",
			reqDuration: time.Second * 1,
			wantCounts: map[string]int{
				"http.ingress_http.backend_auth.token_added":                       1,
				"http.ingress_http.backend_routing.append_path_to_address_request": 1,
				"http.ingress_http.path_matcher.allowed":                           1,
				"http.ingress_http.service_control.allowed":                        1,
			},
			wantHistograms: map[string][]float64{
				// For overhead_time, only verify the maximum(0 will be ignored).
				"http.ingress_http.service_control.overhead_time": {0, 0, 0, 0, 0, 0, 0, 0, 0, 5},
				"http.ingress_http.service_control.backend_time":  {1000, 1025, 1050, 1075, 1090, 1095, 1099, 1099.5, 1099.9, 1100},
				"http.ingress_http.service_control.request_time":  {1000, 1025, 1050, 1075, 1090, 1095, 1099, 1099.5, 1099.9, 1100},
			},
		},
		{
			desc:        "Success after 10s due to ESPv2 default response timeout being 15s",
			reqDuration: time.Second * 2,
			wantCounts: map[string]int{
				"http.ingress_http.backend_auth.token_added":                       2,
				"http.ingress_http.backend_routing.append_path_to_address_request": 2,
				"http.ingress_http.path_matcher.allowed":                           2,
				"http.ingress_http.service_control.allowed":                        2,
			},
			wantHistograms: map[string][]float64{
				// For overhead_time, only verify the maximum(0 will be ignored).
				"http.ingress_http.service_control.overhead_time": {0, 0, 0, 0, 0, 0, 0, 0, 0, 5},
				"http.ingress_http.service_control.backend_time":  {1000, 1050, 1100, 2050, 2080, 2090, 2098, 2099, 2099.8, 2100},
				"http.ingress_http.service_control.request_time":  {1000, 1050, 1100, 2050, 2080, 2090, 2098, 2099, 2099.8, 2100},
			},
		},
	}

	for _, tc := range testData {
		path := fmt.Sprintf("/sleepDefault?duration=%v", tc.reqDuration.String())
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, path)

		_, err := client.DoWithHeaders(url, "GET", "", nil)

		if err != nil {
			t.Fatalf("Test (%s): failed, expected no err, got err (%v)", tc.desc, err)
		}

		// Ensure the stats is available in admin.
		time.Sleep(time.Millisecond * 5000)

		statsUrl := fmt.Sprintf("http://localhost:%v%v", s.Ports().AdminPort, utils.GetStatsPath())
		resp, err := utils.DoWithHeaders(statsUrl, "GET", "", nil)
		if err != nil {
			t.Fatalf("GET %s faild: %v", url, err)
		}

		counts, histograms, err := utils.ParseStats(resp)
		if err != nil {
			t.Fatalf("fail to parse stats: %v", err)
		}

		for wantCountName, wantCountVal := range tc.wantCounts {
			if getCountVal, ok := counts[wantCountName]; !ok {
				t.Errorf("Test (%s): failed, expected counter %v not in the got counters: %v", tc.desc, wantCountName, counts)
				break
			} else if wantCountVal != getCountVal {
				t.Errorf("Test (%s): failed, for counter %s, expected value %v:, got value: %v ", tc.desc, wantCountName, wantCountVal, getCountVal)
				break
			}
		}

		for wantHistogramName, wantHistogramVals := range tc.wantHistograms {
			if getHistogramVals, ok := histograms[wantHistogramName]; !ok {
				t.Errorf("Test (%s): failed, expected histogram %v not in the got histograms: %v", tc.desc, wantHistogramName, histograms)
				break
			} else if len(wantHistogramVals) != len(getHistogramVals) {
				t.Errorf("Test (%s): failed, differnt value number for histogram %v, expected vals: %v , got vals: %v", tc.desc, wantHistogramName, wantHistogramVals, getHistogramVals)
			} else {
				for i, wantHistogramVal := range wantHistogramVals {
					if wantHistogramVal == 0 {
						continue
					}
					if wantHistogramVal/bufferPercent < getHistogramVals[i] || wantHistogramVal*bufferPercent > getHistogramVals[i] {
						t.Errorf("Test (%s): failed, histogram %v not matched, expected vals: %v , got vals: %v", tc.desc, wantHistogramName, wantHistogramVals, getHistogramVals)
						break
					}
				}
			}
		}
	}
}
