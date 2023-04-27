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

package helpers

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

// ClusterGRPCHealthCheckConfiger is a helper to set gRPC backend health check config on a cluster.
type ClusterGRPCHealthCheckConfiger struct {
	ServiceName       string
	Interval          time.Duration
	NoTrafficInterval time.Duration
}

// NewClusterGRPCHealthCheckConfigerFromOPConfig creates a ClusterGRPCHealthCheckConfiger from
// OP service config + descriptor + ESPv2 options.
func NewClusterGRPCHealthCheckConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *ClusterGRPCHealthCheckConfiger {
	if !opts.HealthCheckGrpcBackend {
		return nil
	}

	return &ClusterGRPCHealthCheckConfiger{
		ServiceName:       opts.HealthCheckGrpcBackendService,
		Interval:          opts.HealthCheckGrpcBackendInterval,
		NoTrafficInterval: opts.HealthCheckGrpcBackendNoTrafficInterval,
	}
}

// MaybeAddGRPCHealthCheck adds the generated backend gRPC health check config
// to the cluster.
func MaybeAddGRPCHealthCheck(grpcHealthChecker *ClusterGRPCHealthCheckConfiger, cluster *clusterpb.Cluster) error {
	if grpcHealthChecker == nil {
		return nil
	}

	healthChecks, err := grpcHealthChecker.MakeHealthConfig()
	if err != nil {
		return fmt.Errorf("fail to create gRPC health checks for cluster: %v", err)
	}

	cluster.HealthChecks = healthChecks
	return nil
}

// MakeHealthConfig creates a HealthCheck with gRPC backend health config for a cluster.
func (c *ClusterGRPCHealthCheckConfiger) MakeHealthConfig() ([]*corepb.HealthCheck, error) {
	intervalProto := durationpb.New(c.Interval)
	return []*corepb.HealthCheck{
		{
			// Set the timeout as Interval
			Timeout:            intervalProto,
			Interval:           intervalProto,
			NoTrafficInterval:  durationpb.New(c.NoTrafficInterval),
			UnhealthyThreshold: &wrappers.UInt32Value{Value: 3},
			HealthyThreshold:   &wrappers.UInt32Value{Value: 3},
			HealthChecker: &corepb.HealthCheck_GrpcHealthCheck_{
				GrpcHealthCheck: &corepb.HealthCheck_GrpcHealthCheck{
					ServiceName: c.ServiceName,
				},
			},
		},
	}, nil
}
