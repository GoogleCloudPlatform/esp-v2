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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

type RolloutIdDetector struct {
	serviceName           string
	serviceControlUrl     string
	client                *http.Client
	curRolloutId          string
	accessToken           util.GetAccessTokenFunc
	detectRolloutIdTicker *time.Ticker
}

func NewRolloutIdDetector(client *http.Client, serviceControlUrl, serviceName string,
	accessToken util.GetAccessTokenFunc) *RolloutIdDetector {
	return &RolloutIdDetector{
		client:            client,
		serviceName:       serviceName,
		serviceControlUrl: serviceControlUrl,
		accessToken:       accessToken,
	}

}

func (c *RolloutIdDetector) FetchLatestRolloutId() (string, error) {
	reportResponse := new(scpb.ReportResponse)
	fetchRolloutIdUrl := util.FetchRolloutIdURL(c.serviceControlUrl, c.serviceName)
	if err := util.CallGooglelapis(c.client, fetchRolloutIdUrl, util.POST, c.accessToken, reportResponse); err != nil {
		return "", fmt.Errorf("fail to fetch new rollout id, %v", err)
	}

	return reportResponse.ServiceRolloutId, nil
}

func (c *RolloutIdDetector) SetDetectRolloutIdChangeTimer(interval time.Duration, callback func(latestRolloutId string)) {
	go func() {
		glog.Infof("start detect latest rollout id every %v", interval)
		c.detectRolloutIdTicker = time.NewTicker(interval)

		for range c.detectRolloutIdTicker.C {
			latestRolloutId, err := c.FetchLatestRolloutId()
			if err != nil {
				glog.Errorf("error occurred when checking new rollout id, %v", err)
				continue
			}

			if latestRolloutId == c.curRolloutId {
				continue
			}

			c.curRolloutId = latestRolloutId
			callback(c.curRolloutId)
		}
	}()
}
