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

type ServiceConfigFetcher struct {
	serviceName         string
	checkRolloutsTicker *time.Ticker
	client              http.Client
	opts                *options.ConfigGeneratorOptions

	accessToken func() (string, time.Duration, error)
	newConfigId func() (string, error)

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
	scf := &ServiceConfigFetcher{
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

	configIdFetcher := NewConfigIdFetcher(serviceName, opts.ServiceControlURL,
		scf.client, func() (string, time.Duration, error) { return scf.accessToken() })
	scf.newConfigId = func() (string, error) {
		return configIdFetcher.fetchNewConfigId()
	}

	scf.accessToken = func() (string, time.Duration, error) {
		if accessTokenFromImds == nil && scf.opts.ServiceAccountKey == "" {
			return "", 0, fmt.Errorf("If --non_gcp is specified, --service_account_key has to be specified.")
		}
		if scf.opts.ServiceAccountKey != "" {
			return util.GenerateAccessTokenFromFile(scf.opts.ServiceAccountKey)
		}
		return accessTokenFromImds()
	}

	return scf, nil
}

// Fetch the service config by given configId. If configId is empty, try to
// fetch the latest service config.
func (scf *ServiceConfigFetcher) FetchConfig(configId string) (*confpb.Service, error) {
	_fetchConfig := func(configId string) (*confpb.Service, error) {

		if configId != "" {
			if configId == scf.curConfigId() {
				return nil, nil
			}

			token, _, err := scf.accessToken()
			if err != nil {
				return nil, fmt.Errorf("fail to get access token: %v", err)
			}

			return scf.callServiceManagement(util.FetchConfigURL(scf.opts.ServiceManagementURL, scf.serviceName, configId), token)
		}

		glog.Infof("check new config id for service %v", scf.serviceName)
		newConfigId, err := scf.newConfigId()
		if err != nil {
			return nil, fmt.Errorf("error occurred when checking new service config id: %v", err)
		}

		if newConfigId != scf.curConfigId() {
			token, _, err := scf.accessToken()
			if err != nil {
				return nil, fmt.Errorf("fail to get access token: %v", err)
			}

			return scf.callServiceManagement(util.FetchConfigURL(scf.opts.ServiceManagementURL, scf.serviceName, newConfigId), token)
		}

		return nil, nil
	}
	serviceConfig, err := _fetchConfig(configId)
	if err == nil {
		scf.curServiceConfig = serviceConfig

	}

	return serviceConfig, err
}

func (scf *ServiceConfigFetcher) SetFetchConfigTimer(interval time.Duration, callback func(serviceConfig *confpb.Service)) {
	go func() {
		glog.Infof("start checking new rollouts every %v seconds", interval)
		scf.checkRolloutsTicker = time.NewTicker(interval)

		for range scf.checkRolloutsTicker.C {
			glog.Infof("check new rollouts for service %v", scf.serviceName)

			serviceConfig, err := scf.FetchConfig("")
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

func (scf *ServiceConfigFetcher) curConfigId() string {
	if scf.curServiceConfig == nil {
		return ""
	}
	return scf.curServiceConfig.Id
}

func (scf *ServiceConfigFetcher) callServiceManagement(path, token string) (*confpb.Service, error) {
	respBytes, err := util.CallWithAccessToken(scf.client, util.GET, path, token)
	if err != nil {
		return nil, err
	}

	service := new(confpb.Service)
	if err := proto.Unmarshal(respBytes, service); err != nil {
		return nil, fmt.Errorf("fail to unmarshal Service: %v", err)
	}
	return service, nil
}
