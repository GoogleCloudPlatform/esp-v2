// Copyright 2020 Google LLC
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

package util

import (
	"fmt"
	"strconv"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func DnsResolvers(dnsResolverAddresses string) ([]*corepb.Address, error) {
	var dnsResolvers []*corepb.Address
	addressSlice := strings.Split(dnsResolverAddresses, ";")
	for _, DnsResolverAddress := range addressSlice {
		host, port, err := parseDnsResolverAddress(DnsResolverAddress)
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

func parseDnsResolverAddress(address string) (string, uint32, error) {
	arr := strings.Split(address, ":")
	if len(arr) == 0 || len(arr) > 2 {
		return "", 0, fmt.Errorf("address has a more than one column: %s", address)
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
