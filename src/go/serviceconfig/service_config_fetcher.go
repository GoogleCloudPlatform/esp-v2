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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

type ServiceConfigFetcher struct {
	serviceManagementUrl string
	serviceName          string
	client               *http.Client
	accessToken          util.GetAccessTokenFunc
	curServiceConfig     *confpb.Service
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
	_fetchConfig := func(configId string) (*confpb.Service, error) {
		if configId == s.curConfigId() {
			return nil, nil
		}

		serviceConfig := new(confpb.Service)
		fetchConfigUrl := util.FetchConfigURL(s.serviceManagementUrl, s.serviceName, configId)
		if err := util.CallGooglelapis(s.client, fetchConfigUrl, util.GET, s.accessToken, serviceConfig); err != nil {
			return nil, err
		}

		return serviceConfig, nil
	}

	serviceConfig, err := _fetchConfig(configId)
	if err != nil {
		return nil, err
	}

	if serviceConfig != nil {
		s.curServiceConfig = serviceConfig
	}

	return serviceConfig, nil
}

// Fetch the rollout and among its all service configs, pick up the one with
// highest traffic percentage.
func (s *ServiceConfigFetcher) GetConfigIdByFetchRollout(rolloutId string) (string, error) {
	rollout := new(smpb.Rollout)
	fetchRolloutUrl := util.FetchRolloutURL(s.serviceManagementUrl, s.serviceName, rolloutId)
	if err := util.CallGooglelapis(s.client, fetchRolloutUrl, util.GET, s.accessToken, rollout); err != nil {
		return "", err
	}

	return highestTrafficConfigIdInRollout(rollout)
}

func (s *ServiceConfigFetcher) curConfigId() string {
	if s.curServiceConfig == nil {
		return ""
	}
	return s.curServiceConfig.Id
}

func highestTrafficConfigIdInRollout(rollout *smpb.Rollout) (string, error) {
	if rollout == nil || rollout.GetTrafficPercentStrategy() == nil ||
		len(rollout.GetTrafficPercentStrategy().Percentages) == 0 {
		return "", fmt.Errorf("problematic rollout %v", rollout)
	}

	highestPercent := 0.
	highTrafficConfigId := ""
	for configId, percent := range rollout.GetTrafficPercentStrategy().Percentages {
		if percent > highestPercent {
			highestPercent = percent
			highTrafficConfigId = configId
		}
	}

	return highTrafficConfigId, nil
}
