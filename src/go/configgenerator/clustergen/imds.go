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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// MetadataServerClusterName is the name of the IMDS xDS cluster.
	MetadataServerClusterName = "metadata-cluster"
)

// IMDSCluster is an Envoy cluster to communicate with the GCP Compute Engine
// Instance Metadata Service. This is primarily used to generate access tokens
// and ID tokens.
type IMDSCluster struct {
	MetadataURL           string
	ClusterConnectTimeout time.Duration

	DNS *helpers.ClusterDNSConfiger
	TLS *helpers.ClusterTLSConfiger // Not sure why IMDS would ever be TLS...
}

// NewIMDSClustersFromOPConfig creates a IMDSCluster from
// OP service config + descriptor + ESPv2 options. It is a ClusterGeneratorOPFactory.
func NewIMDSClustersFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error) {
	if opts.NonGCP {
		return nil, nil
	}

	return []ClusterGenerator{
		&IMDSCluster{
			MetadataURL:           opts.MetadataURL,
			ClusterConnectTimeout: opts.ClusterConnectTimeout,
			DNS:                   helpers.NewClusterDNSConfigerFromOPConfig(opts),
			TLS:                   helpers.NewClusterTLSConfigerFromOPConfig(opts, false),
		},
	}, nil
}

// GetName implements the ClusterGenerator interface.
func (c *IMDSCluster) GetName() string {
	return MetadataServerClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *IMDSCluster) GenConfig() (*clusterpb.Cluster, error) {
	scheme, hostname, port, _, err := util.ParseURI(c.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("fail to parse metadata cluster URI: %v", err)
	}

	connectTimeoutProto := durationpb.New(c.ClusterConnectTimeout)
	config := &clusterpb.Cluster{
		Name:           c.GetName(),
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(hostname, nil)
		if err != nil {
			return nil, err
		}
		config.TransportSocket = transportSocket
	}

	if err := helpers.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
