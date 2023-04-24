package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// LocalHTTPBackendCluster is an Envoy cluster to communicate with a local backend
// that speaks internal HTTP protocol (not OpenAPI).
type LocalHTTPBackendCluster struct {
	BackendCluster *helpers.BackendCluster
}

// NewLocalHTTPBackendClusterFromServiceConfig creates a LocalHTTPBackendCluster from
// OP service config + descriptor + ESPv2 options.
func NewLocalHTTPBackendClusterFromServiceConfig(serviceConfig *scpb.Service, opts options.ConfigGeneratorOptions) (*LocalHTTPBackendCluster, error) {
	// TODO(nareddyt)
	return nil, nil
}

// GetName implements the ClusterGenerator interface.
func (c *LocalHTTPBackendCluster) GetName() string {
	return c.BackendCluster.GetName()
}

// GenConfig implements the ClusterGenerator interface.
func (c *LocalHTTPBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	return c.BackendCluster.GenConfig()
}
