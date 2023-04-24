package clustergen

import (
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	TokenAgentClusterName = "token-agent-cluster"
)

type TokenAgentCluster struct {
	ClusterConnectTimeout time.Duration
	TokenAgentPort        int

	DNS *helpers.ClusterDNSConfiger
}

func (c *TokenAgentCluster) GetName() string {
	return TokenAgentClusterName
}

func (c *TokenAgentCluster) GenConfig() (*clusterpb.Cluster, error) {
	config := &clusterpb.Cluster{
		Name:           c.GetName(),
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: durationpb.New(c.ClusterConnectTimeout),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STATIC,
		},
		LoadAssignment: util.CreateLoadAssignment(util.LoopbackIPv4Addr, uint32(c.TokenAgentPort)),
	}

	if err := helpers.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
