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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/api/annotations"

	gen "cloudesf.googlesource.com/gcpproxy/src/go/configgenerator"
	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	sm "github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	fetchConfigSuffix   = "/v1/services/$serviceName/configs/$configId?view=FULL"
	fetchRolloutsSuffix = "/v1/services/$serviceName/rollouts?filter=status=SUCCESS"
)

var (
	fetchConfigURL = func(serviceName, configID string) string {
		path := *flags.ServiceManagementURL + fetchConfigSuffix
		path = strings.Replace(path, "$serviceName", serviceName, 1)
		path = strings.Replace(path, "$configId", configID, 1)
		return path
	}
	fetchRolloutsURL = func(serviceName string) string {
		path := *flags.ServiceManagementURL + fetchRolloutsSuffix
		path = strings.Replace(path, "$serviceName", serviceName, 1)
		return path
	}
	checkNewRolloutInterval = 60 * time.Second
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName  string
	serviceInfo  *sc.ServiceInfo
	curRolloutID string
	curConfigID  string

	client              *http.Client
	cache               cache.SnapshotCache
	checkRolloutsTicker *time.Ticker
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager() (*ConfigManager, error) {
	var err error
	name := *flags.ServiceName
	if name == "" && *flags.CheckMetadata {
		name, err = fetchServiceName()
		if name == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server")
		}
	} else if name == "" && !*flags.CheckMetadata {
		return nil, fmt.Errorf("service name is not specified")
	}
	rolloutStrategy := *flags.RolloutStrategy
	// try to fetch from metadata, if not found, set to fixed instead of throwing an error
	if rolloutStrategy == "" && *flags.CheckMetadata {
		rolloutStrategy, _ = fetchRolloutStrategy()
	}
	if rolloutStrategy == "" {
		rolloutStrategy = ut.FixedRolloutStrategy
	}
	if !(rolloutStrategy == ut.FixedRolloutStrategy || rolloutStrategy == ut.ManagedRolloutStrategy) {
		return nil, fmt.Errorf(`failed to set rollout strategy. It must be either "managed" or "fixed"`)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	m := &ConfigManager{
		serviceName: name,
		client:      &http.Client{Transport: tr},
	}

	m.cache = cache.NewSnapshotCache(true, m, m)

	if rolloutStrategy == ut.ManagedRolloutStrategy {
		// try to fetch rollouts and get newest config, if failed, NewConfigManager exits with failure
		if err := m.loadConfigFromRollouts(); err != nil {
			return nil, err
		}
	} else {
		// rollout strategy is fixed mode
		configID := *flags.ConfigID
		if configID == "" {
			if *flags.CheckMetadata {
				configID, err = fetchConfigId()
				if configID == "" || err != nil {
					return nil, fmt.Errorf("failed to read metadata with key endpoints-service-version from metadata server")
				}
			} else {
				return nil, fmt.Errorf("service config id is not specified")
			}
		}
		m.curConfigID = configID
		if err := m.updateSnapshot(); err != nil {
			return nil, err
		}
	}
	glog.Infof("create new ConfigManager for service (%v) with configuration id (%v), %v rollout strategy",
		m.serviceName, m.curConfigID, rolloutStrategy)

	if rolloutStrategy == ut.ManagedRolloutStrategy {
		go func() {
			glog.Infof("start checking new rollouts every %v seconds", checkNewRolloutInterval)
			m.checkRolloutsTicker = time.NewTicker(checkNewRolloutInterval)
			for range m.checkRolloutsTicker.C {
				m.Infof("check new rollouts for service %v", m.serviceName)
				// only log error and keep checking when fetching rollouts and getting newest config fail
				if err := m.loadConfigFromRollouts(); err != nil {
					glog.Errorf("error occurred when checking new rollouts, %v", err)
				}
			}
		}()
	}
	return m, nil
}

func (m *ConfigManager) loadConfigFromRollouts() error {
	var err error
	var listServiceRolloutsResponse *sm.ListServiceRolloutsResponse
	listServiceRolloutsResponse, err = m.fetchRollouts()
	if err != nil {
		return fmt.Errorf("fail to get rollouts, %s", err)
	}
	m.Infof("get rollouts %v", listServiceRolloutsResponse)
	if len(listServiceRolloutsResponse.Rollouts) == 0 {
		return fmt.Errorf("no active rollouts")
	}
	newRolloutID := listServiceRolloutsResponse.Rollouts[0].RolloutId
	if m.curRolloutID == newRolloutID {
		return nil
	}
	m.curRolloutID = newRolloutID
	m.Infof("found new rollout id %v for service %v, %v", m.curRolloutID, m.serviceName)

	trafficPercentStrategy := listServiceRolloutsResponse.Rollouts[0].GetTrafficPercentStrategy()
	trafficPercentMap := trafficPercentStrategy.GetPercentages()
	if len(trafficPercentMap) == 0 {
		return fmt.Errorf("no active rollouts")
	}
	var newConfigID string
	currentMaxPercent := 0.0
	// take config ID with max traffic percent as new config ID
	for k, v := range trafficPercentMap {
		if v > currentMaxPercent {
			newConfigID = k
			currentMaxPercent = v
		}
	}
	if newConfigID == m.curConfigID {
		m.Infof("no new configuration to load for service %v, current configuration id %v", m.serviceName, m.curConfigID)
		return nil
	}
	if !(math.Abs(100.0-currentMaxPercent) < 1e-9) {
		glog.Warningf("though traffic percentage of configuration %v is %v%%, set it to 100%%", newConfigID, currentMaxPercent)
	}
	m.curConfigID = newConfigID
	m.Infof("found new configuration id %v for service %v", m.curConfigID, m.serviceName)
	return m.updateSnapshot()
}

// updateSnapshot should be called when starting up the server.
// It calls ServiceManager Server to fetch the service configuration in order
// to dynamically configure Envoy.
func (m *ConfigManager) updateSnapshot() error {
	serviceConfig, err := m.fetchConfig(m.curConfigID)
	if err != nil {
		return fmt.Errorf("fail to fetch service config, %s", err)
	}
	m.serviceInfo, err = sc.NewServiceInfoFromServiceConfig(serviceConfig, m.curConfigID)
	if err != nil {
		return fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	m.serviceInfo.GcpAttributes = fetchGCPAttributes()

	snapshot, err := m.makeSnapshot()
	if err != nil {
		return fmt.Errorf("fail to make a snapshot, %s", err)
	}
	return m.cache.SetSnapshot(*flags.Node, *snapshot)
}

func (m *ConfigManager) makeSnapshot() (*cache.Snapshot, error) {
	m.Infof("making configuration for api: %v", m.serviceInfo.ApiName)

	var backendProtocol ut.BackendProtocol
	switch strings.ToLower(*flags.BackendProtocol) {
	case "http1":
		backendProtocol = ut.HTTP1
	case "http2":
		backendProtocol = ut.HTTP2
	case "grpc":
		backendProtocol = ut.GRPC
	default:
		return nil, fmt.Errorf(`unknown backend protocol, should be one of "grpc", "http1" or "http2"`)
	}

	var endpoints, routes, clusters []cache.Resource
	// TODO(jilinxia): move all clusters into cluster_generator.go
	backendCluster, err := gen.MakeBackendCluster(m.serviceInfo, backendProtocol)
	if err != nil {
		return nil, err
	}
	if backendCluster != nil {
		clusters = append(clusters, backendCluster)
	}

	// Note: makeServiceControlCluster should be called before makeListener
	// as makeServiceControlFilter is using m.serviceControlURI assigned by
	// makeServiceControlCluster
	scCluster, err := gen.MakeServiceControlCluster(m.serviceInfo)
	if err != nil {
		return nil, err
	}
	if scCluster != nil {
		clusters = append(clusters, scCluster)
	}

	brClusters, err := gen.MakeBackendRoutingClusters(m.serviceInfo)
	if err != nil {
		return nil, err
	}
	if brClusters != nil {
		clusters = append(clusters, brClusters...)
	}

	m.Infof("adding Listener configuration for api: %v", m.serviceInfo.Name)
	serverlistener, httpManager, err := gen.MakeListener(m.serviceInfo, backendProtocol)
	if err != nil {
		return nil, err
	}

	// HTTP filter configuration
	httpFilterConfig, err := util.MessageToStruct(httpManager)
	if err != nil {
		return nil, err
	}
	serverlistener.FilterChains = []listener.FilterChain{{
		Filters: []listener.Filter{{
			Name:       util.HTTPConnectionManager,
			ConfigType: &listener.Filter_Config{httpFilterConfig},
		}}}}

	snapshot := cache.NewSnapshot(m.curConfigID, endpoints, clusters, routes, []cache.Resource{serverlistener})
	glog.Infof("Envoy Dynamic Configuration is cached for service: %v", m.serviceName)
	return &snapshot, nil
}

// ID Implements the ID method for HashNode interface.
func (m *ConfigManager) ID(node *core.Node) string {
	return node.GetId()
}

// Infof implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) {
	outputString, _ := json.MarshalIndent(args, "", "   ")
	glog.Infof(format, string(outputString))
}

// Errorf implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

// Cache returns snapshot cache.
func (m *ConfigManager) Cache() cache.Cache { return m.cache }

// TODO(jcwang) cleanup here. This function is redundant.
func (m *ConfigManager) fetchRollouts() (*sm.ListServiceRolloutsResponse, error) {
	token, _, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}

	return callServiceManagementRollouts(fetchRolloutsURL(m.serviceName), token, m.client)
}

