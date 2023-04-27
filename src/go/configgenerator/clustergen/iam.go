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
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// IAMServerClusterName is the name of the IAM xDS cluster.
	IAMServerClusterName = "iam-cluster"
)

// IAMCluster is an Envoy cluster to communicate with the GCP Cloud IAM.
// This is primarily used to generate access tokens and ID tokens for API Gateway
// use case.
type IAMCluster struct {
	IamURL                string
	ClusterConnectTimeout time.Duration

	DNS *helpers2.ClusterDNSConfiger
	TLS *helpers2.ClusterTLSConfiger
}

// NewIAMClusterFromServiceConfig creates a IAMCluster from
// OP service config + descriptor + ESPv2 options.
func NewIAMClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*IAMCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *IAMCluster) GetName() string {
	return IAMServerClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *IAMCluster) GenConfig() (*clusterpb.Cluster, error) {
	scheme, hostname, port, _, err := util.ParseURI(c.IamURL)
	if err != nil {
		return nil, fmt.Errorf("fail to parse IAM cluster URI: %v", err)
	}

	connectTimeoutProto := durationpb.New(c.ClusterConnectTimeout)
	config := &clusterpb.Cluster{
		Name:            c.GetName(),
		LbPolicy:        clusterpb.Cluster_ROUND_ROBIN,
		DnsLookupFamily: clusterpb.Cluster_V4_ONLY,
		ConnectTimeout:  connectTimeoutProto,
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

	if err := helpers2.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
