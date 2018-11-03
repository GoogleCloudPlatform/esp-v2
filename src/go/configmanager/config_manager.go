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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/proto/google/api"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	tc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/golang/glog"
)

const (
	listenerAddress = "0.0.0.0"
	listenerPort    = 8080
)

var (
	fetchConfigURL = "https://servicemanagement.googleapis.com/v1/services/$serviceName/configs/$configId?view=FULL"
	node           = "fake node, need to figure out what node name is."
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName string
	configID    string
	client      *http.Client
	cache       cache.SnapshotCache
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager(name, configID string) (*ConfigManager, error) {
	m := &ConfigManager{
		serviceName: name,
		client:      http.DefaultClient,
		configID:    configID,
	}
	m.cache = cache.NewSnapshotCache(true, m, m)
	if err := m.init(); err != nil {
		return nil, err
	}
	return m, nil
}

// init should be called when starting up the server.
// It calls ServiceManager Server to fetch the service configuration in order
// to dynamically configure Envoy.
func (m *ConfigManager) init() error {
	serviceConfig, err := m.fetchConfig(m.configID)
	if err != nil {
		// TODO(jilinxia): changes error generation
		return fmt.Errorf("fail to initialize config manager, %s", err)
	}
	// m.rolloutInfo.configs[configID] = serviceConfig
	snapshot, err := m.makeSnapshot(serviceConfig)
	if err != nil {
		return errors.New("fail to make a snapshot")
	}
	m.cache.SetSnapshot(node, *snapshot)
	return nil
}

func (m *ConfigManager) makeSnapshot(serviceConfig *api.Service) (*cache.Snapshot, error) {
	var clusters, endpoints, routes []cache.Resource
	serverlistener, httpManager := m.makeListener(serviceConfig)

	// HTTP filter configuration
	httpFilterConfig, err := util.MessageToStruct(httpManager)
	if err != nil {
		return nil, err
	}

	serverlistener.FilterChains = []listener.FilterChain{{
		Filters: []listener.Filter{{
			Name:   util.HTTPConnectionManager,
			Config: httpFilterConfig,
		}}}}

	snapshot := cache.NewSnapshot(m.configID, endpoints, clusters, routes, []cache.Resource{serverlistener})
	return &snapshot, nil
}

func (m *ConfigManager) makeListener(serviceConfig *api.Service) (*v2.Listener, *hcm.HttpConnectionManager) {
	httpFilters := []*hcm.HttpFilter{}
	// Add gRPC transcode filter config.
	transcodeConfig := &tc.GrpcJsonTranscoder{
		DescriptorSet: &tc.GrpcJsonTranscoder_ProtoDescriptor{
			// TODO(jilinxia): pass in proto descriptor
		},
		Services: []string{serviceConfig.Apis[0].Name},
	}
	transcodeConfigStruct, _ := util.MessageToStruct(transcodeConfig)
	transcodeFilter := &hcm.HttpFilter{
		Name:   util.GRPCJSONTranscoder,
		Config: transcodeConfigStruct,
	}

	httpFilters = append(httpFilters, transcodeFilter)


	// TODO(jilinxia): Add Service control filter config.
	// TODO(jilinxia): Add JWT filter config.

	return &v2.Listener{
			Address: core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
				Address:       listenerAddress,
				PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(listenerPort)}}}},
		}, &hcm.HttpConnectionManager{
			CodecType:  hcm.AUTO,
			StatPrefix: "ingress_http",
			RouteSpecifier: &hcm.HttpConnectionManager_Rds{
				Rds: &hcm.Rds{ConfigSource: core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}},
				}},
			},
			HttpFilters: httpFilters,
		}
}

// Implements the ID method for HashNode interface.
func (m *ConfigManager) ID(node *core.Node) string {
	return node.GetId()
}

// Implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) { glog.Infof(format, args...) }

// Implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

func (m *ConfigManager) Cache() cache.Cache { return m.cache }

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
