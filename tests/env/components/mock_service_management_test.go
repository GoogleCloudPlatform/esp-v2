// Copyright 2019 Google LLC
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

package components

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golang/protobuf/proto"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func getRolloutID(urlPrefix string) (string, error) {
	url := urlPrefix + "/rollouts?filter=status=SUCCESS"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Failed in request: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed in read response: %v", err)
	}
	rolloutsResponse := new(smpb.ListServiceRolloutsResponse)
	if err := proto.Unmarshal(body, rolloutsResponse); err != nil {
		return "", fmt.Errorf("fail to unmarshal ListServiceRolloutsResponse: %s", err)
	}

	rolloutID := rolloutsResponse.Rollouts[0].RolloutId
	return rolloutID, nil
}

func getServiceConfig(urlPrefix string, rolloutID string) (*conf.Service, error) {
	url := urlPrefix + "/configs/" + rolloutID
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed in request: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed in read response: %v", err)
	}
	service := new(conf.Service)
	if err := proto.Unmarshal(body, service); err != nil {
		return nil, fmt.Errorf("fail to unmarshal Service: %v", err)
	}
	return service, nil
}

func TestMockServiceManagement(t *testing.T) {
	serviceConfig := &conf.Service{Name: "foo", Id: "999"}

	s := NewMockServiceMrg(serviceConfig.Name, serviceConfig)
	urlPrefix := s.Start() + "/v1/services/" + serviceConfig.Name
	rolloutID, err := getRolloutID(urlPrefix)
	if err != nil {
		t.Errorf("TestMockServiceManagement: %v", err)
	}

	gotServiceConfig, err := getServiceConfig(urlPrefix, rolloutID)
	if !proto.Equal(gotServiceConfig, serviceConfig) {
		t.Errorf("The got service config is different than what we what,\ngot: %v,\nwanted: %v", gotServiceConfig, serviceConfig)
	}
	newRollID, err := getRolloutID(urlPrefix)
	if newRollID != rolloutID {
		t.Errorf("TestMockServiceManagement: the rolloutID should be unchanged, got: %v, wanted: %v", newRollID, rolloutID)
	}

	serviceConfig.Id = "1000"
	latestRolloutID, err := getRolloutID(urlPrefix)
	if latestRolloutID == rolloutID {
		t.Errorf("TestMockServiceManagement: the rolloutID should have been updated, got: %v, wanted: %v", latestRolloutID, rolloutID)
	}

	gotServiceConfig, err = getServiceConfig(urlPrefix, latestRolloutID)
	if !proto.Equal(gotServiceConfig, serviceConfig) {
		t.Errorf("The got service config is different than what we what,\ngot: %v,\nwanted: %v", gotServiceConfig, serviceConfig)
	}
}
