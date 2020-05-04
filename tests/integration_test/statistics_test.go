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

func roughEqual(i, j, latencyMargin float64) bool {
	return i > j*(1-latencyMargin) && i < j*(1+latencyMargin)
}

func TestStatistics(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(comp.TestStatistics, platform.EchoRemote)

	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	const latencyMargin = 0.8

	//the latency is different each run. Here the backend server introduces a
	// fixed big N second latency, so the overall latency should be around N
	// second. This test compares the latency with expected N second with certain
	// error margin. As for the overhead time, the test only checks if the number of
	// statistic values is correct by setting 0.0, which will be skipped for exact
	// comparison.
	testData := []struct {
		desc           string
		reqCnt         int
		reqDuration    time.Duration
		wantCounts     map[string]int
		wantHistograms map[string][]float64
	}{
		{
			desc:        "backend respond in 1s",
			reqCnt:      2,
			reqDuration: time.Second * 1,
			wantCounts: map[string]int{
				"http.ingress_http.backend_auth.token_added":                       2,
				"http.ingress_http.backend_routing.append_path_to_address_request": 2,
				"http.ingress_http.path_matcher.allowed":                           2,
				"http.ingress_http.service_control.allowed":                        2,
			},
			wantHistograms: map[string][]float64{
				"http.ingress_http.service_control.overhead_time": {0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
				"http.ingress_http.service_control.backend_time":  {1000, 1025, 1050, 1075, 1090, 1095, 1099, 1099.5, 1099.9, 1100},
				"http.ingress_http.service_control.request_time":  {1000, 1025, 1050, 1075, 1090, 1095, 1099, 1099.5, 1099.9, 1100},
			},
		},
	}

	for _, tc := range testData {
		path := fmt.Sprintf("/sleepDefault?duration=%v", tc.reqDuration.String())
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, path)

		for i := 0; i < tc.reqCnt; i += 1 {
			if _, err := client.DoWithHeaders(url, "GET", "", nil); err != nil {
				t.Fatalf("Test (%s): failed, expected no err, got err (%v)", tc.desc, err)
			}
		}

		// Ensure the stats is available in admin.
		time.Sleep(time.Millisecond * 5000)

		statsUrl := fmt.Sprintf("http://localhost:%v%v", s.Ports().AdminPort, utils.ESpv2FiltersStatsPath)
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
			getHistogramVals, ok := histograms[wantHistogramName]
			if !ok {
				t.Errorf("Test (%s): failed, expected histogram %v not in the got histograms: %v", tc.desc, wantHistogramName, histograms)
				break
			}

			if len(wantHistogramVals) != len(getHistogramVals) {
				t.Errorf("Test (%s): failed, different value number for histogram %v, expected vals: %v , got vals: %v", tc.desc, wantHistogramName, wantHistogramVals, getHistogramVals)
				continue
			}

			for i, wantHistogramVal := range wantHistogramVals {
				if wantHistogramVal == 0.0 {
					continue
				}

				if !roughEqual(getHistogramVals[i], wantHistogramVal, latencyMargin) {
					t.Errorf("Test (%s): failed, histogram %v not matched, expected vals: %v , got vals: %v", tc.desc, wantHistogramName, wantHistogramVals, getHistogramVals)
					break
				}
			}

		}
	}
}
