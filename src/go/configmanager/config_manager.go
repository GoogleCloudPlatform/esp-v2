// Copyright 2018 Google Cloud Platform Proxy Authors
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

package configmanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	// "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"cloudesf.googlesource.com/gcpproxy/src/go/proto/google/api"
)

var (
	fetchConfigURL = "https://servicemanagement.googleapis.com/v1/services/$serviceName/configs/$configId"
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName string
	rolloutInfo *rolloutInfo
	client      *http.Client
}

type rolloutInfo struct {
	// TODO(jilinxia): change Service to Bootstrap.
	configs map[string]*api.Service
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager(name string) (*ConfigManager, error) {
	manager := &ConfigManager{
		serviceName: name,
		client:      http.DefaultClient,
		rolloutInfo: &rolloutInfo{
			configs: make(map[string]*api.Service),
		},
	}

	return manager, nil
}

func (m *ConfigManager) Init(configID string) error {
	serviceConfig, err := m.fetchConfig(configID)
	if err != nil {
		// TODO(jilinxia): changes error generation
		return fmt.Errorf("fail to initialize config manager, %s", err)
	}
	m.rolloutInfo.configs[configID] = serviceConfig
	return nil
}

// TODO(jilinxia): Implement the translation.
func (m *ConfigManager) writeBootstrap() string {
	return ""
}

func (m *ConfigManager) fetchConfig(configId string) (*api.Service, error) {
	token, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}
	path := strings.Replace(fetchConfigURL, "$configId", configId, -1)

	body, err := callServiceManagement(path, m.serviceName, token, m.client)
	if err != nil {
		return nil, fmt.Errorf("fail to call service management server to get config(%s) of service %s", configId, m.serviceName)
	}
	var serviceConfig api.Service
	if err = json.Unmarshal(body, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig")
	}
	return &serviceConfig, nil
}

var callServiceManagement = func(path, serviceName, token string, client *http.Client) ([]byte, error) {
	path = strings.Replace(path, "$serviceName", serviceName, -1)
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
