package helpers

import (
	"fmt"
	"strconv"
	"strings"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

var (
	// DNSDefaultPort is the default port for DNS protocol.
	DNSDefaultPort = "53"
)

type ClusterDNSConfiger struct {
	Address string
}

// MaybeAddDNSResolver adds the generated DNS resolvers config to the given cluster.
func MaybeAddDNSResolver(dnsConfiger *ClusterDNSConfiger, cluster *clusterpb.Cluster) error {
	if dnsConfiger == nil {
		return nil
	}

	resolvers, err := dnsConfiger.MakeResolversConfig()
	if err != nil {
		return fmt.Errorf("fail to create DNS resolver for cluster: %v", err)
	}

	cluster.DnsResolvers = resolvers
	return nil
}

func (c *ClusterDNSConfiger) MakeResolversConfig() ([]*corepb.Address, error) {
	var dnsResolvers []*corepb.Address
	addressSlice := strings.Split(c.Address, ";")
	for _, address := range addressSlice {
		host, port, err := parseAddress(address)
		if err != nil {
			return nil, fmt.Errorf("fail to parse dnsResolverAddress: %v", err)
		}

		dnsResolvers = append(dnsResolvers, &corepb.Address{
			Address: &corepb.Address_SocketAddress{
				SocketAddress: &corepb.SocketAddress{
					Address: host,
					PortSpecifier: &corepb.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		})
	}

	return dnsResolvers, nil
}

func parseAddress(address string) (string, uint32, error) {
	arr := strings.Split(address, ":")
	if len(arr) == 0 || len(arr) > 2 {
		return "", 0, fmt.Errorf("address has a more than one colon: %s", address)
	}

	if len(arr) == 1 {
		arr = append(arr, DNSDefaultPort)
	}

	portVal, err := strconv.Atoi(arr[1])
	if err != nil {
		return "", 0, err
	}

	return arr[0], uint32(portVal), nil
}
