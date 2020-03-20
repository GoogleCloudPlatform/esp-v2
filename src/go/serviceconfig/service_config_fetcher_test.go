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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func initServiceManagementForTestServiceConfigFetcherFetchConfig(t *testing.T,
	serviceRollout *smpb.Rollout, serviceConfig *confpb.Service, serviceName string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/v1/services/%s/rollouts/%s", serviceName, serviceRollout.RolloutId):
			resp, err := proto.Marshal(serviceRollout)
			if err != nil {
				t.Fatalf("fail to generate rollout response: %v", err)
			}
			_, _ = w.Write(resp)
		case fmt.Sprintf("/v1/services/%s/configs/%s", serviceName, serviceConfig.Id):
			resp, err := proto.Marshal(serviceConfig)
			if err != nil {
				t.Fatalf("fail to generate config response: %v", err)
			}
			_, _ = w.Write(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}

	}))
}

func genRolloutAndConfig(serviceRolloutId, serviceConfigId string) (*smpb.Rollout, *confpb.Service) {
	serviceRollout := &smpb.Rollout{
		RolloutId: serviceRolloutId,
		Strategy: &smpb.Rollout_TrafficPercentStrategy_{
			TrafficPercentStrategy: &smpb.Rollout_TrafficPercentStrategy{
				Percentages: map[string]float64{
					serviceConfigId: 100,
				},
			},
		},
	}
	serviceConfig := &confpb.Service{
		Name: "foo",
		Id:   serviceConfigId,
	}
	return serviceRollout, serviceConfig
}

func updateRolloutAndConfig(serviceRollout *smpb.Rollout, serviceConfig *confpb.Service, serviceRolloutId, serviceConfigId string) {
	serviceConfig.Id = serviceConfigId
	serviceRollout.RolloutId = serviceRolloutId
	serviceRollout.GetTrafficPercentStrategy().Percentages = map[string]float64{
		serviceConfigId: 100,
	}
}

func TestServiceConfigFetcherFetchConfig(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	serviceRollout, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcherFetchConfig(t, serviceRollout, serviceConfig, serviceName)
	opts := options.DefaultConfigGeneratorOptions()
	opts.ServiceManagementURL = serviceManagementServer.URL
	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }

	scf, err := NewServiceConfigFetcher(&opts, serviceName, accessToken)
	if err != nil {
		t.Fatal(err)
	}

	scf.newRolloutId = func() (string, error) { return serviceRolloutId, nil }

	_test := func(configId string, wantServiceConfig *confpb.Service, wantError string) {
		getConfig, err := scf.FetchConfig(configId)
		if err != nil {
			if wantError == "" {
				t.Fatalf("fail to fetch config: %v", err)
			}

			if err.Error() != wantError {
				t.Fatalf("want error: %s, get error: %v", wantError, err)
			}
			return
		}

		if getConfig == nil {
			if wantServiceConfig != nil {
				t.Fatalf("want service config: %v, get service config: nil", wantServiceConfig)
			}
			return
		}

		if !proto.Equal(getConfig, serviceConfig) {
			t.Fatalf("want service config: %v, get service config: %v", serviceConfig, getConfig)
		}
	}

	// Test managed mode.
	_test("", serviceConfig, "")

	// Test fixed mode.
	serviceConfigId = "test-config-id-1"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	_test(serviceConfigId, serviceConfig, "")

	// When the service config id specified is same, no service config really fetched w
	_test(serviceConfigId, nil, "")

	// Error caused by failing to get new rollout id.
	serviceRolloutId = "test-rollout-id-2"
	serviceConfigId = "test-config-id-2"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	scf.newRolloutId = func() (string, error) { return "", fmt.Errorf("newRolloutIdError") }
	_test("", nil, "error occurred when checking new service rollout id: newRolloutIdError")

	// Error caused by problematic rollout.
	serviceRolloutId = "test-rollout-id-3"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	serviceRollout.GetTrafficPercentStrategy().Percentages = nil
	scf.newRolloutId = func() (string, error) { return serviceRolloutId, nil }
	_test("", nil, `problematic rollout rollout_id:"test-rollout-id-3" traffic_percent_strategy:<> `)

	// Not fetch if rollout id from servicecontrol report doesn't change.
	serviceRolloutId = "test-rollout-id-3"
	serviceConfigId = "test-config-id-31"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	_test("", nil, "")

	// Use the highest traffic config id.
	serviceRolloutId = "test-rollout-id-4"
	serviceConfigId = "test-config-id-41"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	serviceRollout.GetTrafficPercentStrategy().Percentages = map[string]float64{
		"test-config-id-41": 60,
		"test-config-id-42": 40,
	}
	_test("", serviceConfig, "")

	// Error caused by failing to get access token.
	serviceRolloutId = "test-rollout-id-5"
	serviceConfigId = "test-config-id-5"
	updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	scf.accessToken = func() (string, time.Duration, error) { return "", time.Duration(0), fmt.Errorf("accessTokenError") }
	_test("", nil, "fail to get access token: accessTokenError")
}

func TestServiceConfigFetcherSetFetchConfigTimer(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	serviceRollout, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcherFetchConfig(t, serviceRollout, serviceConfig, serviceName)
	opts := options.DefaultConfigGeneratorOptions()
	opts.ServiceManagementURL = serviceManagementServer.URL

	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }
	scf, err := NewServiceConfigFetcher(&opts, serviceName, accessToken)
	if err != nil {
		t.Fatal(err)
	}

	scf.newRolloutId = func() (string, error) { return serviceRolloutId, nil }
	cnt := 0
	scf.SetFetchConfigTimer(time.Millisecond*100, func(getService *confpb.Service) {
		cnt += 1
		if !proto.Equal(getService, serviceConfig) {
			t.Fatalf("want service %v, get service %v", serviceConfig, getService)
		}

		// Update service config so fetchConfig will do real fetching.
		serviceRolloutId = fmt.Sprintf("test-rollout-id-%v", cnt)
		serviceConfigId = fmt.Sprintf("test-config-id-%v", cnt)
		updateRolloutAndConfig(serviceRollout, serviceConfig, serviceRolloutId, serviceConfigId)
	})
	wantCnt := 10
	time.Sleep(time.Millisecond * time.Duration(100*wantCnt))

	// grace buffer
	if cnt < wantCnt-2 {
		t.Fatalf("want callback called by %v times, get %v times", wantCnt, cnt)
	}
}
