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
	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

type GetAccessTokenFunc func() (string, time.Duration, error)
type GetNewConfigIdFunc func() (string, error)

type ServiceConfigFetcher struct {
	serviceName         string
	checkRolloutsTicker *time.Ticker
	client              http.Client
	opts                *options.ConfigGeneratorOptions

	accessToken GetAccessTokenFunc
	newConfigId GetNewConfigIdFunc

	curServiceConfig *confpb.Service
}

func NewServiceConfigFetcher(opts *options.ConfigGeneratorOptions,
	serviceName string, accessTokenFromImds func() (string, time.Duration, error)) (*ServiceConfigFetcher, error) {
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
	}

	configIdFetcher := NewServiceConfigIdFetcher(serviceName, opts.ServiceControlURL,
		s.client, func() (string, time.Duration, error) { return s.accessToken() })
	s.newConfigId = func() (string, error) {
		return configIdFetcher.fetchNewConfigId()
	}

	s.accessToken = func() (string, time.Duration, error) {
		// when --non_gcp  is set, instance metadata server(imds) is not defined so
		// accessToken is unavailable from imds and serviceAccountKey must be set to
		// generate accessToken.
		if accessTokenFromImds == nil && s.opts.ServiceAccountKey == "" {
			return "", 0, fmt.Errorf("If --non_gcp is specified, --service_account_key has to be specified.")
		}
		if s.opts.ServiceAccountKey != "" {
			return util.GenerateAccessTokenFromFile(s.opts.ServiceAccountKey)
		}
		return accessTokenFromImds()
	}

	return s, nil
}

// Fetch the service config by given configId. If configId is empty, try to
// fetch the latest service config.
func (s *ServiceConfigFetcher) FetchConfig(configId string) (*confpb.Service, error) {
	_fetchConfig := func(configId string) (*confpb.Service, error) {

		if configId != "" {
			if configId == s.curConfigId() {
				return nil, nil
			}

			token, _, err := s.accessToken()
			if err != nil {
				return nil, fmt.Errorf("fail to get access token: %v", err)
			}

			return s.callServiceManagement(util.FetchConfigURL(s.opts.ServiceManagementURL, s.serviceName, configId), token)
		}

		glog.Infof("check new config id for service %v", s.serviceName)
		newConfigId, err := s.newConfigId()
		if err != nil {
			return nil, fmt.Errorf("error occurred when checking new service config id: %v", err)
		}

		if newConfigId != s.curConfigId() {
			token, _, err := s.accessToken()
			if err != nil {
				return nil, fmt.Errorf("fail to get access token: %v", err)
			}

			return s.callServiceManagement(util.FetchConfigURL(s.opts.ServiceManagementURL, s.serviceName, newConfigId), token)
		}

		return nil, nil
	}
	serviceConfig, err := _fetchConfig(configId)
	if err == nil {
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

func (s *ServiceConfigFetcher) callServiceManagement(path, token string) (*confpb.Service, error) {
	respBytes, err := util.CallWithAccessToken(s.client, util.GET, path, token)
	if err != nil {
		return nil, err
	}

	service := new(confpb.Service)
	if err := proto.Unmarshal(respBytes, service); err != nil {
		return nil, fmt.Errorf("fail to unmarshal Service: %v", err)
	}
	return service, nil
}
