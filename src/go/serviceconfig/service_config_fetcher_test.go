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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func initServiceManagementForTestServiceConfigFetcher(t *testing.T,
	serviceRollouts *smpb.ListServiceRolloutsResponse, serviceConfig *confpb.Service, serviceName string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/v1/services/%s/rollouts", serviceName):
			resp, err := proto.Marshal(serviceRollouts)
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
			fmt.Printf(r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}

	}))
}

func genRolloutAndConfig(serviceRolloutId, serviceConfigId string) (*smpb.ListServiceRolloutsResponse, *confpb.Service) {
	serviceRollouts := &smpb.ListServiceRolloutsResponse{
		Rollouts: []*smpb.Rollout{
			{
				RolloutId: serviceRolloutId,
				Strategy: &smpb.Rollout_TrafficPercentStrategy_{
					TrafficPercentStrategy: &smpb.Rollout_TrafficPercentStrategy{
						Percentages: map[string]float64{
							serviceConfigId: 100,
						},
					},
				},
			},
		},
	}
	serviceConfig := &confpb.Service{
		Name: "foo",
		Id:   serviceConfigId,
	}
	return serviceRollouts, serviceConfig
}

func TestServiceConfigFetcherFetchConfig(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	serviceRollout, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcher(t, serviceRollout, serviceConfig, serviceName)
	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }

	scf := NewServiceConfigFetcher(&http.Client{}, serviceManagementServer.URL, "service-name", accessToken)

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

	// Test success of fetching the service config.
	_test(serviceConfigId, serviceConfig, "")

	// Test failure due to calling googleapis.
	callGoogleapis := util.CallGoogleapis
	util.CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, output proto.Message) error {
		return fmt.Errorf("error-from-CallGoogleapis")
	}
	_test(serviceConfigId, nil, "error-from-CallGoogleapis")
	util.CallGoogleapis = callGoogleapis
}

func TestServiceConfigFetcherLoadConfigIdFromRollouts(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	serviceRollouts, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcher(t, serviceRollouts, serviceConfig, serviceName)
	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }

	scf := NewServiceConfigFetcher(&http.Client{}, serviceManagementServer.URL, "service-name", accessToken)
	_test := func(wantConfigId string, wantError string) {
		getConfigId, err := scf.LoadConfigIdFromRollouts()

		if err != nil {
			if err.Error() != wantError {
				t.Errorf("want error: %s, get error: %v", wantError, err)
			}
			return
		}

		if getConfigId != wantConfigId {
			t.Errorf("wante configId: %s, get configId: %s", wantConfigId, getConfigId)
		}
	}

	// Test success of loading configId.
	_test(serviceConfigId, "")

	// Test getting the configId with highest traffic percentage.
	serviceRollouts.Rollouts[0].GetTrafficPercentStrategy().Percentages = map[string]float64{
		serviceConfigId:      20,
		"new-test-config-id": 80,
	}
	_test("new-test-config-id", "")

	// Test failure due to problematic rollouts.
	serviceRollouts.Rollouts = nil
	_test("", "problematic rollouts: ")

	// Test failure due to calling googleapis.
	callGoogleapis := util.CallGoogleapis
	util.CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, output proto.Message) error {
		return fmt.Errorf("error-from-CallGoogleapis")
	}
	_test("", "error-from-CallGoogleapis")
	util.CallGoogleapis = callGoogleapis
}
