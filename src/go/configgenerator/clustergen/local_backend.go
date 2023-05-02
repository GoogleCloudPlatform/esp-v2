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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// LocalBackendCluster is an Envoy cluster to communicate with a local backend
// that speaks HTTP (OpenAPI) or gRPC (service config) protocol.
type LocalBackendCluster struct {
	BackendCluster *helpers.BaseBackendCluster
	GRPCHealth     *helpers.ClusterGRPCHealthCheckConfiger
}

// NewLocalBackendClustersFromOPConfig creates a LocalBackendCluster from
// OP service config + descriptor + ESPv2 options. It is a ClusterGeneratorOPFactory.
func NewLocalBackendClustersFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error) {
	scheme, hostname, port, _, err := util.ParseURI(opts.BackendAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing uri: %v", err)
	}

	// For local backend, user cannot configure http protocol explicitly.
	// If this is for test only http backend address, use http/1.1 by default.
	protocol, useTLS, err := util.ParseBackendProtocol(scheme, "")
	if err != nil {
		return nil, fmt.Errorf("error parsing local backend protocol: %v", err)
	}

	if opts.HealthCheckGrpcBackend {
		if protocol != util.GRPC {
			return nil, fmt.Errorf("invalid flag --health_check_grpc_backend, backend protocol must be GRPC")
		}
	}

	var tls *helpers.ClusterTLSConfiger
	if useTLS {
		tls = helpers.NewClusterTLSConfigerFromOPConfig(opts, true)
	}

	return []ClusterGenerator{
		&LocalBackendCluster{
			BackendCluster: &helpers.BaseBackendCluster{
				ClusterName:            fmt.Sprintf("backend-cluster-%s_local", serviceConfig.GetName()),
				Hostname:               hostname,
				Port:                   port,
				Protocol:               protocol,
				ClusterConnectTimeout:  opts.ClusterConnectTimeout,
				MaxRequestsThreshold:   opts.BackendClusterMaxRequests,
				BackendDnsLookupFamily: opts.BackendDnsLookupFamily,
				DNS:                    helpers.NewClusterDNSConfigerFromOPConfig(opts),
				TLS:                    tls,
			},
			GRPCHealth: helpers.NewClusterGRPCHealthCheckConfigerFromOPConfig(opts),
		},
	}, nil
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

	if err := helpers.MaybeAddGRPCHealthCheck(c.GRPCHealth, config); err != nil {
		return nil, err
	}

	return config, nil
}
