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
	"net/http"
	"time"

	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

type ServiceRolloutIdFetcher struct {
	serviceName       string
	serviceControlUrl string
	client            http.Client
	accessToken       func() (string, time.Duration, error)
}

func NewServiceRolloutIdFetcher(serviceName, serviceControlUrl string, client http.Client,
	accessToken util.GetAccessTokenFunc) *ServiceRolloutIdFetcher {
	return &ServiceRolloutIdFetcher{
		serviceName:       serviceName,
		serviceControlUrl: serviceControlUrl,
		client:            client,
		accessToken:       accessToken,
	}

}

func (c *ServiceRolloutIdFetcher) fetchNewRolloutId() (string, error) {
	reportResponse := new(scpb.ReportResponse)
	fetchRolloutIdUrl := util.FetchRolloutIdURL(c.serviceControlUrl, c.serviceName)
	if err := util.CallGooglelapis(c.client, fetchRolloutIdUrl, util.POST, c.accessToken, reportResponse); err != nil {
		return "", fmt.Errorf("fail to fetch new rollout id, %v", err)
	}

	return reportResponse.ServiceRolloutId, nil
}
