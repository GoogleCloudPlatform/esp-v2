package clustergen

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

type IAMCluster struct {
	IamURL                string
	ClusterConnectTimeout time.Duration

	DNS *helpers.ClusterDNSConfiger
	TLS *helpers.ClusterTLSConfiger
}

var (
	IamServerClusterName = "iam-cluster"
)

func (c *IAMCluster) GetName() string {
	return IamServerClusterName
}

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

	if err := helpers.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
