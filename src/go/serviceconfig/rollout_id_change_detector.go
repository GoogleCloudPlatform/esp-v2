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
	"time"

	scpb "cloud.google.com/go/servicecontrol/apiv1/servicecontrolpb"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
)

type RolloutIdChangeDetector struct {
	serviceName           string
	serviceControlUrl     string
	client                *http.Client
	curRolloutId          string
	accessToken           util.GetAccessTokenFunc
	detectRolloutIdTicker *time.Ticker
}

func NewRolloutIdChangeDetector(client *http.Client, serviceControlUrl, serviceName string,
	accessToken util.GetAccessTokenFunc) *RolloutIdChangeDetector {
	return &RolloutIdChangeDetector{
		client:            client,
		serviceName:       serviceName,
		serviceControlUrl: serviceControlUrl,
		accessToken:       accessToken,
	}

}

func (c *RolloutIdChangeDetector) fetchLatestRolloutId() (string, error) {
	reportResponse := new(scpb.ReportResponse)
	fetchRolloutIdUrl := util.FetchRolloutIdURL(c.serviceControlUrl, c.serviceName)
	util.CallGoogleapisMu.RLock()
	callGoogleapis := util.CallGoogleapis
	util.CallGoogleapisMu.RUnlock()
	if err := callGoogleapis(c.client, fetchRolloutIdUrl, util.POST, c.accessToken, nil, reportResponse); err != nil {
		return "", fmt.Errorf("fail to fetch new rollout id, %v", err)
	}

	return reportResponse.ServiceRolloutId, nil
}

// SetDetectRolloutIdChangeTimer checks the latest rollout id every interval and
// calls callback with it when it differs from the last handled one. The rollout
// id is recorded as handled only when callback returns true, so a rollout whose
// config could not be loaded yet is checked again on the next tick.
func (c *RolloutIdChangeDetector) SetDetectRolloutIdChangeTimer(interval time.Duration, callback func(latestRolloutId string) bool) {
	go func() {
		glog.Infof("start detect latest rollout id every %v", interval)
		c.detectRolloutIdTicker = time.NewTicker(interval)

		for range c.detectRolloutIdTicker.C {
			latestRolloutId, err := c.fetchLatestRolloutId()
			if err != nil {
				glog.Errorf("error occurred when checking new rollout id, %v", err)
				continue
			}

			if latestRolloutId == c.curRolloutId {
				continue
			}

			if callback(latestRolloutId) {
				c.curRolloutId = latestRolloutId
			}
		}
	}()
}
