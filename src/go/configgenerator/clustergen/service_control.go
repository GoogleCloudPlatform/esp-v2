package clustergen

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	ServiceControlClusterName = "service-control-cluster"
)

type ServiceControlCluster struct {
	ServiceControlURI url.URL

	DNS *helpers.ClusterDNSConfiger
	TLS *helpers.ClusterTLSConfiger
}

func (c *ServiceControlCluster) GetName() string {
	return ServiceControlClusterName
}

func (c *ServiceControlCluster) GenConfig() (*clusterpb.Cluster, error) {
	port, err := strconv.Atoi(c.ServiceControlURI.Port())
	if err != nil {
		return nil, fmt.Errorf("failed to parse port from url %+v: %v", c.ServiceControlURI, err)
	}

	connectTimeoutProto := durationpb.New(5 * time.Second)
	config := &clusterpb.Cluster{
		Name:                 c.GetName(),
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(c.ServiceControlURI.Hostname(), uint32(port)),
	}

	if c.ServiceControlURI.Scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(c.ServiceControlURI.Hostname(), nil)
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
