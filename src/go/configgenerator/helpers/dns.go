// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

var (
	// DNSDefaultPort is the default port for DNS protocol.
	DNSDefaultPort = "53"
)

// ClusterDNSConfiger is a helper to set DNS addresses on a cluster.
type ClusterDNSConfiger struct {
	Address string
}

// NewClusterDNSConfigerFromOPConfig creates a ClusterTLSConfiger from
// OP service config + descriptor + ESPv2 options.
func NewClusterDNSConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *ClusterDNSConfiger {
	if opts.DnsResolverAddresses == "" {
		return nil
	}

	return &ClusterDNSConfiger{
		Address: opts.DnsResolverAddresses,
	}
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

// MakeResolversConfig creates an address with DNS config for a cluster.
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
