package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

type LocalBackendCluster struct {
	BackendCluster *helpers.BackendCluster
	GRPCHealth     *helpers.ClusterGRPCHealthCheckConfiger
}

func (c *LocalBackendCluster) GetName() string {
	return c.BackendCluster.GetName()
}

func (c *LocalBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	config, err := c.BackendCluster.GenConfig()
	if err != nil {
		return nil, err
	}

	if err := helpers.MaybeAddGRPCHealthCheck(c.GRPCHealth, config); err != nil {
		return nil, err
	}

	return config, nil
}
