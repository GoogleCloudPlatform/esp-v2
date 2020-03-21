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
	"math"
	"net/http"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

type ServiceConfigFetcher struct {
	serviceManagementUrl string
	serviceName          string
	client               *http.Client
	accessToken          util.GetAccessTokenFunc
}

func NewServiceConfigFetcher(client *http.Client, serviceManagementUrl,
	serviceName string, accessToken util.GetAccessTokenFunc) *ServiceConfigFetcher {
	return &ServiceConfigFetcher{
		client:               client,
		serviceName:          serviceName,
		serviceManagementUrl: serviceManagementUrl,
		accessToken:          accessToken,
	}
}

// Fetch the service config by given configId.
func (s *ServiceConfigFetcher) FetchConfig(configId string) (*confpb.Service, error) {
	serviceConfig := new(confpb.Service)
	fetchConfigUrl := util.FetchConfigURL(s.serviceManagementUrl, s.serviceName, configId)
	if err := util.CallGooglelapis(s.client, fetchConfigUrl, util.GET, s.accessToken, serviceConfig); err != nil {
		return nil, err
	}

	return serviceConfig, nil
}

// Fetch all the rollouts and use the latest success rollout. Among its all
// service configs, pick up the one with highest traffic percentage.
func (s *ServiceConfigFetcher) LoadConfigIdFromRollouts() (string, error) {
	rollouts := new(smpb.ListServiceRolloutsResponse)
	fetchRolloutUrl := util.FetchRolloutsURL(s.serviceManagementUrl, s.serviceName)
	if err := util.CallGooglelapis(s.client, fetchRolloutUrl, util.GET, s.accessToken, rollouts); err != nil {
		return "", err
	}

	return highestTrafficConfigIdInLatestRollout(rollouts)
}

func highestTrafficConfigIdInLatestRollout(rollouts *smpb.ListServiceRolloutsResponse) (string, error) {
	if rollouts == nil || len(rollouts.GetRollouts()) == 0 {
		return "", fmt.Errorf("problematic rollouts: %v", rollouts)
	}

	latestRollout := rollouts.GetRollouts()[0]

	highestPercent := 0.
	highTrafficConfigId := ""
	for configId, percent := range latestRollout.GetTrafficPercentStrategy().Percentages {
		if percent > highestPercent {
			highestPercent = percent
			highTrafficConfigId = configId
		}
	}

	if !(math.Abs(100.0-highestPercent) < 1e-9) {
		glog.Warningf("though traffic percentage of configuration %v is %v%%, set it to 100%%", highTrafficConfigId, highestPercent)
	}
	return highTrafficConfigId, nil
}
