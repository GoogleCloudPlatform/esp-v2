package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// RemoteBackendCluster is an Envoy cluster to communicate with remote backends
// via dynamic routing. Primarily for API Gateway use case.
type RemoteBackendCluster struct {
	BackendCluster *helpers.BackendCluster
}

// NewRemoteBackendClustersFromServiceConfig creates all RemoteBackendCluster from
// OP service config + descriptor + ESPv2 options.
//
// Generates multiple clusters, 1 per remote backend.
func NewRemoteBackendClustersFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*[]RemoteBackendCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GetName() string {
	return c.BackendCluster.GetName()
}

// GenConfig implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	return c.BackendCluster.GenConfig()
}
