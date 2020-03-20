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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

type ServiceConfigFetcher struct {
	serviceName         string
	checkRolloutsTicker *time.Ticker
	client              http.Client
	opts                *options.ConfigGeneratorOptions

	accessToken  util.GetAccessTokenFunc
	newRolloutId util.GetNewRolloutIdFunc

	curServiceConfig *confpb.Service
	curRolloutId     string
}

func NewServiceConfigFetcher(opts *options.ConfigGeneratorOptions,
	serviceName string, accessToken util.GetAccessTokenFunc) (*ServiceConfigFetcher, error) {
	caCert, err := ioutil.ReadFile(opts.RootCertsPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	s := &ServiceConfigFetcher{
		client: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
			Timeout: opts.HttpRequestTimeout,
		},
		serviceName: serviceName,
		opts:        opts,
		accessToken: accessToken,
	}

	rolloutIdFetcher := NewServiceRolloutIdFetcher(serviceName, opts.ServiceControlURL,
		s.client, accessToken)
	s.newRolloutId = func() (string, error) {
		return rolloutIdFetcher.fetchNewRolloutId()
	}

	return s, nil
}

// Fetch the service config by given configId. If configId is empty, try to
// fetch the latest rollout and among all its config traffic percentages, pickup
// the config id with highest traffic percentage and fetch the service config.
func (s *ServiceConfigFetcher) FetchConfig(configId string) (*confpb.Service, error) {
	_fetchConfig := func(rolloutId string) (string, *confpb.Service, error) {
		if configId != "" {
			if configId == s.curConfigId() {
				return "", nil, nil
			}

			serviceConfig := new(confpb.Service)
			fetchConfigUrl := util.FetchConfigURL(s.opts.ServiceManagementURL, s.serviceName, configId)
			if err := util.CallGooglelapis(s.client, fetchConfigUrl, util.GET, s.accessToken, serviceConfig); err != nil {
				return "", nil, err
			}

			return "", serviceConfig, nil
		}

		glog.Infof("check new config id for service %v", s.serviceName)
		newRolloutId, err := s.newRolloutId()
		if err != nil {
			return "", nil, fmt.Errorf("error occurred when checking new service rollout id: %v", err)
		}

		if newRolloutId == s.curRolloutId {
			return "", nil, nil
		}

		rollout := new(smpb.Rollout)
		fetchRolloutUrl := util.FetchRolloutURL(s.opts.ServiceManagementURL, s.serviceName, newRolloutId)
		if err = util.CallGooglelapis(s.client, fetchRolloutUrl, util.GET, s.accessToken, rollout); err != nil {
			return "", nil, err
		}

		newConfigId, err := highestTrafficConfigIdInRollout(rollout)
		if err != nil {
			return "", nil, err
		}

		if newConfigId == s.curConfigId() {
			return newRolloutId, nil, nil
		}

		serviceConfig := new(confpb.Service)
		fetchConfigUrl := util.FetchConfigURL(s.opts.ServiceManagementURL, s.serviceName, newConfigId)
		if err := util.CallGooglelapis(s.client, fetchConfigUrl, util.GET, s.accessToken, serviceConfig); err != nil {
			return "", nil, err
		}

		return newRolloutId, serviceConfig, err
	}

	rolloutId, serviceConfig, err := _fetchConfig(configId)

	if err == nil && serviceConfig != nil {
		s.curRolloutId = rolloutId
		s.curServiceConfig = serviceConfig
	}

	return serviceConfig, err
}

func (s *ServiceConfigFetcher) SetFetchConfigTimer(interval time.Duration, callback func(serviceConfig *confpb.Service)) {
	go func() {
		glog.Infof("start checking new rollouts every %v seconds", interval)
		s.checkRolloutsTicker = time.NewTicker(interval)

		for range s.checkRolloutsTicker.C {
			glog.Infof("check new rollouts for service %v", s.serviceName)

			serviceConfig, err := s.FetchConfig("")
			if err != nil {
				glog.Errorf("error occurred when checking new rollouts, %v", err)
				continue

			}

			if serviceConfig != nil {
				callback(serviceConfig)
			}
		}
	}()
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
