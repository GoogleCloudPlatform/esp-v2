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

package configmanager

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/glog"

	gen "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator"
	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/serviceconfig"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	// These flags are used by config manage only.
	checkNewRolloutInterval = flag.Duration("check_rollout_interval", 60*time.Second, `the interval periodically to call servicemanagment to check the latest rolloutil.`)
	CheckMetadata           = flag.Bool("check_metadata", false, `enable fetching service name, config ID and rollout strategy from service metadata server`)
	RolloutStrategy         = flag.String("rollout_strategy", "fixed", `service config rollout strategy, must be either "managed" or "fixed"`)
	ServiceConfigID         = flag.String("service_config_id", "", "initial service config id")
	ServiceName             = flag.String("service", "", "endpoint service name")
	ServicePath             = flag.String("service_json_path", "", `file path to the endpoint service config.
					When this flag is used, fixed rollout_strategy will be used,
					GCP metadata server will not be called to fetch access token, and
					following flags will be ignored; --service_config_id, --service,
					--rollout_strategy`)
)

// Config Manager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName                   string
	serviceInfo                   *configinfo.ServiceInfo
	envoyConfigOptions            options.ConfigGeneratorOptions
	configIdFromServiceConfigFile string

	cache               cache.SnapshotCache
	checkRolloutsTicker *time.Ticker

	metadataFetcher      *metadata.MetadataFetcher
	serviceConfigFetcher *sc.ServiceConfigFetcher
}

// NewConfigManager creates new instance of Config Manager.
// mf is set to nil on non-gcp deployments
func NewConfigManager(mf *metadata.MetadataFetcher, opts options.ConfigGeneratorOptions) (*ConfigManager, error) {
	m := &ConfigManager{
		metadataFetcher:    mf,
		envoyConfigOptions: opts,
	}
	m.cache = cache.NewSnapshotCache(true, m, m)

	// If service config is provided as a file, just use it and disable managed rollout
	if *ServicePath != "" {
		// Following flags will not be used
		if *ServiceName != "" {
			glog.Infof("flag --service is ignored when --service_json_path is specified.")
		}
		if *ServiceConfigID != "" {
			glog.Infof("flag --service_config_id is ignored when --service_json_path is specified.")
		}
		if *RolloutStrategy != "fixed" {
			glog.Infof("flag --rollout_strategy will be fixed when --service_json_path is specified.")
		}

		if err := m.readAndApplyServiceConfig(*ServicePath); err != nil {
			return nil, err
		}

		glog.Infof("create new Config Manager from static service config json file at %v", *ServicePath)
		return m, nil
	}

	m.serviceName = *ServiceName
	checkMetadata := *CheckMetadata
	var err error

	if m.serviceName == "" && checkMetadata && mf != nil {
		m.serviceName, err = mf.FetchServiceName()
		if m.serviceName == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server")
		}
	} else if m.serviceName == "" && !checkMetadata {
		return nil, fmt.Errorf("service name is not specified, required because metadata fetching is disabled")
	} else if m.serviceName == "" && mf == nil {
		return nil, fmt.Errorf("service name is not specified, required on a non-gcp deployment")
	}
	rolloutStrategy := *RolloutStrategy
	// try to fetch from metadata, if not found, set to fixed instead of throwing an error
	if rolloutStrategy == "" && checkMetadata && mf != nil {
		rolloutStrategy, _ = mf.FetchRolloutStrategy()
	}
	if rolloutStrategy == "" {
		rolloutStrategy = util.FixedRolloutStrategy
	}
	if !(rolloutStrategy == util.FixedRolloutStrategy || rolloutStrategy == util.ManagedRolloutStrategy) {
		return nil, fmt.Errorf(`failed to set rollout strategy. It must be either "managed" or "fixed"`)
	}

	if rolloutStrategy == util.ManagedRolloutStrategy {
		if m.serviceConfigFetcher, err = sc.NewServiceConfigFetcher(mf, opts, m.serviceName, ""); err != nil {
			return nil, fmt.Errorf(`failed to create https client to call ServiceManagement service, got error: %v`, err)

		}
	} else {
		// rollout strategy is fixed mode
		configId := *ServiceConfigID
		if configId == "" {
			if checkMetadata && mf != nil {
				configId, err = mf.FetchConfigId()
				if configId == "" || err != nil {
					return nil, fmt.Errorf("failed to read metadata with key endpoints-service-version from metadata server")
				}
			} else if !checkMetadata {
				return nil, fmt.Errorf("service config id is not specified, required because metadata fetching is disabled")
			} else if mf == nil {
				return nil, fmt.Errorf("service config id is not specified, required on a non-gcp deployment")
			}
		}

		if m.serviceConfigFetcher, err = sc.NewServiceConfigFetcher(mf, opts, m.serviceName, configId); err != nil {
			return nil, fmt.Errorf(`failed to create https client to call ServiceManagement service, got error: %v`, err)

		}
	}

	if serviceConfig, err := m.serviceConfigFetcher.CurServiceConfig(); err != nil {
		return nil, err
	} else if err = m.applyServiceConfig(serviceConfig); err != nil {
		return nil, err
	}

	glog.Infof("create new Config Manager for service (%v) with configuration id (%v), %v rollout strategy",
		m.serviceName, m.curConfigId(), rolloutStrategy)

	if rolloutStrategy == util.ManagedRolloutStrategy {
		m.serviceConfigFetcher.SetFetchConfigTimer(checkNewRolloutInterval, func(serviceConfig *confpb.Service) {
			err := m.applyServiceConfig(serviceConfig)
			if err != nil {
				glog.Errorf("error occurred when checking new rollouts, %v", err)
			}
		})
	}
	return m, nil
}