func (m *ConfigManager) fetchConfig(configId string) (*conf.Service, error) {
	token, _, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}

	return callServiceManagement(fetchConfigURL(m.serviceName, configId), token, m.client)
}

// Helper to convert Json string to protobuf.Any.
type funcResolver func(url string) (proto.Message, error)

func (fn funcResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var callServiceManagementRollouts = func(path, token string, client *http.Client) (*sm.ListServiceRolloutsResponse, error) {
	var err error
	var resp *http.Response
	if resp, err = callWithAccessToken(path, token, client); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	unmarshaler := &jsonpb.Unmarshaler{}
	var rolloutsResponse sm.ListServiceRolloutsResponse
	if err = unmarshaler.Unmarshal(resp.Body, &rolloutsResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal ListServiceRolloutsResponse: %s", err)
	}
	return &rolloutsResponse, nil
}

var callServiceManagement = func(path, token string, client *http.Client) (*conf.Service, error) {
	var err error
	var resp *http.Response
	if resp, err = callWithAccessToken(path, token, client); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resolver := funcResolver(func(url string) (proto.Message, error) {
		switch url {
		case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
			return new(sm.ConfigFile), nil
		case "type.googleapis.com/google.api.HttpRule":
			return new(annotations.HttpRule), nil
		case "type.googleapis.com/google.protobuf.BoolValue":
			return new(types.BoolValue), nil
		default:
			return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
		}
	})
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: resolver,
	}
	var serviceConfig conf.Service
	if err = unmarshaler.Unmarshal(resp.Body, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig: %s", err)
	}
	return &serviceConfig, nil
}

var callWithAccessToken = func(path, token string, client *http.Client) (*http.Response, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http call to %s returns not 200 OK: %v", path, resp.Status)
	}
	return resp, nil
}
