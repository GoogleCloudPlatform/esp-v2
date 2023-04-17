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
	"google.golang.org/protobuf/proto"

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
	listServiceRolloutsResponse := &smpb.ListServiceRolloutsResponse{
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
	return listServiceRolloutsResponse, serviceConfig
}

func TestServiceConfigFetcherFetchConfig(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	serviceRollout, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcher(t, serviceRollout, serviceConfig, serviceName)
	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }

	scf := NewServiceConfigFetcher(&http.Client{}, serviceManagementServer.URL, "service-name", accessToken)

	testCase := []struct {
		desc                     string
		serviceConfigId          string
		callGoogleapisOverridden bool
		wantServiceConfig        *confpb.Service
		wantError                string
	}{
		{
			desc:              "Success of fetching the service config",
			serviceConfigId:   serviceConfigId,
			wantServiceConfig: serviceConfig,
		},
		{
			desc:                     "Failure due to calling googleapis",
			serviceConfigId:          serviceConfigId,
			callGoogleapisOverridden: true,
			wantError:                "error-from-CallGoogleapis",
		},
	}

	for _, tc := range testCase {
		_test := func(desc string, callGoogleapisOverridden bool, configId string, wantServiceConfig *confpb.Service, wantError string) {
			if callGoogleapisOverridden {
				oldCallGoogleapis := util.CallGoogleapis
				util.CallGoogleapisMu.Lock()
				util.CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, retryConfigs map[int]util.RetryConfig, output proto.Message) error {
					return fmt.Errorf("error-from-CallGoogleapis")
				}
				util.CallGoogleapisMu.Unlock()
				defer func() {
					util.CallGoogleapisMu.Lock()
					util.CallGoogleapis = oldCallGoogleapis
					util.CallGoogleapisMu.Unlock()
				}()
			}

			getConfig, err := scf.FetchConfig(configId)
			if err != nil {
				if wantError == "" {
					t.Fatalf("test(%s), fail to fetch config: %v", desc, err)
				}

				if err.Error() != wantError {
					t.Fatalf("test(%s), want error: %s, get error: %v", desc, wantError, err)
				}
				return
			}

			if getConfig == nil {
				if wantServiceConfig != nil {
					t.Fatalf("test(%s), want service config: %v, get service config: nil", desc, wantServiceConfig)
				}
				return
			}

			if !proto.Equal(getConfig, serviceConfig) {
				t.Fatalf("test(%s), want service config: %v, get service config: %v", desc, serviceConfig, getConfig)
			}
		}

		_test(tc.desc, tc.callGoogleapisOverridden, tc.serviceConfigId, tc.wantServiceConfig, tc.wantError)
	}
}

func TestServiceConfigFetcherLoadConfigIdFromRollouts(t *testing.T) {
	serviceName := "service-name"
	serviceRolloutId := "test-rollout-id"
	serviceConfigId := "test-config-id"
	listServiceRolloutsResponse, serviceConfig := genRolloutAndConfig(serviceRolloutId, serviceConfigId)

	serviceManagementServer := initServiceManagementForTestServiceConfigFetcher(t, listServiceRolloutsResponse, serviceConfig, serviceName)
	accessToken := func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil }

	scf := NewServiceConfigFetcher(&http.Client{}, serviceManagementServer.URL, "service-name", accessToken)

	testCase := []struct {
		desc                     string
		callGoogleapisOverridden bool
		serviceRollouts          []*smpb.Rollout
		wantConfigId             string
		wantError                string
	}{
		{
			desc:         "Success of fetching the service config",
			wantConfigId: serviceConfigId,
		},
		{
			desc: "Test getting the configId with highest traffic percentage",
			serviceRollouts: []*smpb.Rollout{
				{
					Strategy: &smpb.Rollout_TrafficPercentStrategy_{
						TrafficPercentStrategy: &smpb.Rollout_TrafficPercentStrategy{
							Percentages: map[string]float64{
								serviceConfigId:      20,
								"new-test-config-id": 80,
							},
						},
					},
				},
			},
			wantConfigId: "new-test-config-id",
		},
		{
			desc:            "failure due to problematic rollouts",
			serviceRollouts: []*smpb.Rollout{},
			wantConfigId:    serviceConfigId,
			wantError:       "problematic rollouts: ",
		},
		{
			desc:                     "Test failure due to calling googleapis",
			callGoogleapisOverridden: true,
			wantConfigId:             serviceConfigId,
			wantError:                "error-from-CallGoogleapis",
		},
	}

	for _, tc := range testCase {
		_test := func(desc string, callGoogleapisOverridden bool, serviceRollouts []*smpb.Rollout, wantConfigId string, wantError string) {
			if callGoogleapisOverridden {
				util.CallGoogleapisMu.Lock()
				oldCallGoogleapis := util.CallGoogleapis
				util.CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc util.GetAccessTokenFunc, retryConfigs map[int]util.RetryConfig, output proto.Message) error {
					return fmt.Errorf("error-from-CallGoogleapis")
				}
				util.CallGoogleapisMu.Unlock()
				defer func() {
					util.CallGoogleapisMu.Lock()
					util.CallGoogleapis = oldCallGoogleapis
					util.CallGoogleapisMu.Unlock()
				}()
			}

			if serviceRollouts != nil {
				oldserviceRollouts := listServiceRolloutsResponse.Rollouts
				listServiceRolloutsResponse.Rollouts = serviceRollouts
				defer func() { listServiceRolloutsResponse.Rollouts = oldserviceRollouts }()
			}

			getConfigId, err := scf.LoadConfigIdFromRollouts()

			if err != nil {
				if err.Error() != wantError {
					t.Errorf("test(%s), want error: %s, get error: %v", desc, wantError, err)
				}
				return
			}

			if getConfigId != wantConfigId {
				t.Errorf("test(%s),wante configId: %s, get configId: %s", desc, wantConfigId, getConfigId)
			}
		}

		_test(tc.desc, tc.callGoogleapisOverridden, tc.serviceRollouts, tc.wantConfigId, tc.wantError)
	}
}