func (m *ConfigManager) readAndApplyServiceConfig(servicePath string) error {
	config, err := ioutil.ReadFile(servicePath)
	if err != nil {
		return fmt.Errorf("fail to read service config file: %s, error: %s", servicePath, err)
	}

	serviceConfig, err := util.UnmarshalServiceConfig(bytes.NewReader(config))
	if err != nil {
		return fmt.Errorf("fail to unmarshal service config: %v, error: %s", config, err)
	}

	m.serviceName = serviceConfig.GetName()
	m.configIdFromServiceConfigFile = serviceConfig.GetId()

	return m.applyServiceConfig(serviceConfig)
}

func (m *ConfigManager) applyServiceConfig(serviceConfig *confpb.Service) error {
	var err error
	m.serviceInfo, err = configinfo.NewServiceInfoFromServiceConfig(serviceConfig, m.curConfigId(), m.envoyConfigOptions)
	if err != nil {
		return fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	if m.metadataFetcher != nil {
		attrs, err := m.metadataFetcher.FetchGCPAttributes()
		if err != nil {
			m.Infof("metadata server was not reached, skipping GCP Attributes")
		} else {
			m.serviceInfo.GcpAttributes = attrs
		}
	}

	snapshot, err := m.makeSnapshot()
	if err != nil {
		return fmt.Errorf("fail to make a snapshot, %s", err)
	}
	return m.cache.SetSnapshot(m.envoyConfigOptions.Node, *snapshot)
}

func (m *ConfigManager) makeSnapshot() (*cache.Snapshot, error) {
	m.Infof("making configuration for api: %v", m.serviceInfo.Name)

	var clusterResources, endpoints, runtimes, routes, listenerResources []cache.Resource
	clusters, err := gen.MakeClusters(m.serviceInfo)
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		clusterResources = append(clusterResources, clusters[i])
	}

	m.Infof("adding Listeners configuration for api: %v", m.serviceInfo.Name)
	listeners, err := gen.MakeListeners(m.serviceInfo)
	if err != nil {
		return nil, err
	}
	for _, lis := range listeners {
		listenerResources = append(listenerResources, lis)
	}

	snapshot := cache.NewSnapshot(m.curConfigId(), endpoints, clusterResources, routes, listenerResources, runtimes)
	m.Infof("Envoy Dynamic Configuration is cached for service: %v", m.serviceName)
	return &snapshot, nil
}

func (m *ConfigManager) curConfigId() string {
	if m.configIdFromServiceConfigFile != "" {
		return m.configIdFromServiceConfigFile
	}
	return m.serviceConfigFetcher.CurConfigId()
}

func (m *ConfigManager) curRolloutId() string {
	return m.serviceConfigFetcher.CurRolloutId()
}

func (m *ConfigManager) ID(node *corepb.Node) string {
	return node.GetId()
}

// Infof implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) {
	glog.Infof(format, args...)
}

// Errorf implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

// Cache returns snapshot cache.
func (m *ConfigManager) Cache() cache.Cache { return m.cache }
