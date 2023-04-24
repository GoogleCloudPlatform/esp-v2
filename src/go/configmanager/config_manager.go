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
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/tokengenerator"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/glog"

	gen "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator"
	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/serviceconfig"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	rsrc "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	// These flags are used by config manage only.
	checkNewRolloutInterval = flag.Duration("check_rollout_interval", 60*time.Second, `the interval periodically to call servicemanagment to check the latest rolloutil.`)
	CheckMetadata           = flag.Bool("check_metadata", false, `enable fetching service name, config ID and rollout strategy from service metadata server`)
	RolloutStrategy         = flag.String("rollout_strategy", "fixed", `service config rollout strategy, must be either "managed" or "fixed"`)
	ServiceConfigId         = flag.String("service_config_id", "", "initial service config id")
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
	serviceName        string
	envoyConfigOptions options.ConfigGeneratorOptions
	serviceInfo        *configinfo.ServiceInfo
	cache              cache.SnapshotCache

	metadataFetcher         *metadata.MetadataFetcher
	serviceConfigFetcher    *sc.ServiceConfigFetcher
	rolloutIdChangeDetector *sc.RolloutIdChangeDetector

	curServiceConfig *confpb.Service
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
		if *ServiceConfigId != "" {
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
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server: %v", err)
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

	// when --non_gcp  is set, instance metadata server(imds) is not defined. So
	// accessToken is unavailable from imds and --service_account_key must be
	// set to generate accessToken.
	// The inverse is not true. We can still use IMDS on GCP when service account key is specified.
	if mf == nil && opts.ServiceAccountKey == "" {
		return nil, fmt.Errorf("if flag --non_gcp is specified, flag --service_account_key must be specified")
	}

	accessToken := func() (string, time.Duration, error) {
		if opts.ServiceAccountKey != "" {
			return tokengenerator.GenerateAccessTokenFromFile(opts.ServiceAccountKey)
		}
		return mf.FetchAccessToken()
	}

	client, err := httpsClient(opts)
	if err != nil {
		return nil, fmt.Errorf("fail to init httpsClient: %v", err)
	}

	m.serviceConfigFetcher = sc.NewServiceConfigFetcher(client, opts.ServiceManagementURL,
		m.serviceName, accessToken)

	configId := ""
	if rolloutStrategy == util.FixedRolloutStrategy {
		configId = *ServiceConfigId
		if configId == "" {
			if mf == nil {
				return nil, fmt.Errorf("service config id is not specified, required on a non-gcp deployment")
			}

			if !checkMetadata {
				return nil, fmt.Errorf("service config id is not specified, required because metadata fetching is disabled")
			}

			configId, err = mf.FetchConfigId()
			if configId == "" || err != nil {
				return nil, fmt.Errorf("failed to read metadata with key endpoints-service-version from metadata server: %v", err)
			}
		}
	} else if rolloutStrategy == util.ManagedRolloutStrategy {
		configId, err = m.serviceConfigFetcher.LoadConfigIdFromRollouts()
		if err != nil {
			return nil, err
		}
	}

	if err = m.fetchAndApplyServiceConfig(configId); err != nil {
		return nil, fmt.Errorf("fail to fetch and apply the startup service config, %v", err)
	}

	if rolloutStrategy == util.ManagedRolloutStrategy {
		m.rolloutIdChangeDetector = sc.NewRolloutIdChangeDetector(client, opts.ServiceControlURL, m.serviceName, accessToken)
		m.rolloutIdChangeDetector.SetDetectRolloutIdChangeTimer(*checkNewRolloutInterval, func() {
			latestConfigId, err := m.serviceConfigFetcher.LoadConfigIdFromRollouts()
			if err != nil {
				glog.Errorf("error occurred when getting configId by fetching rollout, %v", err)
				return
			}

			if err = m.fetchAndApplyServiceConfig(latestConfigId); err != nil {
				glog.Errorf("error occurred when fetching and applying new service config, %v", err)
			}
		})
	}

	glog.Infof("create new Config Manager for service (%v) with configuration id (%v), %v rollout strategy",
		m.serviceName, m.curConfigId(), rolloutStrategy)
	return m, nil
}

