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
	"flag"

	"fmt"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/glog"

	gen "cloudesf.googlesource.com/gcpproxy/src/go/configgenerator"
	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
)

var (
	// These flags are used by config manage only.
	ServiceName     = flag.String("service", "", "endpoint service name")
	ServiceConfigID = flag.String("service_config_id", "", "initial service config id")
	RolloutStrategy = flag.String("rollout_strategy", "fixed", `service config rollout strategy, must be either "managed" or "fixed"`)
	CheckMetadata   = flag.Bool("check_metadata", false, `enable fetching service name, config ID and rollout strategy from service metadata server`)
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName  string
	serviceInfo  *sc.ServiceInfo
	curRolloutID string
	curConfigID  string

	cache               cache.SnapshotCache
	checkRolloutsTicker *time.Ticker
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager() (*ConfigManager, error) {
	var err error
	name := *ServiceName
	checkMetadata := *CheckMetadata
	if name == "" && checkMetadata {
		name, err = fetchServiceName()
		if name == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server")
		}
	} else if name == "" && !checkMetadata {
		return nil, fmt.Errorf("service name is not specified")
	}
	rolloutStrategy := *RolloutStrategy
	// try to fetch from metadata, if not found, set to fixed instead of throwing an error
	if rolloutStrategy == "" && checkMetadata {
		rolloutStrategy, _ = fetchRolloutStrategy()
	}
	if rolloutStrategy == "" {
		rolloutStrategy = ut.FixedRolloutStrategy
	}
	if !(rolloutStrategy == ut.FixedRolloutStrategy || rolloutStrategy == ut.ManagedRolloutStrategy) {
		return nil, fmt.Errorf(`failed to set rollout strategy. It must be either "managed" or "fixed"`)
	}

	m := &ConfigManager{
		serviceName: name,
	}

	m.cache = cache.NewSnapshotCache(true, m, m)

	if rolloutStrategy == ut.ManagedRolloutStrategy {
		// try to fetch rollouts and get newest config, if failed, NewConfigManager exits with failure
		newRolloutID, newConfigID, err := loadConfigFromRollouts(m.serviceName, m.curRolloutID, m.curConfigID)
		if err != nil {
			return nil, err
		}
		if m.curRolloutID != newRolloutID && m.curConfigID != newConfigID {
			m.curRolloutID = newRolloutID
			m.curConfigID = newConfigID
			if err := m.updateSnapshot(); err != nil {
				return nil, err
			}
		}
	} else {
		// rollout strategy is fixed mode
		configID := *ServiceConfigID
		if configID == "" {
			if checkMetadata {
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
				newRolloutID, newConfigID, err := loadConfigFromRollouts(m.serviceName, m.curRolloutID, m.curConfigID)
				if err != nil {
					glog.Errorf("error occurred when checking new rollouts, %v", err)
				}
				if m.curRolloutID != newRolloutID && m.curConfigID != newConfigID {
					m.curRolloutID = newRolloutID
					m.curConfigID = newConfigID
					if err := m.updateSnapshot(); err != nil {
						glog.Errorf("error occurred when checking new rollouts, %v", err)
					}
				}
			}
		}()
	}
	return m, nil
}

// updateSnapshot should be called when starting up the server.
// It calls ServiceManager Server to fetch the service configuration in order
// to dynamically configure Envoy.
func (m *ConfigManager) updateSnapshot() error {
	serviceConfig, err := fetchConfig(m.serviceName, m.curConfigID)
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

	var clusterResources, endpoints, routes []cache.Resource
	clusters, err := gen.MakeClusters(m.serviceInfo)
	if err != nil {
		return nil, err
	}
	for i, _ := range clusters {
		clusterResources = append(clusterResources, &clusters[i])
	}

	m.Infof("adding Listener configuration for api: %v", m.serviceInfo.Name)
	listener, err := gen.MakeListeners(m.serviceInfo)
	if err != nil {
		return nil, err
	}

	snapshot := cache.NewSnapshot(m.curConfigID, endpoints, clusterResources, routes, []cache.Resource{listener})
	m.Infof("Envoy Dynamic Configuration is cached for service: %v", m.serviceName)
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
