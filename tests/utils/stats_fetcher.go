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

package utils

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/golang/glog"
)

type StatCounters map[string]int
type StatHistograms map[string][]float64

const (
	// Path with filtering for ESPv2 stats.
	ESpv2FiltersStatsPath = "/stats?format=json&usedonly&filter=http.ingress_http.(backend_auth|service_control|path_rewrite)"
)

// Stats is the struct to decode envoy admin json raw data.
type Stats struct {
	Stat []StatsData `json:"stats"`
}

// StatsData is the struct to decode the stats data, which may include the
// metric name and value, and metrics' histogram.
type StatsData struct {
	MetricName  string    `json:"name,omitempty"`
	MetricValue float64   `json:"value,omitempty"`
	Bar         Histogram `json:"histograms,omitempty"`
}

// Histogram is the struct which is an optional part of StatsData.
type Histogram struct {
	// Cq represents computed_quantiles.
	Cq []ComputedQuantiles `json:"computed_quantiles,omitempty"`
	// Sq represents supported_quantiles.
	Sq []interface{} `json:"supported_quantiles,omitempty"`
}

// ComputedQuantiles is the struct to represent the computed quantile for each histogram.
type ComputedQuantiles struct {
	Name   string  `json:"name"`
	Values []Point `json:"values"`
}

// Point is the struct to decode the values of computed_quantiles.
type Point struct {
	Cumulative float64 `json:"cumulative,omitempty"`
	Interval   float64 `json:"interval,omitempty"`
}

func FetchStats(adminPort uint16) (StatCounters, StatHistograms, error) {
	glog.Infof("Fetching stats from envoy")

	// Fetch from envoy admin.
	statsUrl := fmt.Sprintf("http://localhost:%v%v", adminPort, ESpv2FiltersStatsPath)
	_, statsResp, err := DoWithHeaders(statsUrl, "GET", "", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch envoy stats: %v", err)
	}

	// Parse into our own structures (maps) for easy lookups.
	var stats Stats
	if err := json.Unmarshal(statsResp, &stats); err != nil {
		return nil, nil, fmt.Errorf("fail to unmarshal respnse to Stats: %v", err)
	}

	counters := StatCounters{}
	histograms := StatHistograms{}

	for _, stat := range stats.Stat {
		if metricName := stat.MetricName; metricName != "" {
			counters[metricName] = int(stat.MetricValue)
			continue
		}

		for _, cp := range stat.Bar.Cq {
			if len(cp.Values) == 0 {
				continue
			}
			sort.SliceStable(cp.Values, func(i, j int) bool { return cp.Values[i].Interval < cp.Values[j].Interval })
			for _, v := range cp.Values {
				histograms[cp.Name] = append(histograms[cp.Name], v.Cumulative)
			}
		}
	}

	glog.Infof("Fetched stats\n  counters: %v\n  histograms: %v", counters, histograms)
	return counters, histograms, nil
}