func (m *ConfigManager) fetchAndApplyServiceConfig(latestConfigId string) error {
	if latestConfigId == m.curConfigId() {
		glog.Infof("no new configuration to load for service %v, current configuration Id %v", m.serviceName, m.curConfigId())
		return nil
	}

	serviceConfig, err := m.serviceConfigFetcher.FetchConfig(latestConfigId)
	if err != nil {
		return err
	}

	return m.applyServiceConfig(serviceConfig)
}

func (m *ConfigManager) readAndApplyServiceConfig(servicePath string) error {
	config, err := ioutil.ReadFile(servicePath)
	if err != nil {
		return fmt.Errorf("fail to read service config file: %s, error: %s", servicePath, err)
	}

	serviceConfig, err := util.UnmarshalServiceConfig(config)
	if err != nil {
		return fmt.Errorf("fail to unmarshal service config with error: %s", err)
	}

	m.serviceName = serviceConfig.GetName()
	return m.applyServiceConfig(serviceConfig)
}

func (m *ConfigManager) applyServiceConfig(serviceConfig *confpb.Service) error {
	if serviceConfig == nil {
		return fmt.Errorf("applid service config is empty")
	}

	var err error
	m.curServiceConfig = serviceConfig
	m.serviceInfo, err = configinfo.NewServiceInfoFromServiceConfig(serviceConfig, serviceConfig.Id, m.envoyConfigOptions)
	if err != nil {
		return fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	if m.metadataFetcher != nil {
		attrs, err := m.metadataFetcher.FetchGCPAttributes()
		if err != nil {
			m.Infof("metadata server was not reached, skipping GCP Attributes: %v", err)
		} else {
			m.serviceInfo.GcpAttributes = attrs
		}
	}

	snapshot, err := m.makeSnapshot()
	if err != nil {
		return fmt.Errorf("fail to make a snapshot, %s", err)
	}
	return m.cache.SetSnapshot(context.Background(), m.envoyConfigOptions.Node, snapshot)
}

func (m *ConfigManager) makeSnapshot() (*cache.Snapshot, error) {
	m.Infof("making configuration for api: %v", m.serviceInfo.Name)

	var clusterResources, listenerResources []types.Resource

	gens, err := gen.NewClusterGeneratorsFromOPConfig(m.serviceInfo.ServiceConfig(), m.serviceInfo.Options)
	if err != nil {
		return nil, err
	}
	clusters, err := gen.MakeClusters(gens)
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

	snapshot, err := cache.NewSnapshot(m.curConfigId(), map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: listenerResources,
		rsrc.ClusterType:  clusterResources,
	})
	if err != nil {
		return nil, err
	}
	m.Infof("Envoy Dynamic Configuration is cached for service: %v", m.serviceName)
	return snapshot, nil
}

func (m *ConfigManager) curConfigId() string {
	if m.curServiceConfig == nil {
		return ""
	}
	return m.curServiceConfig.Id
}

func (m *ConfigManager) ID(node *corepb.Node) string {
	return node.GetId()
}

// Infof implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) {
	glog.Infof(format, args...)
}

// Debugf implements the Debugf method for Log interface.
func (m *ConfigManager) Debugf(format string, args ...interface{}) {
	glog.Infof(format, args...)
}

// Warnf implements the Warnf method for Log interface.
func (m *ConfigManager) Warnf(format string, args ...interface{}) {
	glog.Infof(format, args...)
}

// Errorf implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

// Cache returns snapshot cache.
func (m *ConfigManager) Cache() cache.Cache { return m.cache }

func httpsClient(opts options.ConfigGeneratorOptions) (*http.Client, error) {
	caCert, err := ioutil.ReadFile(opts.SslSidestreamClientRootCertsPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
		Timeout: opts.HttpRequestTimeout,
	}, nil
}
