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

package configgenerator

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// GetESPv2ClusterGenFactories returns the enabled ClusterGenerators for
// ESPv2.
func GetESPv2ClusterGenFactories() []clustergen.ClusterGeneratorOPFactory {
	return []clustergen.ClusterGeneratorOPFactory{
		clustergen.NewLocalBackendClustersFromOPConfig,
		clustergen.NewTokenAgentClustersFromOPConfig,
		clustergen.NewIMDSClustersFromOPConfig,
		clustergen.NewIAMClustersFromOPConfig,
		clustergen.NewServiceControlClustersFromOPConfig,
		clustergen.NewRemoteBackendClustersFromOPConfig,
		clustergen.NewJWTProviderClustersFromOPConfig,
	}
}

// NewClusterGeneratorsFromOPConfig creates all required ClusterGenerators from
// OP service config + descriptor + ESPv2 options.
func NewClusterGeneratorsFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, factories []clustergen.ClusterGeneratorOPFactory) ([]clustergen.ClusterGenerator, error) {
	var gens []clustergen.ClusterGenerator
	for _, factory := range factories {
		generator, err := factory(serviceConfig, opts)
		if err != nil {
			return nil, fmt.Errorf("fail to run ClusterGeneratorOPFactory: %v", err)
		}
		gens = append(gens, generator...)
	}

	for i, gen := range gens {
		glog.Infof("ClusterGenerator %d is %q", i, gen.GetName())
	}
	return gens, nil
}

// MakeClusters creates the xDS cluster configs from the ClusterGenerators.
func MakeClusters(gens []clustergen.ClusterGenerator) ([]*clusterpb.Cluster, error) {
	var clusters []*clusterpb.Cluster
	for _, gen := range gens {
		cluster, err := gen.GenConfig()
		if err != nil {
			return nil, fmt.Errorf("cluster generator %q failed to generate xDS cluster config: %v", gen.GetName(), err)
		}
		clusters = append(clusters, cluster)
	}

	glog.Infof("generate clusters: %v", clusters)
	return clusters, nil
}
