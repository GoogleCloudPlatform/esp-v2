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
)


func getServiceConfig(urlPrefix string, configId string) (*conf.Service, error) {
	url := urlPrefix + "/configs/" + configId
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


	gotServiceConfig, err := getServiceConfig(urlPrefix, serviceConfig.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(gotServiceConfig, serviceConfig) {
		t.Errorf("The got service config is different than what we what,\ngot: %v,\nwanted: %v", gotServiceConfig, serviceConfig)
	}

}
