package helpers

import (
	"fmt"
	"time"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

type ClusterGRPCHealthCheckConfiger struct {
	ServiceName       string
	Interval          time.Duration
	NoTrafficInterval time.Duration
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
