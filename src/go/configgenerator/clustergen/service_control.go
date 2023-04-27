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

	helpers2 "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// ServiceControlClusterName is the name of the Service Control v1 xDS cluster.
	ServiceControlClusterName = "service-control-cluster"
)

// ServiceControlCluster is an Envoy cluster to communicate with the remote
// Service Control v1 server.
type ServiceControlCluster struct {
	ServiceControlURI url.URL

	DNS *helpers2.ClusterDNSConfiger
	TLS *helpers2.ClusterTLSConfiger
}

// NewServiceControlClusterFromServiceConfig creates a ServiceControlCluster from
// OP service config + descriptor + ESPv2 options.
func NewServiceControlClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*ServiceControlCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *ServiceControlCluster) GetName() string {
	return ServiceControlClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *ServiceControlCluster) GenConfig() (*clusterpb.Cluster, error) {
	port, err := strconv.Atoi(c.ServiceControlURI.Port())
	if err != nil {
		return nil, fmt.Errorf("failed to parse port from url %+v: %v", c.ServiceControlURI, err)
	}

	connectTimeoutProto := durationpb.New(5 * time.Second)
	config := &clusterpb.Cluster{
		Name:                 c.GetName(),
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(c.ServiceControlURI.Hostname(), uint32(port)),
	}

	if c.ServiceControlURI.Scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(c.ServiceControlURI.Hostname(), nil)
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
