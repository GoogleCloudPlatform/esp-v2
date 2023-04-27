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
	"fmt"
	"time"

	helpers2 "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

// JWTProviderCluster is an Envoy cluster to communicate with a remote JWKS
// provider. Each cluster talks to one remote server.
type JWTProviderCluster struct {
	Provider              *scpb.AuthProvider
	ClusterConnectTimeout time.Duration

	DNS *helpers2.ClusterDNSConfiger
	TLS *helpers2.ClusterTLSConfiger
}

// NewJWTProviderClustersFromServiceConfig creates all JWTProviderCluster from
// OP service config + descriptor + ESPv2 options.
//
// Generates multiple clusters, one per each JWT provider address.
// Automatically de-duplicates addresses.
func NewJWTProviderClustersFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) ([]*JWTProviderCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *JWTProviderCluster) GetName() string {
	return c.Provider.Id
}

// GenConfig implements the ClusterGenerator interface.
func (c *JWTProviderCluster) GenConfig() (*clusterpb.Cluster, error) {
	jwksUri := c.Provider.GetJwksUri()
	addr, err := util.ExtractAddressFromURI(jwksUri)
	if err != nil {
		return nil, fmt.Errorf("failed to extract address from JWKS URI: %v", err)
	}

	clusterName := fmt.Sprintf("jwt-provider-cluster-%s", addr)

	scheme, hostname, port, _, err := util.ParseURI(jwksUri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS URI: %v", err)
	}

	connectTimeoutProto := durationpb.New(c.ClusterConnectTimeout)

	config := &clusterpb.Cluster{
		Name:           clusterName,
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		// Note: It may not be V4.
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(hostname, port),
	}
	if scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(hostname, nil)
		if err != nil {
			return nil, err
		}
		config.TransportSocket = transportSocket
	}

	if err := helpers2.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
