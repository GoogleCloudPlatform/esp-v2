package helpers

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

type BackendCluster struct {
	ClusterName string
	Hostname    string
	Port        uint32
	UseTLS      bool
	Protocol    util.BackendProtocol

	ClusterConnectTimeout  time.Duration
	MaxRequestsThreshold   int
	BackendDnsLookupFamily string

	DNS *ClusterDNSConfiger
	TLS *ClusterTLSConfiger
}

func (c *BackendCluster) GetName() string {
	return c.ClusterName
}

func (c *BackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	config := &clusterpb.Cluster{
		Name:                 c.ClusterName,
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       durationpb.New(c.ClusterConnectTimeout),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(c.Hostname, c.Port),
	}

	if c.MaxRequestsThreshold > 0 {
		config.CircuitBreakers = &clusterpb.CircuitBreakers{
			Thresholds: []*clusterpb.CircuitBreakers_Thresholds{
				makeCircuitBreakersThresholds(corepb.RoutingPriority_DEFAULT, c.MaxRequestsThreshold),
				makeCircuitBreakersThresholds(corepb.RoutingPriority_HIGH, c.MaxRequestsThreshold),
			},
		}
	}

	isHttp2 := c.Protocol == util.GRPC || c.Protocol == util.HTTP2

	if c.UseTLS {
		var alpnProtocols []string
		if isHttp2 {
			alpnProtocols = []string{"h2"}
		}
		transportSocket, err := c.TLS.MakeTLSConfig(c.Hostname, alpnProtocols)
		if err != nil {
			return nil, err
		}
		config.TransportSocket = transportSocket
	}

	if isHttp2 {
		config.TypedExtensionProtocolOptions = util.CreateUpstreamProtocolOptions()
	}

	switch c.BackendDnsLookupFamily {
	case "auto":
		config.DnsLookupFamily = clusterpb.Cluster_AUTO
	case "v4only":
		config.DnsLookupFamily = clusterpb.Cluster_V4_ONLY
	case "v6only":
		config.DnsLookupFamily = clusterpb.Cluster_V6_ONLY
	case "v4preferred":
		config.DnsLookupFamily = clusterpb.Cluster_V4_PREFERRED
	case "all":
		config.DnsLookupFamily = clusterpb.Cluster_ALL
	default:
		return nil, fmt.Errorf("invalid DnsLookupFamily: %s; Only auto, v4only, v6only, v4preferred, and all are valid", c.BackendDnsLookupFamily)
	}

	if err := MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}

func makeCircuitBreakersThresholds(prio corepb.RoutingPriority, maxRequests int) *clusterpb.CircuitBreakers_Thresholds {
	return &clusterpb.CircuitBreakers_Thresholds{
		Priority: prio,
		MaxRequests: &wrappers.UInt32Value{
			Value: uint32(maxRequests),
		},
	}
}
