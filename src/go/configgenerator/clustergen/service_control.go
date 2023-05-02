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
	"net/url"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// ServiceControlClusterName is the name of the Service Control v1 xDS cluster.
	ServiceControlClusterName = "service-control-cluster"
)

// ServiceControlCluster is an Envoy cluster to communicate with the remote
// Service Control v1 server.
type ServiceControlCluster struct {
	ServiceControlURL url.URL

	DNS *helpers.ClusterDNSConfiger
	TLS *helpers.ClusterTLSConfiger
}

// NewServiceControlClustersFromOPConfig creates a ServiceControlCluster from
// OP service config + descriptor + ESPv2 options. It is a ClusterGeneratorOPFactory.
func NewServiceControlClustersFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error) {
	scURL, err := helpers.ParseServiceControlURLFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("ParseServiceControlURLFromOPConfig got error: %v", err)
	}

	return []ClusterGenerator{
		&ServiceControlCluster{
			ServiceControlURL: scURL,
			DNS:               helpers.NewClusterDNSConfigerFromOPConfig(opts),
			TLS:               helpers.NewClusterTLSConfigerFromOPConfig(opts, false),
		},
	}, nil
}

// GetName implements the ClusterGenerator interface.
func (c *ServiceControlCluster) GetName() string {
	return ServiceControlClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *ServiceControlCluster) GenConfig() (*clusterpb.Cluster, error) {
	port, err := strconv.Atoi(c.ServiceControlURL.Port())
	if err != nil {
		return nil, fmt.Errorf("failed to parse port from url %+v: %v", c.ServiceControlURL, err)
	}

	connectTimeoutProto := durationpb.New(5 * time.Second)
	config := &clusterpb.Cluster{
		Name:                 c.GetName(),
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(c.ServiceControlURL.Hostname(), uint32(port)),
	}

	if c.ServiceControlURL.Scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(c.ServiceControlURL.Hostname(), nil)
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
