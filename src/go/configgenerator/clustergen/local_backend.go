// Copyright 2023 Google LLC
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

package clustergen

import (
	helpers2 "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// LocalBackendCluster is an Envoy cluster to communicate with a local backend
// that speaks HTTP (OpenAPI) or gRPC (service config) protocol.
type LocalBackendCluster struct {
	BackendCluster *helpers2.BaseBackendCluster
	GRPCHealth     *helpers2.ClusterGRPCHealthCheckConfiger
}

// NewLocalBackendClusterFromServiceConfig creates a LocalBackendCluster from
// OP service config + descriptor + ESPv2 options.
func NewLocalBackendClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*LocalBackendCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *LocalBackendCluster) GetName() string {
	return c.BackendCluster.ClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *LocalBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	config, err := c.BackendCluster.GenBaseConfig()
	if err != nil {
		return nil, err
	}

	if err := helpers2.MaybeAddGRPCHealthCheck(c.GRPCHealth, config); err != nil {
		return nil, err
	}

	return config, nil
}
