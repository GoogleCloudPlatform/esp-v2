package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

type LocalHTTPBackendCluster struct {
	BackendCluster *helpers.BackendCluster
}

func (c *LocalHTTPBackendCluster) GetName() string {
	return c.BackendCluster.GetName()
}

func (c *LocalHTTPBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	return c.BackendCluster.GenConfig()
}
