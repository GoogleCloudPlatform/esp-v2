package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// LocalBackendCluster is an Envoy cluster to communicate with a local backend
// that speaks HTTP (OpenAPI) or gRPC (service config) protocol.
type LocalBackendCluster struct {
	BackendCluster *helpers.BackendCluster
	GRPCHealth     *helpers.ClusterGRPCHealthCheckConfiger
}

// NewLocalBackendClusterFromServiceConfig creates a LocalBackendCluster from
// OP service config + descriptor + ESPv2 options.
func NewLocalBackendClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*LocalBackendCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *LocalBackendCluster) GetName() string {
	return c.BackendCluster.GetName()
}

// GenConfig implements the ClusterGenerator interface.
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
