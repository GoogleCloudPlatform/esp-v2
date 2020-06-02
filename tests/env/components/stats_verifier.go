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

package components

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/golang/glog"
)

const (
	// Ensure the stats are available in admin.
	fetchDelay = time.Second * 3

	// Margin of error for latency equality.
	latencyMargin = 0.8
)

type StatsVerifier struct {
	adminPort uint16
}

func NewStatsVerifier(ports *Ports) *StatsVerifier {
	return &StatsVerifier{
		adminPort: ports.AdminPort,
	}
}

func (sv StatsVerifier) CheckExpectedCounters(wantCounters utils.StatCounters) error {
	glog.Infof("Checking envoy counters")
	time.Sleep(fetchDelay)

	counters, _, err := utils.FetchStats(sv.adminPort)
	if err != nil {
		return err
	}

	for wantCounter, wantCounterVal := range wantCounters {
		if getCountVal, ok := counters[wantCounter]; !ok {
			return fmt.Errorf("expected counter %v not in the got counters: %v", wantCounter, counters)
		} else if getCountVal != wantCounterVal {
			return fmt.Errorf("for counter %v, expected value %v:, got value: %v ", wantCounter, wantCounterVal, getCountVal)
		}
	}

	return nil
}

func (sv StatsVerifier) CheckExpectedHistograms(wantHistograms utils.StatHistograms) error {
	glog.Infof("Checking envoy histograms")
	time.Sleep(fetchDelay)

	_, histograms, err := utils.FetchStats(sv.adminPort)
	if err != nil {
		return err
	}

	for wantHistogramName, wantHistogramVals := range wantHistograms {
		getHistogramVals, ok := histograms[wantHistogramName]
		if !ok {
			return fmt.Errorf("expected histogram %v not in the got histograms: %v", wantHistogramName, histograms)
		}

		if len(wantHistogramVals) != len(getHistogramVals) {
			return fmt.Errorf("different value number for histogram %v, expected vals: %v , got vals: %v", wantHistogramName, wantHistogramVals, getHistogramVals)
		}

		for i, wantHistogramVal := range wantHistogramVals {
			if wantHistogramVal == 0.0 {
				continue
			}

			if !roughEqual(getHistogramVals[i], wantHistogramVal, latencyMargin) {
				return fmt.Errorf("histogram %v not matched, expected vals: %v , got vals: %v", wantHistogramName, wantHistogramVals, getHistogramVals)
			}
		}
	}
	return nil
}

func (sv StatsVerifier) VerifyInvariants() error {
	glog.Infof("Verifying envoy stats invariants")
	time.Sleep(fetchDelay)

	counters, _, err := utils.FetchStats(sv.adminPort)
	if err != nil {
		return err
	}

	if counters["http.ingress_http.service_control.allowed"] <
		counters["http.ingress_http.service_control.allowed_control_plane_fault"] {
		return fmt.Errorf("sc allowed invariant failed: %v", counters)
	}

	if counters["http.ingress_http.service_control.denied"] !=
		counters["http.ingress_http.service_control.denied_control_plane_fault"]+
			counters["http.ingress_http.service_control.denied_producer_error"]+
			counters["http.ingress_http.service_control.denied_consumer_quota"]+
			counters["http.ingress_http.service_control.denied_consumer_blocked"]+
			counters["http.ingress_http.service_control.denied_consumer_error"] {
		return fmt.Errorf("sc denied invariant failed: %v", counters)
	}

	glog.Infof("Verified stats invariants successfully")
	return nil
}

func (sv StatsVerifier) String() string {
	return "Envoy Admin Endpoint for Stats"
}

func (sv StatsVerifier) CheckHealth() error {
	endpoint := fmt.Sprintf("http://%v:%v%v", platform.GetLoopbackAddress(), sv.adminPort, utils.ESpv2FiltersStatsPath)
	opts := NewHealthCheckOptions()
	return HttpHealthCheck(endpoint, opts)
}

func roughEqual(i, j, latencyMargin float64) bool {
	return i > j*(1-latencyMargin) && i < j*(1+latencyMargin)
}
