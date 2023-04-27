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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// TokenAgentClusterName is the name of the token agent xDS cluster.
	TokenAgentClusterName = "token-agent-cluster"
)

// TokenAgentCluster is an Envoy cluster to communicate with the localhost golang
// token agent.
type TokenAgentCluster struct {
	ClusterConnectTimeout time.Duration
	TokenAgentPort        int

	DNS *helpers.ClusterDNSConfiger
}

// NewTokenAgentClusterFromServiceConfig creates a TokenAgentCluster from
// OP service config + descriptor + ESPv2 options.
func NewTokenAgentClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*TokenAgentCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *TokenAgentCluster) GetName() string {
	return TokenAgentClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *TokenAgentCluster) GenConfig() (*clusterpb.Cluster, error) {
	config := &clusterpb.Cluster{
		Name:           c.GetName(),
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: durationpb.New(c.ClusterConnectTimeout),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STATIC,
		},
		LoadAssignment: util.CreateLoadAssignment(util.LoopbackIPv4Addr, uint32(c.TokenAgentPort)),
	}

	if err := helpers.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
