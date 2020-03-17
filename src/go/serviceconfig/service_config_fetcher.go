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
	"math"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

type ServiceConfigFetcher struct {
	serviceName         string
	checkRolloutsTicker *time.Ticker
	client              http.Client
	mf                  *metadata.MetadataFetcher
	opts                options.ConfigGeneratorOptions

	curServiceConfig *confpb.Service
	curRolloutId     string
}

func NewServiceConfigFetcher(mf *metadata.MetadataFetcher, opts options.ConfigGeneratorOptions, serviceName string) (*ServiceConfigFetcher, error) {
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
		mf:          mf,
		opts:        opts,
	}
	return scf, nil
}

// Fetch the service config by given configId. If configId is empty, try to
// fetch the latest service config,.
func (scf *ServiceConfigFetcher) FetchConfig(configId string) (*confpb.Service, error) {
	_fetchConfig := func(configId string) (*confpb.Service, error) {
		if configId != "" {
			token, _, err := scf.accessToken()
			if err != nil {
				return nil, fmt.Errorf("fail to get access token: %v", err)
			}
			return scf.callServiceManagement(util.FetchConfigURL(scf.opts.ServiceManagementURL, scf.serviceName, configId), token)
		}

		glog.Infof("check new rollouts for service %v", scf.serviceName)
		newRolloutId, newConfigId, err := scf.loadConfigFromRollouts(scf.serviceName, scf.curRolloutId, scf.curConfigId())
		if err != nil {
			glog.Errorf("error occurred when checking new rollouts, %v", err)
		}
		if scf.curRolloutId != newRolloutId && scf.curConfigId() != newConfigId {
			scf.curRolloutId = newRolloutId
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

func (scf *ServiceConfigFetcher) SetFetchConfigTimer(interval *time.Duration, callback func(serviceConfig *confpb.Service)) {
	go func() {
		glog.Infof("start checking new rollouts every %v seconds", *interval)
		scf.checkRolloutsTicker = time.NewTicker(*interval)

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

// TODO(taoxuy): remove this after relying on service control for configId
func (scf *ServiceConfigFetcher) CurRolloutId() string {
	return scf.curRolloutId
}

func (scf *ServiceConfigFetcher) curConfigId() string {
	if scf.curServiceConfig == nil {
		return ""
	}
	return scf.curServiceConfig.Id
}

func (scf *ServiceConfigFetcher) loadConfigFromRollouts(serviceName, curRolloutId, curConfigId string) (string, string, error) {
	var err error
	var listServiceRolloutsResponse *smpb.ListServiceRolloutsResponse
	listServiceRolloutsResponse, err = scf.fetchRollouts()
	if err != nil {
		return "", "", fmt.Errorf("fail to get rollouts, %s", err)
	}

	if len(listServiceRolloutsResponse.Rollouts) == 0 {
		return "", "", fmt.Errorf("no active rollouts")
	}
	newRolloutId := listServiceRolloutsResponse.Rollouts[0].RolloutId
	if newRolloutId == curRolloutId {
		return curRolloutId, curConfigId, nil
	}
	glog.Infof("found new rollout Id %v for service %v", newRolloutId, serviceName)
	glog.Infof("new rollout: %v", listServiceRolloutsResponse.Rollouts[0])
	trafficPercentStrategy := listServiceRolloutsResponse.Rollouts[0].GetTrafficPercentStrategy()
	trafficPercentMap := trafficPercentStrategy.GetPercentages()
	if len(trafficPercentMap) == 0 {
		return "", "", fmt.Errorf("no active rollouts")
	}
	var newConfigId string
	currentMaxPercent := 0.0
	// take config Id with max traffic percent as new config Id
	for k, v := range trafficPercentMap {
		if v > currentMaxPercent {
			newConfigId = k
			currentMaxPercent = v
		}
	}
	if newConfigId == curConfigId {
		glog.Infof("no new configuration to load for service %v, current configuration Id %v", serviceName, curConfigId)
		return newRolloutId, curConfigId, nil
	}
	if !(math.Abs(100.0-currentMaxPercent) < 1e-9) {
		glog.Warningf("though traffic percentage of configuration %v is %v%%, set it to 100%%", newConfigId, currentMaxPercent)
	}
	glog.Infof("found new configuration Id %v for service %v", curConfigId, serviceName)
	return newRolloutId, newConfigId, nil
}

func (scf *ServiceConfigFetcher) accessToken() (string, time.Duration, error) {
	if scf.mf == nil && scf.opts.ServiceAccountKey == "" {
		return "", 0, fmt.Errorf("If --non_gcp is specified, --service_account_key has to be specified.")
	}
	if scf.opts.ServiceAccountKey != "" {
		return util.GenerateAccessTokenFromFile(scf.opts.ServiceAccountKey)
	}
	return scf.mf.FetchAccessToken()
}

// TODO(jcwang) cleanup here. This function is redundant.
func (scf *ServiceConfigFetcher) fetchRollouts() (*smpb.ListServiceRolloutsResponse, error) {
	token, _, err := scf.accessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token: %v", err)
	}

	return scf.callServiceManagementRollouts(util.FetchRolloutsURL(scf.opts.ServiceManagementURL, scf.serviceName), token)
}

// TODO(taoxuy): replace this with callServiceControl for configId
func (scf *ServiceConfigFetcher) callServiceManagementRollouts(path, token string) (*smpb.ListServiceRolloutsResponse, error) {
	var err error
	var resp *http.Response
	if resp, err = scf.callWithAccessToken(path, token); err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read response body: %s", err)
	}
	defer resp.Body.Close()
	rolloutsResponse := new(smpb.ListServiceRolloutsResponse)
	if err := proto.Unmarshal(body, rolloutsResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal ListServiceRolloutsResponse: %s", err)
	}
	return rolloutsResponse, nil
}

func (scf *ServiceConfigFetcher) callServiceManagement(path, token string) (*confpb.Service, error) {
	var err error
	var resp *http.Response
	if resp, err = scf.callWithAccessToken(path, token); err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read response body: %s", err)
	}
	defer resp.Body.Close()
	service := new(confpb.Service)
	if err := proto.Unmarshal(body, service); err != nil {
		return nil, fmt.Errorf("fail to unmarshal Service: %v", err)
	}
	return service, nil
}

func (scf *ServiceConfigFetcher) callWithAccessToken(path, token string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-protobuf")
	resp, err := scf.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http call to %s returns not 200 OK: %v", path, resp.Status)
	}
	return resp, nil
}
