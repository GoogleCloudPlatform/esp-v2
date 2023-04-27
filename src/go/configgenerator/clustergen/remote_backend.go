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
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// RemoteBackendCluster is an Envoy cluster to communicate with remote backends
// via dynamic routing. Primarily for API Gateway use case.
type RemoteBackendCluster struct {
	BackendCluster *helpers.BaseBackendCluster
}

// NewRemoteBackendClustersFromServiceConfig creates all RemoteBackendCluster from
// OP service config + descriptor + ESPv2 options.
//
// Generates multiple clusters, 1 per remote backend.
func NewRemoteBackendClustersFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*[]RemoteBackendCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GetName() string {
	return c.BackendCluster.ClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	return c.BackendCluster.GenBaseConfig()
}
